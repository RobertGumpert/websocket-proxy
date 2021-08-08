package client

import (
	"fmt"
	"log"
	"sync"
	"time"
	"ws-server/src/model"

	"github.com/gorilla/websocket"
)

type message struct {
	// кол-во попыток отправить сообщение
	countRepeat int
	msg         model.Message
}


/*

DOC:

Решение заключает в том, что для каждого клиента
храним открытое соедиение и пулл сообщений полученных от
других клиентов.

Отправка сообщений клиенту будет осуществляться каждый Квант времени,
ведя статистику попыток отправить каждое сообщение из пула.
Сообщение будут считаться невалидными если кол-во попыток отправить
его клиенту превысит установленный порог.
Невалидные сообщения удаляются из пула.

Получение сообщения от другого клиента происходит из объекта Proxy,
который читает его из канала, хранящего сообщения полученные от все клиентов.

Здесь мы регламентируем порядок получения сообщения от клиента
и отправки сообщения клиенту.
Каждое сообщения полученное от клиента,
пишем в канал содержащий сообщения полученные от всех клиентов.
Сообщения от других клиентов получаем из этого же канала,
но через промежудочное звено - объект Proxy, который непосредственно
читает очередное сообщение из канала и передает его в пул каждого клиента.

*/
type Client struct {
	// Канал для передачи сообщения для завершения
	// горутины отправляющей сообщения по соекту
	done chan struct{}

	// Пул сообщений, ожидающих отправки клиенту
	pool []*message

	// Предельное количество попыток отправить сообщение
	maxCountRepeat int

	// Максимальный размер пула сообщений для отправки
	maxSizePool int

	// Открытое соединение
	connection *websocket.Conn

	// Флаг разрешения на отправку сообщений клиенту
	possiblySendingMessages bool

	// Предельное кол-во сообщений которое может принять Прокси-Сервер
	maximumLoad int

	// Канал передачи сообщений для рассылки сообщений остальным клиентам
	redirectChannel chan model.Message

	// Адрес клиента
	RemoteAddr string

	// Импользуем для защиты пула при вставке и удаление сообщений
	mx *sync.Mutex
}

func NewClient(
	connection *websocket.Conn,
	redirectChannel chan model.Message,
	maximumLoad int,
	maxSizePool int,
	maxCountRepeat int,
) *Client {
	this := &Client{
		connection:              connection,
		RemoteAddr:              connection.RemoteAddr().String(),
		possiblySendingMessages: true,
		redirectChannel:         redirectChannel,
		maximumLoad:             maximumLoad,
		pool:                    make([]*message, 0),
		maxSizePool:             maxSizePool,
		done:                    make(chan struct{}),
		maxCountRepeat:          maxCountRepeat,
		mx:                      new(sync.Mutex),
	}
	// Читаем сообщения от клиента
	go this.read()
	// Отправляем сообщения клиенту
	go this.send()
	return this
}

func (this *Client) Send(msg model.Message) {
	size := len(this.pool)
	if this.maxSizePool < (size + 1) {
		// Если размер пула достиг предельного значения,
		// считаем что клиент скорее всего не может принимать
		// сообщения но еще не закрыл соединение. Сообщаем что ожидать
		// сообщений более не стоит.
		this.done <- struct{}{}
	} else {
		// Помещаем новое сообщение в пулл
		this.mx.Lock()
		this.pool = append(this.pool, &message{
			msg: msg,
		})
		this.mx.Unlock()
	}
}

func (this *Client) deleteFromPool(index int) {
	tmp := make([]*message, 0)
	for i, msg := range this.pool {
		if i == index {
			continue
		}
		tmp = append(tmp, msg)
	}
	this.mx.Lock()
	this.pool = tmp
	this.mx.Unlock()
}

func (this *Client) send() {
	// Каждый n * миллисек. пытаемся отправить сообщения
	// из пула сообщений
	timer := time.NewTicker(time.Millisecond)
	defer func() {
		if this.possiblySendingMessages {
			err := this.connection.Close()
			if err != nil {
				log.Println("")
			}
			this.possiblySendingMessages = false
		}
	}()
	for {
		select {
		case <-this.done:
			return
		case <-timer.C:
			for index, msg := range this.pool {
				// Если счетчик сообщения достиг предельного
				// кол-ва попыток отправить его, удаляем.
				// С целью экономии места, отправим самый свежие сообщения
				// сообщения если клиент вдруг проснется
				if msg.countRepeat == this.maxCountRepeat {
					this.deleteFromPool(index)
				}
				err := this.connection.WriteMessage(
					msg.msg.Type,
					msg.msg.Content,
				)
				if err != nil {
					log.Println(
						fmt.Sprintf(" ----> Server: Redirect to remote client [%s] send message wiht error [%s].",
							this.RemoteAddr,
							err.Error(),
						),
					)
					msg.countRepeat++
				} else {
					this.deleteFromPool(index)
					log.Println(
						fmt.Sprintf(" ----> Server: redirect to remote client [%s] successfully.",
							this.RemoteAddr,
						),
					)
				}
			}
		}
	}
}

func (this *Client) read() {
	for {
		if !this.possiblySendingMessages {
			// Если "закрыли" горутину на запись,
			// то завершаем и горутину на чтение
			return
		}

		msgType, msgBytes, err := this.connection.ReadMessage()
		if len(this.redirectChannel) >= this.maximumLoad {
			// Если канал рассылки заполнен, завершим соединение
			// с клиентом, чтобы разгрузить прокси сервер
			//
			// P.S. боевое решение будет учитывать не размер (len) канала
			// а статистику отправки сообщений клиентом и закрывать соединение
			// исходя из его активности
			log.Println(
				fmt.Sprintf(" ----> Server: client [%s] close connection because buffer is filled", this.RemoteAddr),
			)
			break
		}
		if err != nil {
			// Не учитваем возможных краевых соедений, только считаем
			// что клиент инициализировал процесс закрытия соединения
			log.Println(
				fmt.Sprintf(" ----> Server: client [%s] close connection with error [%s]", this.RemoteAddr, err.Error()),
			)
			break
		}
		go func() {
			// Пишем полученное от клиента сообщение в канал 
 			// рассылки сообщений другим клиентам
			this.redirectChannel <- model.Message{
				Receiver: this.RemoteAddr,
				Type:     msgType,
				Content:  msgBytes,
			}
		}()
	}
	if this.possiblySendingMessages {
		err := this.connection.Close()
		if err != nil {
			log.Println(
				fmt.Sprintf(" ----> Server: client [%s] close connection with error [%s]", this.RemoteAddr, err.Error()),
			)
		}
		this.possiblySendingMessages = false
		this.done <- struct{}{}
	}
}

func (this *Client) PossiblySendingMessage() bool {
	return this.possiblySendingMessages
}
