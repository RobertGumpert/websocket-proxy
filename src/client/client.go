package client

import (
	"github.com/RobertGumpert/wsunit"
	"net/http"
)

type RemoteClient struct {
	unit *wsunit.Unit
	Host string
}

func NewRemoteClient(
	receiveChannel chan wsunit.RecievedMessage,
	closeChannel chan wsunit.ConnectionCloser,
	wsunitLogs bool,
) *RemoteClient {
	unit := wsunit.NewUnit(receiveChannel, closeChannel, 100)
	// if !wsunitLogs {
	// 	unit.TurnOnOffLogs()
	// }
	return &RemoteClient{
		unit: unit,
	}
}

func (this *RemoteClient) SwitchConnectionHttpToWebsocket(w http.ResponseWriter, r *http.Request) error {
	err := this.unit.AsRemoteClientUnit(w, r)
	if err != nil {
		return err
	}
	this.Host = this.unit.RemoteAddr
	return nil
}

func (this *RemoteClient) SendMessage(message wsunit.RecievedMessage) error {
	this.unit.SendPreparedMessage(message)
	return nil
}

func (this *RemoteClient) HardCloseConnection() error {
	return this.unit.Close()
}

func (this *RemoteClient) SendMessageOnClose() error {
	return this.unit.SendMessageOnClose("goodbye")
}

