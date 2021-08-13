package wsclient

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"ws-server/src/model"
	"github.com/gorilla/websocket"
)

type CallbackOnReceive func(msg model.WSMessage)

type Client struct {
	connection                *websocket.Conn
	eventOnCloseConnection    chan struct{}
	connectionIsClosed        bool
	mx                        *sync.Mutex
	Address                   model.RemoteAddress
}

func NewClient(w http.ResponseWriter, r *http.Request) (*Client, error) {
	upgrader := &websocket.Upgrader{}
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	address, err := model.NewRemoteAddress(connection.RemoteAddr().String())
	if err != nil {
		connection.Close()
		return nil, err
	}
	this := &Client{
		connection:                connection,
		Address:                   address,
		eventOnCloseConnection:    make(chan struct{}),
		connectionIsClosed:        false,
		mx:                        new(sync.Mutex),
	}
	return this, nil
}

func (this *Client) StartServing(receiveCallback CallbackOnReceive) {
	go this.receiveMessages(receiveCallback)
}

func (this *Client) closeConnection() {
	if !this.connectionIsClosed {
		close(this.eventOnCloseConnection)
		this.connectionIsClosed = true
		if err := this.connection.Close(); err != nil {
			log.Println(
				fmt.Sprintf(
					" ---> Server: Connection with client [%s] closed with error [%s]",
					this.Address,
					err,
				),
			)
		} else {
			log.Println(
				fmt.Sprintf(
					" ---> Server: Connection with client [%s] closed",
					this.Address,
				),
			)
		}
	}
}

func (this *Client) receiveMessages(callback CallbackOnReceive) {
	defer this.closeConnection()
	for {
		if this.connectionIsClosed {
			return
		}
		msgType, msgBytes, err := this.connection.ReadMessage()
		if err != nil {
			return
		}
		go callback(model.WSMessage{
			Type:    msgType,
			Content: msgBytes,
			Err:     err,
			SenderAddress:  this.Address,
		})
	}
}


func (this *Client) Send(msg model.WSMessage) error {
	if this.connectionIsClosed {
		return nil
	}
	defer this.mx.Unlock()
	this.mx.Lock()
	return this.connection.WriteMessage(
		msg.Type,
		msg.Content,
	)
}

func (this *Client) Close() {
	this.closeConnection()
}

func (this *Client) ConnectionIsClosed() bool {
	return this.connectionIsClosed
}
