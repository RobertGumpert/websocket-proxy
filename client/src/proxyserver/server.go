package proxyserver

import (
	"fmt"
	"log"
	"net/url"
	"github.com/gorilla/websocket"
)

type Server struct {
	connection      *websocket.Conn
	serverURL       *url.URL
	WSAddr          string
	done            chan struct{}
	CloseConnection chan struct{}
	SendMessages    chan struct{}
}

func NewServer(serverURL *url.URL) *Server {
	this := new(Server)
	serverURL.Scheme = "ws"
	this.serverURL = serverURL
	this.done = make(chan struct{})
	this.CloseConnection = make(chan struct{})
	this.SendMessages = make(chan struct{})
	return this
}

func (this *Server) ConnectToServer() error {
	connection, _, err := websocket.DefaultDialer.Dial(
		this.serverURL.String(),
		nil,
	)
	if err != nil {
		return err
	}
	this.connection = connection
	this.WSAddr = connection.LocalAddr().String()
	go this.handleWS()
	return nil
}

func (this *Server) handleWS() {

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
						" ----> Client addr [%s] receive message with error [%s]",
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
					" ----> Client addr [%s] receive message [%s]",
					this.WSAddr,
					string(msg),
				),
			)
		}
	}()

	for {
		select {
		case <-this.done:
			// Звершаем отправку
			return
		case <-this.SendMessages:
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
		case <-this.CloseConnection:
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
