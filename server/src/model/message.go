package model

type Message struct {
	Type     int
	Receiver string
	Content  []byte
}
