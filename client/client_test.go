package client

import (
	"testing"

	"github.com/foomo/contentserver/repo/mock"
	"github.com/foomo/contentserver/server"
)

var testServerIsRunning = false

func getTestClient(t *testing.T) *Client {
	addr := "127.0.0.1:9999"
	if !testServerIsRunning {
		testServerIsRunning = true
		testServer, varDir := mock.GetMockData(t)
		go server.Run(testServer.URL+"/repo-two-dimensions.json", addr, varDir)
	}
	return &Client{
		Server: addr,
	}
}

func TestUpdate(t *testing.T) {
	c := getTestClient(t)
	response, err := c.Update()
	t.Log("test update", response, err)
}
