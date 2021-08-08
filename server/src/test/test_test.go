package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

var (
	countMessages = 100
	countClients  = 5
)

func TestStackFlow(t *testing.T) {
	var (
		fakeClients = make([]*FakeClientWrapper, 0)
	)
	fakeServer := NewFakeStackWrapper(countMessages, countClients)
	for i := 0; i < countClients; i++ {
		fakeClient := NewFakeClientWrapper(fakeServer.Server, i)
		resp, err := http.Get(
			fmt.Sprintf("%s/ws",
				fakeClient.Server.URL,
			),
		)
		if err != nil {
			t.Fatal(
				fmt.Sprintf(
					"Code [%s], err [%s]",
					resp.Status,
					err.Error(),
				),
			)
		}
		fakeClients = append(fakeClients, fakeClient)
	}

	timerTestFinish := time.NewTicker(35 * time.Second)
	timerForAPIReq := time.NewTicker(time.Second)

	for {
		select {
		case <-timerForAPIReq.C:
			resp, err := http.Get(
				fmt.Sprintf("%s/msg",
					fakeServer.Server.URL,
				),
			)
			if err != nil {
				t.Fatal(
					fmt.Sprintf(
						"Code [%s], err [%s]",
						resp.Status,
						err.Error(),
					),
				)
			}
			for _, cl := range fakeClients {
				resp, err := http.Get(
					fmt.Sprintf("%s/msg",
						cl.Server.URL,
					),
				)
				if err != nil {
					t.Fatal(
						fmt.Sprintf(
							"Code [%s], err [%s]",
							resp.Status,
							err.Error(),
						),
					)
				}
			}
		case <-timerTestFinish.C:
			// for _, cl := range fakeClients {
			// 	err := cl.Unit.SendMessageOnClose("")
			// 	if err != nil {
			// 		log.Fatal(err)
			// 	}
			// }
			fakeServer.Stack.CloseConnectionWithAllClients()
			return
		}
	}
}
