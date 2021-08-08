package client

import (
	"fmt"
	"log"
	"sync"
	"time"
	"ws-server/src/model"

	"github.com/gorilla/websocket"
)

type message struct {
	countRepeat int
	msg         model.Message
}

type Client struct {
	done                    chan struct{}
	pool                    []*message
	maxCountRepeat          int
	maxSizePool             int
	connection              *websocket.Conn
	possiblySendingMessages bool
	maximumLoad             int
	redirectChannel         chan model.Message
	RemoteAddr              string
	mx                      *sync.Mutex
}

func NewClient(
	connection *websocket.Conn,
	redirectChannel chan model.Message,
	maximumLoad int,
	maxSizePool int,
	maxCountRepeat int,
) *Client {
	this := &Client{
		connection:              connection,
		RemoteAddr:              connection.RemoteAddr().String(),
		possiblySendingMessages: true,
		redirectChannel:         redirectChannel,
		maximumLoad:             maximumLoad,
		pool:                    make([]*message, 0),
		maxSizePool:             maxSizePool,
		done:                    make(chan struct{}),
		maxCountRepeat:          maxCountRepeat,
		mx:                      new(sync.Mutex),
	}
	go this.read()
	go this.send()
	return this
}

func (this *Client) Send(msg model.Message) {
	size := len(this.pool)
	if this.maxSizePool < (size + 1) {
		this.done <- struct{}{}
	} else {
		this.mx.Lock()
		this.pool = append(this.pool, &message{
			msg: msg,
		})
		this.mx.Unlock()
	}
}

func (this *Client) deleteFromPool(index int) {
	tmp := make([]*message, 0)
	for i, msg := range this.pool {
		if i == index {
			continue
		}
		tmp = append(tmp, msg)
	}
	this.mx.Lock()
	this.pool = tmp
	this.mx.Unlock()
}

func (this *Client) send() {
	timer := time.NewTicker(time.Millisecond)
	defer func() {
		if this.possiblySendingMessages {
			err := this.connection.Close()
			if err != nil {
				log.Println("")
			}
			this.possiblySendingMessages = false
		}
	}()
	for {
		select {
		case <-this.done:
			return
		case <-timer.C:
			for index, msg := range this.pool {
				if msg.countRepeat == this.maxCountRepeat {
					this.deleteFromPool(index)
				}
				err := this.connection.WriteMessage(
					msg.msg.Type,
					msg.msg.Content,
				)
				if err != nil {
					log.Println(
						fmt.Sprintf(" ----> Server: Redirect to remote client [%s] send message wiht error [%s].",
							this.RemoteAddr,
							err.Error(),
						),
					)
					msg.countRepeat++
				} else {
					this.deleteFromPool(index)
					log.Println(
						fmt.Sprintf(" ----> Server: redirect to remote client [%s] successfully.",
							this.RemoteAddr,
						),
					)
				}
			}
		}
	}
}

func (this *Client) read() {
	for {
		if !this.possiblySendingMessages {
			return
		}
		msgType, msgBytes, err := this.connection.ReadMessage()
		if len(this.redirectChannel) >= this.maximumLoad {
			log.Println(
				fmt.Sprintf(" ----> Server: client [%s] close connection because buffer is filled", this.RemoteAddr),
			)
			break
		}
		if err != nil {
			log.Println(
				fmt.Sprintf(" ----> Server: client [%s] close connection with error [%s]", this.RemoteAddr, err.Error()),
			)
			break
		}
		go func() {
			this.redirectChannel <- model.Message{
				Receiver: this.RemoteAddr,
				Type:     msgType,
				Content:  msgBytes,
			}
		}()
	}
	if this.possiblySendingMessages {
		err := this.connection.Close()
		if err != nil {
			log.Println("")
		}
		this.possiblySendingMessages = false
	}
}

func (this *Client) PossiblySendingMessage() bool {
	return this.possiblySendingMessages
}
