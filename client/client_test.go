package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo/mock"
	"github.com/foomo/contentserver/server"
)

var testServerIsRunning = false

func dump(t *testing.T, v interface{}) {
	jsonBytes, err := json.MarshalIndent(v, "", "	")
	if err != nil {
		t.Fatal("could not dump v", v, "err", err)
		return
	}
	t.Log(string(jsonBytes))
}

func getTestClient(t *testing.T) *Client {
	log.SelectedLevel = log.LevelError
	addr := "127.0.0.1:9999"
	if !testServerIsRunning {
		testServerIsRunning = true
		testServer, varDir := mock.GetMockData(t)
		go server.Run(testServer.URL+"/repo-two-dimensions.json", addr, varDir)
		time.Sleep(time.Millisecond * 100)
	}
	return &Client{
		Server: addr,
	}
}

func TestUpdate(t *testing.T) {
	c := getTestClient(t)
	response, err := c.Update()
	if err != nil {
		t.Fatal("unexpected err", err)
	}
	if !response.Success {
		t.Fatal("update has to return .Sucesss true", response)
	}
	stats := response.Stats
	if !(stats.RepoRuntime > float64(0.0)) || !(stats.OwnRuntime > float64(0.0)) {
		t.Fatal("stats invalid")
	}
}

func TestGetContent(t *testing.T) {
	c := getTestClient(t)
	request := mock.MakeValidContentRequest()
	for i := 0; i < 1000; i++ {
		response, err := c.GetContent(request)
		if err != nil {
			t.Fatal("unexpected err", err)
		}
		if request.URI != response.URI {
			dump(t, request)
			dump(t, response)
			t.Fatal("uri mismatch")
		}

		if response.Status != content.StatusOk {
			t.Fatal("unexpected status")
		}

	}

}
