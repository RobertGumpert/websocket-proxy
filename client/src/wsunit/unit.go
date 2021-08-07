package wsunit

import (
	"fmt"
	"log"
	"net/url"
	"time"
	"github.com/gorilla/websocket"
)

type RemoteServerUnit struct {
	connect          *websocket.Conn
	backendServerURL *url.URL
	//
	receiveMessageChannel    chan message
	receiveMessageLogicTime  int
	receiveMessageBufferSize int
	//
	sendMessageChannel   chan []byte
	sendMessageLogicTime int
}

func NewRemoteServerUnit(backendServerURL *url.URL, bufferSize uint) (*RemoteServerUnit, error) {
	backendServerURL.Scheme = "ws"
	connent, resp, err := websocket.DefaultDialer.Dial(backendServerURL.String(), nil)
	if err != nil {
		log.Println(
			fmt.Sprintf(
				"Handshake have error: status [%s], context [%s]",
				resp.Status,
				err.Error(),
			),
		)
		return nil, err
	}
	bfSize := int(bufferSize)
	this := &RemoteServerUnit{
		connect:                  connent,
		backendServerURL:         backendServerURL,
		receiveMessageChannel:    make(chan message, bfSize),
		receiveMessageBufferSize: bfSize,
		sendMessageChannel:       make(chan []byte),
	}
	go this.read()
	go this.send()
	return this, nil
}

func (this *RemoteServerUnit) Close() error {
	return this.connect.Close()
}

func (this *RemoteServerUnit) GetReceivedMessges() chan message {
	return this.receiveMessageChannel
}

func (this *RemoteServerUnit) ReadNext() message {
	return <-this.receiveMessageChannel
}

func (this *RemoteServerUnit) close(msg string) {
	defer this.connect.Close()
	m, err := NewMessage(
		websocket.CloseMessage,
		[]byte(msg),
		this.connect.LocalAddr().String(),
		time.Now().Unix(),
		this.receiveMessageLogicTime,
		this.backendServerURL.Host,
		0, 0,
	).Marshall()
	if err != nil {
		this.connect.Close()
		panic(err)
	}
	this.connect.WriteMessage(
		websocket.CloseMessage,
		m,
	)
}

func (this *RemoteServerUnit) read() {
	for {
		this.receiveMessageLogicTime++
		msgType, msgBytes, err := this.connect.ReadMessage()
		if msgType == websocket.CloseMessage {
			this.connect.Close()
			break
		}
		if this.receiveMessageBufferSize < this.receiveMessageLogicTime+1 {
			this.close("QUEUE IS FILLED")
			break
		}
		if err != nil {
			this.close(err.Error())
			break
		}
		msg, err := UnmarshallMassage(msgBytes)
		if err != nil {
			log.Println(err)
			continue
		}
		msg.ReceiveLogicTime = this.receiveMessageLogicTime
		this.receiveMessageChannel <- msg
	}
}

func (this *RemoteServerUnit) Send(content string) error {
	this.sendMessageLogicTime++
	msg, err := NewMessage(
		websocket.TextMessage,
		[]byte(content),
		this.connect.RemoteAddr().String(), 0, 0,
		this.connect.LocalAddr().String(), time.Now().Unix(), this.sendMessageLogicTime,
	).Marshall()
	if err != nil {
		log.Println(err)
		this.sendMessageLogicTime--
		return err
	}
	go func(content []byte) {
		this.sendMessageChannel <- content
	}(msg)
	return nil
}

func (this *RemoteServerUnit) send() {
	for msg := range this.sendMessageChannel {
		err := this.connect.WriteMessage(
			websocket.TextMessage,
			msg,
		)
		if err != nil {
			log.Println(err)
		}
	}
}
