package model

import (
	"errors"
	"regexp"
)

type Message struct {
	Type     int
	Receiver string
	Content  []byte
}

type RemoteAddress string

func NewRemoteAddress(address string) (RemoteAddress, error) {
	match, err := regexp.MatchString(
		`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]):[0-9]+$`,
		address,
	)
	if err != nil {
		return RemoteAddress(""), err
	}
	if !match {
		return RemoteAddress(""), errors.New("No matched.")
	}
	return RemoteAddress(address), nil
}

type WSMessage struct {
	SenderAddress RemoteAddress
	Type          int
	Content       []byte
	Err           error
}
