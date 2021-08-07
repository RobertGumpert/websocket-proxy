package test

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
	"wsclient/src/wsunit"

	"github.com/gorilla/websocket"
)

type (
	fakeServer struct {
		URL                     *url.URL
		ws                      websocket.Upgrader
		sendMessageLogicTime    int
		receiveMessageLogicTime int
	}
)

func NewFakeServer() *fakeServer {
	ws := websocket.Upgrader{}
	return &fakeServer{
		ws: ws,
	}
}

func (this *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sender":
		this.send(w, r)
	case "/receiver":
		this.receiver(w, r)
	default:

	}
}

func (this *fakeServer) send(w http.ResponseWriter, r *http.Request) {
	connention, err := this.ws.Upgrade(w, r, nil)
	if err != nil {
		log.Println("fakeServer send: step 'upgrade' have error [", err, "]")
		return
	}
	go func() {
		timer := time.NewTicker(time.Second)
		defer connention.Close()
		for range timer.C {
			this.sendMessageLogicTime++
			msg, err := wsunit.NewMessage(
				websocket.TextMessage,
				[]byte(
					fmt.Sprintf(
						"%s : %d",
						"Message from server",
						this.sendMessageLogicTime,
					),
				),
				r.RemoteAddr, 0, 0,
				this.URL.Host, time.Now().Unix(), this.sendMessageLogicTime,
			).Marshall()
			if err != nil {
				log.Println(err)
				continue
			}
			err = connention.WriteMessage(
				websocket.TextMessage,
				msg,
			)
			if err != nil {
				log.Println("fakeServer send: step send message have error [", err, "]")
				return
			}
		}
	}()
}

func (this *fakeServer) receiver(w http.ResponseWriter, r *http.Request) {
	connention, err := this.ws.Upgrade(w, r, nil)

	if err != nil {
		log.Println("fakeServer receiver: step 'upgrade' have error [", err, "]")
		return
	}
	go func() {
		defer connention.Close()
		for {
			msgType, msgBytes, err := connention.ReadMessage()
			this.receiveMessageLogicTime++
			if err != nil {
				log.Println(
					fmt.Sprintf(
						"fakeServer send: read message have error [%s] with type [%d] and message [%s]",
						err.Error(),
						msgType,
						string(msgBytes),
					),
				)
				return
			}
			msg, err := wsunit.UnmarshallMassage(msgBytes)
			if err != nil {
				log.Println(err)
			} else {
				msg.ReceiveLogicTime = this.receiveMessageLogicTime
				log.Println(msg.String())
			}
		}
	}()
}
