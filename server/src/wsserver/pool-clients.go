package wsserver

import (
	"sync"
	"ws-server/src/model"
)

type clientPoolTuple struct {
	Address       model.RemoteAddress
	ClientWrapper *clientWrapper
}

type poolClients struct {
	mx   *sync.RWMutex
	pool map[model.RemoteAddress]*clientWrapper
}

func NewPoolClients() *poolClients {
	return &poolClients{
		mx:   new(sync.RWMutex),
		pool: make(map[model.RemoteAddress]*clientWrapper),
	}
}

func (this *poolClients) Push(client *clientWrapper) {
	defer this.mx.Unlock()
	this.mx.Lock()
	this.pool[client.Client.Address] = client
}

func (this *poolClients) Pop(address model.RemoteAddress) {
	defer this.mx.Unlock()
	this.mx.Lock()
	if _, exist := this.pool[address]; exist {
		delete(this.pool, address)
	}
}

func (this *poolClients) Iterate() <-chan clientPoolTuple {
	var (
		c = make(chan clientPoolTuple, len(this.pool))
	)
	go func() {
		defer this.mx.RUnlock()
		this.mx.RLock()
		for address, client := range this.pool {
			c <- clientPoolTuple{
				Address:       address,
				ClientWrapper: client,
			}
		}
		return
	}()
	return c
}

func (this *poolClients) Size() int {
	defer this.mx.RUnlock()
	this.mx.RLock()
	return len(this.pool)
}

func (this *poolClients) Get(address model.RemoteAddress) (clientPoolTuple, bool) {
	defer this.mx.RUnlock()
	this.mx.RLock()
	if cl, exist := this.pool[address]; exist {
		return clientPoolTuple{
			Address:       address,
			ClientWrapper: cl,
		}, true
	}
	return clientPoolTuple{}, false
}
