package proxy

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"ws-server/src/client"
	"ws-server/src/model"

	"github.com/gorilla/websocket"
)

/*

DOC:

Решение заключает в том, что для каждого клиента
храним открытое соедиение и пулл сообщений полученных от
других клиентов.

Для каждого клиента, отправку сообщений ему и получения их от него,
выполняем параллельно основному потоку (главной горутине main).
Получение сообщения от клиента и передачу его другим клиентам осуществляется
через канал, который хранит все полученные сообщения от всех клиентов
за Квант вемени. Чтение сообщения из канала и передачу его в пул других клиентов
осуществляется в отдельной горутине. Запись в канал осуществляется в горутине клиента.

Здесь мы храним объекты всех подключенных клиентов. Отключение клиентов выполняется
в горутинах клиентов, а вот удаление и создание экземпляра клиента выполняется здесь.
Переход с HTTP на TCP для нового клиента выполняется в основном потоке, а вот управление
передачей и получения сообщений выполняется в отдельных горутинах каждого нового клиента.
Таким образом основной поток только ожидает либо новых новых клиентов, либо занимается
другим делами.

*/
type (
	RemoteClientAddress string

	Server struct {
		// Подключенные клиенты
		clients            map[RemoteClientAddress]*client.Client

		// Предельное кол-во обслуживаемых клиентов 
		maxCountClients    int

		// Предельная нагрузка на канал сообщений 
		maximumLoad        int

		// Предельный размер пула клиентов
		maximumPoolSize    int

		// Предельное кол-во попыток отправить сообщение из пула
		maximumCountRepeat int

		// Канал рассылки сообщений клиентам
		redirectChannel    chan model.Message

		// Защищаем мапу клиентов
		mx                 *sync.Mutex
	}
)

func NewServer(maxCountClients int, maximumLoad int, maximumPoolSize int, maximumCountRepeat int) *Server {
	this := &Server{
		clients:            make(map[RemoteClientAddress]*client.Client),
		maxCountClients:    maxCountClients,
		maximumLoad:        maximumLoad,
		maximumPoolSize:    maximumPoolSize,
		maximumCountRepeat: maximumCountRepeat,
		mx:                 new(sync.Mutex),
		redirectChannel:    make(chan model.Message, maxCountClients),
	}
	// Обслуживаем канал передачи сообщений клиентам
	go this.redirect()
	return this
}

func (this *Server) WriteMessage(msgType int, content []byte, receiver string) {
	go func() {
		msg := model.Message{
			Type: msgType,
			Receiver: receiver,
			Content: content,
		}
		this.redirectChannel <- msg
	}()
}

func (this *Server) AddNewClient(w http.ResponseWriter, r *http.Request) error {
	if len(this.clients) == this.maxCountClients {
		// Если кол-во подключенных клиентов достигло предела,
		// отказываем в обслуживании
		return errors.New("")
	}
	// Выполняем переключение протокола
	upgrader := &websocket.Upgrader{}
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	this.mx.Lock()
	this.clients[RemoteClientAddress(connection.RemoteAddr().String())] = client.NewClient(
		connection,
		this.redirectChannel,
		this.maximumLoad,
		this.maximumPoolSize,
		this.maximumCountRepeat,
	)
	this.mx.Unlock()
	return nil
}

func (this *Server) redirect() {
	for msg := range this.redirectChannel {
		log.Println(" ----> Server: redirect message ", string(msg.Content))
		for _, cl := range this.clients {
			// Не посылаем сообщение клиенту, который отправил
			// сообщение 
			if msg.Receiver == cl.RemoteAddr {
				continue
			}
			// Проверяем закрыто ли соединение с клиентов
			if cl.PossiblySendingMessage() {
				// Отправляем собщение в пул клиента
				cl.Send(
					msg,
				)
			} else {
				// Удаляем клиента если соедение с сервером закрыто
				this.mx.Lock()
				delete(this.clients, RemoteClientAddress(cl.RemoteAddr))
				this.mx.Unlock()
			}
		}
	}
}
