package wstest

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"ws-server/src/wsserver"
)

type ServerWrapper struct {
	Proxy  *wsserver.Server
	Server *httptest.Server
}

func NewFakeServer() *ServerWrapper {
	this := new(ServerWrapper)
	this.Proxy = wsserver.NewServer(
		5,
		5,
		5,
	)
	server := httptest.NewServer(this)
	log.Println(fmt.Sprintf("Server started with address [%s]", server.URL))
	this.Server = server
	return this
}

func (this *ServerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ws":
		if err := this.Proxy.NewClient(w, r); err != nil {
			log.Println(err)
		}
	case "/msg":
		log.Println(
			"Server receive API message",
		)
	}
}