package test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"ws-server/src/stack"
)

type FakeStackWrapper struct {
	Stack  *stack.RemoteClientsStack
	Server *httptest.Server
}

func NewFakeStackWrapper(
	sizeMessagesStack int,
	sizeClientsStack int,
) *FakeStackWrapper {
	st := stack.NewStack(
		sizeMessagesStack,
		sizeClientsStack,
	)
	this := &FakeStackWrapper{
		Stack: st,
	}
	server := httptest.NewServer(this)
	log.Println(fmt.Sprintf("Server started with address [%s]", server.URL))
	this.Server = server
	return this
}

func (this *FakeStackWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ws":
		err := this.Stack.AddClient(w, r, false)
		if err != nil {
			log.Fatal(err)
		}
	case "/msg":
		log.Println(
			"Receive API message on SERVER",
		)
	}
}
