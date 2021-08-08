package stack

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"ws-server/src/client"

	"github.com/RobertGumpert/wsunit"
)

type RemoteClientsStack struct {
	clients              map[string]*client.RemoteClient
	stackReceiveMessages chan wsunit.RecievedMessage
	stackErrors          chan wsunit.ConnectionCloser
	sizeMessagesStack    int
	sizeClientsStack     int
	mx                   *sync.Mutex
}

func NewStack(sizeMessagesStack, sizeClientsStack int) *RemoteClientsStack {
	this := &RemoteClientsStack{
		clients:              make(map[string]*client.RemoteClient),
		stackReceiveMessages: make(chan wsunit.RecievedMessage, sizeMessagesStack),
		stackErrors:          make(chan wsunit.ConnectionCloser, sizeClientsStack),
		mx:                   new(sync.Mutex),
		sizeMessagesStack:    sizeMessagesStack,
		sizeClientsStack:     sizeMessagesStack,
	}
	go this.receiveMessages()
	go this.receiveErrors()
	return this
}

func (this *RemoteClientsStack) CloseConnectionWithAllClients() {
	for _, cl := range this.clients {
		err := cl.SendMessageOnClose()
		if err != nil {
			log.Println(err)
		}
	}
}

func (this *RemoteClientsStack) AddClient(w http.ResponseWriter, r *http.Request, wsunitLogs bool) error {
	if len(this.clients) == this.sizeClientsStack {
		return errors.New(fmt.Sprintf("Clients stack is filled [%d]", this.sizeClientsStack))
	}
	cl := client.NewRemoteClient(
		this.stackReceiveMessages,
		this.stackErrors,
		wsunitLogs,
	)
	err := cl.SwitchConnectionHttpToWebsocket(w, r)
	if err != nil {
		return err
	}
	this.mx.Lock()
	this.clients[cl.Host] = cl
	this.mx.Unlock()
	return nil
}

func (this *RemoteClientsStack) receiveMessages() {
	for message := range this.stackReceiveMessages {
		for _, client := range this.clients {
			if message.SenderAddr == client.Host {
				continue
			}
			if err := client.SendMessage(message); err != nil {
				log.Println(fmt.Sprintf("Send to client have error [%s]", err.Error()))
			}
		}
	}
}

func (this *RemoteClientsStack) receiveErrors() {
	for err := range this.stackErrors {
		this.mx.Lock()
		cl := this.clients[err.RemoteAddr]
		if cl == nil {
			continue
		}
		log.Println(fmt.Sprintf("Receive error [%s] from remote client [%s]", err.Error, cl.Host))
		// switch err.ErrorType {
		// case wsunit.UndefinedError:
		// 	if err := cl.SendMessageOnClose(); err != nil {
		// 		log.Println(fmt.Sprintf("Close connect with client [%s] have error [%s]", cl.Host, err.Error()))
		// 	}
		// case wsunit.CloseConnection:
		// 	if err := cl.HardCloseConnection(); err != nil {
		// 		log.Println(fmt.Sprintf("Close connect with client [%s] have error [%s]", cl.Host, err.Error()))
		// 	}
		// }
		this.clients[err.RemoteAddr] = nil
		this.mx.Unlock()
	}
}
