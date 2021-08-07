package stack

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"ws-server/src/client"
	"github.com/RobertGumpert/wsunit"
)

type RemoteClientsStack struct {
	clients              map[string]*client.RemoteClient
	stackReceiveMessages chan wsunit.Message
	stackErrors          chan wsunit.ConnectionCloser
	mx                   *sync.Mutex
}

func NewStack(sizeMessagesStack int, sizeClientsStack int) *RemoteClientsStack {
	this := &RemoteClientsStack{
		clients:              make(map[string]*client.RemoteClient),
		stackReceiveMessages: make(chan wsunit.Message, sizeMessagesStack),
		stackErrors:          make(chan wsunit.ConnectionCloser, sizeClientsStack),
		mx:                   new(sync.Mutex),
	}
	go this.ReceiveMessages()
	return this
}

func (this *RemoteClientsStack) AddClient(w http.ResponseWriter, r *http.Request) error {
	cl := client.NewRemoteClient(
		this.stackReceiveMessages,
		this.stackErrors,
	)
	err := cl.ConnectWithClient(w, r)
	if err != nil {
		return err
	}
	this.mx.Lock()
	this.clients[cl.Host] = cl
	this.mx.Unlock()
	return nil
}

func (this *RemoteClientsStack) ReceiveMessages() {
	for message := range this.stackReceiveMessages {
		for _, client := range this.clients {
			if message.SenderAddr == client.Host {
				continue
			}
			if err := client.SendToClient(message); err != nil {
				log.Println(fmt.Sprintf("Send to client have error [%s]", err.Error()))
			}
		}
	}
}

func (this *RemoteClientsStack) ReceiveErrors() {
	for err := range this.stackErrors {
		this.mx.Lock()
		cl := this.clients[err.RemoteAddr]
		log.Println(fmt.Sprintf("Receive error [%s] from remote client [%s]", err.Error, cl.Host))
		if err.ErrorType == wsunit.UndefinedError {
			if err := cl.SendCloseMessage(); err != nil {
				log.Println(fmt.Sprintf("Close connect with client [%s] have error [%s]", cl.Host, err.Error()))
			}
		} else {
			if err := cl.CloseConnectWithClient(); err != nil {
				log.Println(fmt.Sprintf("Close connect with client [%s] have error [%s]", cl.Host, err.Error()))
			}
		}
		this.clients[err.RemoteAddr] = nil
		this.mx.Unlock()
	}
}
