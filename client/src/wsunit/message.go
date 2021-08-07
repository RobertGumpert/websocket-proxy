package wsunit

import (
	"encoding/json"
	"fmt"
)

type message struct {
	Type    int    `json:"type"`
	Content []byte `json:"content"`

	ReceiverAddr     string `json:"recever_addr"`
	ReceiveTime      int64  `json:"receive_time"`
	ReceiveLogicTime int    `json:"receive_logic_time"`

	SenderAddr    string `json:"sender_addr"`
	SendTime      int64  `json:"send_time"`
	SendLogicTime int    `json:"send_logic_time"`
}

func NewMessage(
	msgType int,
	msgContent []byte,
	//
	receiverAddr string,
	receiveTime int64,
	receiveTimeLogic int,
	//
	senderAddr string,
	sendTime int64,
	sendTimeLogic int,
) message {
	return message{
		Type:    msgType,
		Content: msgContent,
		//
		ReceiverAddr:     receiverAddr,
		ReceiveTime:      receiveTime,
		ReceiveLogicTime: receiveTimeLogic,
		//
		SenderAddr:    senderAddr,
		SendTime:      sendTime,
		SendLogicTime: sendTimeLogic,
	}
}

func (this message) Marshall() ([]byte, error) {
	bts, err := json.Marshal(&this)
	if err != nil {
		return nil, err
	}
	return bts, nil
}

func UnmarshallMassage(bts []byte) (message, error) {
	var this message
	err := json.Unmarshal(bts, &this)
	if err != nil {
		return message{}, err
	}
	return this, nil
}

func (this message) String() string {
	return fmt.Sprintf(
		"MSG: [%s]; receiver/sender: [%s/%s]; logic time local/logic time remote : [%d/%d]",
		string(this.Content),
		this.ReceiverAddr,
		this.SenderAddr,
		this.ReceiveLogicTime,
		this.SendLogicTime,
	)
}
