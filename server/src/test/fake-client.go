package test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type FakeClientWrapper struct {
	backend         *url.URL
	Server          *httptest.Server
	ServerAddr      string
	WSAddr          string
	connection      *websocket.Conn
	ID              int
	done            chan struct{}
	closeConnection chan struct{}
}

func NewFakeClient(id int, backend *url.URL) *FakeClientWrapper {
	this := new(FakeClientWrapper)
	this.Server = httptest.NewServer(this)
	this.ServerAddr = this.Server.URL
	this.backend = backend
	this.ID = id
	this.done = make(chan struct{})
	this.closeConnection = make(chan struct{})
	return this
}

func (this *FakeClientWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ws":
		err := this.ConnectWithServerByWS("/ws")
		if err != nil {
			log.Println(err)
		}
	case "/msg":
		log.Println(
			fmt.Sprintf(
				"Receive API message on client [%d]",
				this.ID,
			),
		)
	}
}

func (this *FakeClientWrapper) ConnectWithServerByWS(endpoint string) error {
	this.backend.Scheme = "ws"
	this.backend.Path = endpoint
	connection, _, err := websocket.DefaultDialer.Dial(this.backend.String(), nil)
	if err != nil {
		return err
	}
	this.connection = connection
	this.WSAddr = connection.LocalAddr().String()
	log.Println(
		fmt.Sprintf(
			"Client [%d] on adress [%s] start ws on adress [%s]",
			this.ID,
			this.ServerAddr,
			this.WSAddr,
		),
	)
	go this.handleWS()
	return nil
}

func (this *FakeClientWrapper) handleWS() {

	defer func() {
		err := this.connection.Close()
		if err != nil {
			log.Println("Fnish :", err)
		}
	}()

	go func() {
		defer close(this.done)
		for {
			_, msg, err := this.connection.ReadMessage()
			if err != nil {
				log.Println(
					fmt.Sprintf(
						" ----> Client [%d] addr [%s] receive message with error [%s]",
						this.ID,
						this.WSAddr,
						err.Error(),
					),
				)
				return
			}
			log.Println(
				fmt.Sprintf(
					" ----> Client [%d] addr [%s] receive message [%s]",
					this.ID,
					this.WSAddr,
					string(msg),
				),
			)
		}
	}()

	timer := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-this.done:
			return
		case <-timer.C:
			if this.ID == 0 {
				err := this.connection.WriteMessage(
					websocket.TextMessage,
					[]byte(fmt.Sprintf(
						"Message from [%s] client",
						this.WSAddr,
					)))
				if err != nil {
					log.Println("write:", err)
					return
				}
			}
		case <-this.closeConnection:
			err := this.connection.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(
					websocket.CloseNormalClosure,
					"goobye",
				),
			)
			if err != nil {
				log.Println("write close:", err)
				return
			}
		}
	}
}
