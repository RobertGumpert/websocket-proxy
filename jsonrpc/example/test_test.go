package example

import (
	"encoding/json"
	"jrpc"
	"log"
	"testing"
)

func TestFlow1(t *testing.T) {
	//
	// Client -->
	//
	dataForRequest := &Data{
		Params: "hello, world!",
	}
	reqBody := &jrpc.Request{
		Jsonrpc: jrpc.ProtoVersion,
		Method: "hello",
		Params: dataForRequest,
		ID: 1,
	}
	bts, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	//
	// Server -->
	//
	request, _, err := jrpc.ParseRequest(bts, new(Data))
	if err != nil {
		t.Fatal(err)
	}
	request.Params.(*Data).Params = "Hello, from server!"
	response, err := request.Response(request.Params)
	if err != nil {
		t.Fatal(err)
	}
	bts, err = response.Marshall()
	if err != nil {
		t.Fatal(err)
	}
	//
	// Client -->
	//
	is := jrpc.IsResponse(bts)
	if is {
		response, err := jrpc.ParseResponse(bts, new(Data))
		if err != nil {
			t.Fatal(err)
		}
		log.Println(response.Result.(*Data).Params)
	} else {
		t.Fatal("Non response")
	}
}
