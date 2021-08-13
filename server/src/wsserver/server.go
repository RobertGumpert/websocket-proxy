package wsserver

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
	"ws-server/src/model"
)

type Server struct {
	poolRetryMessages []*retryMessageWrapper
	clients           *poolClients
	//
	maxPoolClientsSize        int
	maxCountMessagesPerSecond int
	maxPoolRetryMessagesSize  int
}

func NewServer(maxPoolClientsSize int, maxCountMessagesPerSecond int, maxPoolRetryMessagesSize int) *Server {
	this := new(Server)
	this.maxCountMessagesPerSecond = maxCountMessagesPerSecond
	this.maxPoolClientsSize = maxPoolClientsSize
	this.maxPoolRetryMessagesSize = maxPoolRetryMessagesSize
	this.poolRetryMessages = make([]*retryMessageWrapper, 0)
	this.clients = NewPoolClients()
	//
	go this.countNumberReceivedMessagesPerSecond()
	go this.retrySendingMessages()
	//
	return this
}

func (this *Server) NewClient(w http.ResponseWriter, r *http.Request) error {
	if this.clients.Size() >= this.maxPoolClientsSize {
		return errors.New("Pool clients is filled")
	}
	wrapper, err := NewClient(w, r)
	if err != nil {
		return err
	}
	this.clients.Push(wrapper)
	wrapper.StartServing(this.Redirect)
	return nil
}

func (this *Server) Redirect(msg model.WSMessage) {
	var (
		retryMessage *retryMessageWrapper
	)
	log.Println(
		fmt.Sprintf(
			" ---> Server: receive message [%s] from remote client [%s] and start redirect to remote clients",
			string(msg.Content),
			msg.SenderAddress,
		),
	)
	for tuple := range this.clients.Iterate() {
		if tuple.ClientWrapper == nil {
			continue
		}
		if tuple.Address == msg.SenderAddress {
			tuple.ClientWrapper.CountMessages++
			continue
		}
		if !tuple.ClientWrapper.Client.ConnectionIsClosed() {
			err := tuple.ClientWrapper.Client.Send(msg)
			if err != nil {
				if retryMessage == nil {
					retryMessage = &retryMessageWrapper{
						Message:        msg,
						ClientCounters: make(map[model.RemoteAddress]retryCounter),
					}
				}
				retryMessage.ClientCounters[tuple.Address] = retryCounter(1)
			} else {
				log.Println(
					fmt.Sprintf(
						" ---> Server: send message [%s] to remote client [%s]",
						string(msg.Content),
						tuple.Address,
					),
				)
			}
		}
	}
	if retryMessage != nil {
		this.poolRetryMessages = append(
			this.poolRetryMessages,
			retryMessage,
		)
	}
}

func (this *Server) countNumberReceivedMessagesPerSecond() {
	var (
		ticker = time.NewTicker(time.Second)
	)
	for {
		select {
		case <-ticker.C:
			for tuple := range this.clients.Iterate() {
				if tuple.ClientWrapper.CountMessages >= this.maxCountMessagesPerSecond {
					tuple.ClientWrapper.Client.Close()
					log.Println(
						fmt.Sprintf(
							" ---> Server: The client [%s] has exceeded the number of sent messages per second",
							tuple.Address,
						),
					)
					this.clients.Pop(tuple.Address)
				} else {
					tuple.ClientWrapper.CountMessages = 0
				}
			}
		}
	}
}

func (this *Server) retrySendingMessages() {
	var (
		ticker = time.NewTicker(time.Second)
		del = func (index int)  {
			var(
				temp = make([]*retryMessageWrapper, 0)
			)
			for i := 0 ; i < len(this.poolRetryMessages); i++ {
				if i == index {
					continue
				}
				temp = append(
					temp, 
					this.poolRetryMessages[i],
				)
			}
			this.poolRetryMessages = temp
		}
	)
	for {
		select {
		case <-ticker.C:
			for index, retryMessage := range this.poolRetryMessages {
				if len(retryMessage.ClientCounters) == 0 {
					del(index)
				}
				for clientAddress, retryCounter := range retryMessage.ClientCounters {
					item, exist := this.clients.Get(clientAddress)
					if exist {
						if item.ClientWrapper != nil {
							err := item.ClientWrapper.Client.Send(retryMessage.Message)
							if err != nil {
								retryCounter++
								continue
							} else {
								log.Println(
									fmt.Sprintf(
										" ---> Server: send retry message [%s] to remote client [%s]",
										string(retryMessage.Message.Content),
										clientAddress,
									),
								)
							}
						}
					}
					delete(retryMessage.ClientCounters, clientAddress)
				}
			}
		}
	}
}
