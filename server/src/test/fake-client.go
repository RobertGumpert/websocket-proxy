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

// Выполняем подключение клиенту
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
	// Обрабатываем сокет
	go this.handleWS()
	return nil
}

func (this *FakeClientWrapper) handleWS() {

	defer func() {
		// Закрываем подключение с сервером
		err := this.connection.Close()
		if err != nil {
			log.Println("Fnish :", err)
		}
	}()

	go func() {
		defer close(this.done)
		for {
			// Получаем сообщение от прокси сервера
			_, msg, err := this.connection.ReadMessage()
			if err != nil {
				// Клиента отключили или соединение по какой-то причине
				// завершено сервером
				log.Println(
					fmt.Sprintf(
						" ----> Client [%d] addr [%s] receive message with error [%s]",
						this.ID,
						this.WSAddr,
						err.Error(),
					),
				)
				// Завершаем чтение
				return
			}
			// Выводим сообщение
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
			// Звершаем отправку
			return
		case <-timer.C:
			// Для наглядности (не засорять консоль) отправляет сообщения
			// только первыый клиент
			if this.ID == 0 {
				err := this.connection.WriteMessage(
					websocket.TextMessage,
					[]byte(fmt.Sprintf(
						"Message from [%s] client",
						this.WSAddr,
					)))
				if err != nil {
					// Звершаем отправку
					log.Println(err)
					return
				}
			}
		case <-this.closeConnection:
			// Закрываем клиента извне (из теста)
			err := this.connection.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(
					websocket.CloseNormalClosure,
					"goobye",
				),
			)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}
