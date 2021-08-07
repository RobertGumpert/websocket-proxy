package client

import (
	"net/http"
	"github.com/RobertGumpert/wsunit"
)

type RemoteClient struct {
	unit *wsunit.Unit
	Host string
}

func NewRemoteClient(receiveChannel chan wsunit.Message, closeChannel chan wsunit.ConnectionCloser) *RemoteClient {
	unit := wsunit.NewUnit(receiveChannel, closeChannel)
	return &RemoteClient{
		unit: unit,
	}
}

func (this *RemoteClient) ConnectWithClient(w http.ResponseWriter, r *http.Request) error {
	err := this.unit.AsRemoteClientUnit(w, r)
	if err != nil {
		return err
	}
	this.Host = this.unit.RemoteAddr
	return nil
}

func (this *RemoteClient) SendToClient(message wsunit.Message) error {
	this.unit.SendPreparedMessage(message)
	return nil
}

func (this *RemoteClient) CloseConnectWithClient() error {
	return this.unit.Close()
}

func (this *RemoteClient) SendCloseMessage() error {
	this.unit.SendMessageOnClose("")
	return this.unit.Close()
}