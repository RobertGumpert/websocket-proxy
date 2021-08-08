package test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"ws-server/src/proxy"
)

type ServerWrapper struct {
	Proxy  *proxy.Server
	Server *httptest.Server
}

func NewFakeServer() *ServerWrapper {
	this := new(ServerWrapper)
	this.Proxy = proxy.NewServer(
		5, // clients - кол-во клиентов
		5, // max messages - предельное кол-во получаемых сообщений 
		5, // max pool messages for client - предельноый размер пула подкл. клиентов
		2, // max count repeat - предельное кол-во попыток отправить сообщение из пула подкл. клиента 
	)
	server := httptest.NewServer(this)
	log.Println(fmt.Sprintf("Server started with address [%s]", server.URL))
	this.Server = server
	return this
}

func (this *ServerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ws":
		if err := this.Proxy.AddNewClient(w, r); err != nil {
			log.Println(err)
		}
	case "/msg":
		log.Println(
			"Server receive API message",
		)
	}
}
