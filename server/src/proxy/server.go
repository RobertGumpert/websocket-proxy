package proxy

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"ws-server/src/client"
	"ws-server/src/model"

	"github.com/gorilla/websocket"
)

type Server struct {
	clients            map[string]*client.Client
	maxCountClients    int
	maximumLoad        int
	maximumPoolSize    int
	maximumCountRepeat int
	redirectChannel    chan model.Message
	mx                 *sync.Mutex
}

func NewServer(maxCountClients int, maximumLoad int, maximumPoolSize int, maximumCountRepeat int) *Server {
	this := &Server{
		clients:            make(map[string]*client.Client),
		maxCountClients:    maxCountClients,
		maximumLoad:        maximumLoad,
		maximumPoolSize:    maximumPoolSize,
		maximumCountRepeat: maximumCountRepeat,
		mx:                 new(sync.Mutex),
		redirectChannel:    make(chan model.Message, maxCountClients),
	}
	go this.redirect()
	return this
}

func (this *Server) AddNewClient(w http.ResponseWriter, r *http.Request) error {
	if len(this.clients) == this.maxCountClients {
		return errors.New("")
	}
	upgrader := &websocket.Upgrader{}
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	this.mx.Lock()
	this.clients[connection.RemoteAddr().String()] = client.NewClient(
		connection,
		this.redirectChannel,
		this.maximumLoad,
		this.maximumPoolSize,
		this.maximumCountRepeat,
	)
	this.mx.Unlock()
	return nil
}

func (this *Server) redirect() {
	for msg := range this.redirectChannel {
		log.Println(" ----> Server: redirect message ", string(msg.Content))
		for _, cl := range this.clients {
			if msg.Receiver == cl.RemoteAddr {
				continue
			}
			if cl.PossiblySendingMessage() {
				cl.Send(
					msg,
				)
			} else {
				this.mx.Lock()
				delete(this.clients, cl.RemoteAddr)
				this.mx.Unlock()
			}
		}
	}
}
