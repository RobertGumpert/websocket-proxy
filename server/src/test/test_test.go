package test

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestProxyFlow(t *testing.T) {
	server := NewFakeServer()
	serverURL, err := url.Parse(server.Server.URL)
	if err != nil {
		t.Fatal(err)
	}
	var (
		clients = make([]*FakeClientWrapper, 0)
	)
	for i := 0; i < 5; i++ {
		client := NewFakeClient(
			i,
			serverURL,
		)
		_, err := http.Get(
			fmt.Sprintf(
				"%s/ws",
				client.ServerAddr,
			),
		)
		if err != nil {
			log.Println(err)
		} else {
			clients = append(clients, client)
		}
	}

	sendMessages := time.NewTicker(time.Second)
	closeClient := time.NewTicker(13 * time.Second)
	index := len(clients) - 1

	for {
		select {
		case <-closeClient.C:
			if index == 0 {
				time.Sleep(5 * time.Second)
				return
			}
			clients[index].closeConnection <- struct{}{}
			index--
		case <-sendMessages.C:
			http.Get(
				fmt.Sprintf(
					"%s/msg",
					server.Server.URL,
				),
			)
			for _, cl := range clients {
				http.Get(
					fmt.Sprintf(
						"%s/msg",
						cl.ServerAddr,
					),
				)
			}
		}
	}
}
