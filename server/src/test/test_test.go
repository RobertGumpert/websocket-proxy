package test

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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
	for i := 0; i < 2; i++ {
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
	serverMessages := time.NewTicker(14 * time.Second)
	index := len(clients) - 1

	for {
		select {
		case <-serverMessages.C:
			server.Proxy.WriteMessage(
				websocket.TextMessage,
				[]byte("Message from server"),
				server.Server.URL,
			)
		case <-closeClient.C:
			// поэтапно отключаем клиентов
			if index == 0 {
				time.Sleep(5 * time.Second)
				return
			}
			clients[index].closeConnection <- struct{}{}
			index--
		case <-sendMessages.C:
			// Шлем сообщения серверу и клиентам для того чтобы убедиться
			// в том что не блокируются обработкой открытого подключения
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
