package test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/RobertGumpert/wsunit"
)

func chans(mb int, cb int) (receiveChannel chan wsunit.RecievedMessage, closeChannel chan wsunit.ConnectionCloser) {
	return make(chan wsunit.RecievedMessage, mb), make(chan wsunit.ConnectionCloser, cb)
}

type FakeClientWrapper struct {
	Unit           *wsunit.Unit
	Server         *httptest.Server
	URL            *url.URL
	remoteServer   *httptest.Server
	receiveChannel chan wsunit.RecievedMessage
	closeChannel   chan wsunit.ConnectionCloser
	ID             int
}

func NewFakeClientWrapper(remoteServer *httptest.Server, id int) *FakeClientWrapper {
	receiveChannel, closeChannel := chans(100, 1)
	unit := wsunit.NewUnit(receiveChannel, closeChannel)
	unit.TurnOnOffLogs()
	this := &FakeClientWrapper{
		Unit:           unit,
		remoteServer:   remoteServer,
		receiveChannel: receiveChannel,
		closeChannel:   closeChannel,
		ID:             id,
	}
	server := httptest.NewServer(this)
	log.Println(fmt.Sprintf("Client [%d] started with address [%s]", this.ID, server.URL))
	u, err := url.Parse(server.URL)
	if err != nil {
		log.Fatal(err)
	}
	this.URL = u
	this.Server = server
	return this
}

func (this *FakeClientWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ws":
		u, err := url.Parse(this.remoteServer.URL)
		if err != nil {
			log.Fatal(err)
		}
		u.Path = "/ws"
		_, err = this.Unit.AsRemoteServerUnit(u)
		if err != nil {
			log.Fatal(err)
		}
		go this.send()
		go this.receive()
		go this.receiveErrors()
	case "/msg":
		log.Println(
			fmt.Sprintf(
				"Receive API message on client [%d]",
				this.ID,
			),
		)
	}
}

// Для наглядности будем
// отправлять сообщения только с первого клиента (не засорять консоль)
//
func (this *FakeClientWrapper) send() {
	if this.ID == 0 {
		timer := time.NewTicker(time.Duration(5) * time.Second)
		for range timer.C {
			err := this.Unit.CreateMessageAndSend(
				fmt.Sprintf("Message by client [%d]", this.ID),
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (this *FakeClientWrapper) receive() {
	for msg := range this.receiveChannel {
		log.Println(
			fmt.Sprintf(
				" --------> Receive WS message on client [%d], msg: [%s]",
				this.ID,
				string(msg.Content),
			),
		)
	}
}

func (this *FakeClientWrapper) receiveErrors() {
	for err := range this.closeChannel {
		switch err.ErrorType {
		case wsunit.CloseConnection:
			if err := this.Unit.Close(); err != nil {
				log.Println(fmt.Sprintf("---> Client [%d] when trying to close the connection have error [%s]", this.ID, err.Error()))
			} else {
				log.Println(fmt.Sprintf("---> Client [%d] close the connection", this.ID))
			}
		}
	}
}
