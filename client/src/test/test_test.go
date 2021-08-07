package test

import (
	"log"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
	"wsclient/src/wsunit"
)

func TestReadMessagesFlow(t *testing.T) {
	server := newFakeServer()
	client := newFakeClient(server, "/sender")
	var count int
	for m := range client.GetReceivedMessges() {
		log.Println(m.String())
		if count == 30 {
			return
		}
		count++
	}
	server.Close()
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSendMessagesFlow(t *testing.T) {
	server := newFakeServer()
	client := newFakeClient(server, "/receiver")
	var count int
	timer := time.NewTicker(time.Second)
	for range timer.C {
		err := client.Send(
			"Message from client",
		)
		if err != nil {
			log.Println(err)
		}
		if count == 30 {
			return
		}
		count++
	}
}

func newFakeServer() *httptest.Server {
	fakeServer := NewFakeServer()
	testServer := httptest.NewServer(fakeServer)
	u, err := url.Parse(testServer.URL)
	if err != nil {
		log.Fatal(err)
	}
	fakeServer.URL = u
	return testServer
}

func newFakeClient(server *httptest.Server, endpoint string) *wsunit.RemoteServerUnit {
	backendURL, err := url.Parse(server.URL)
	backendURL.Path = endpoint
	if err != nil {
		log.Fatal("newFakeClient: url parse have error [", err, "]")
	}
	unit, err := wsunit.NewRemoteServerUnit(backendURL, 100)
	if err != nil {
		log.Fatal("newFakeClient: new Unit have error [", err, "]")
	}
	log.Println("Remote ADDR: ", backendURL.Host)
	return unit
}
