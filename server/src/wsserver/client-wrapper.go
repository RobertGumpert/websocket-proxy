package wsserver

import (
	"net/http"
	"ws-server/src/wsclient"
)

type clientWrapper struct {
	Client        *wsclient.Client
	CountMessages int
}

func NewClient(w http.ResponseWriter, r *http.Request) (*clientWrapper, error) {
	cl, err := wsclient.NewClient(w, r)
	if err != nil {
		return nil, err
	}
	this := new(clientWrapper)
	this.Client = cl
	this.CountMessages = 0
	return this, nil
}

func (this *clientWrapper) StartServing(callback wsclient.CallbackOnReceive) {
	this.Client.StartServing(callback)
	return
}