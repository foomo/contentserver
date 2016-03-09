package client

import (
	"encoding/json"
	"sync"
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

func getTestClient(t testing.TB) *Client {
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

func TestGetURIs(t *testing.T) {
	c := getTestClient(t)
	request := mock.MakeValidURIsRequest()
	uriMap, err := c.GetURIs(request.Dimension, request.IDs)
	if err != nil {
		t.Fatal(err)
	}
	if uriMap[request.IDs[0]] != "/a" {
		t.Fatal(uriMap)
	}
}

func TestGetRepo(t *testing.T) {
	c := getTestClient(t)
	r, err := c.GetRepo()
	if err != nil {
		t.Fatal(err)
	}
	if r["dimension_foo"].Nodes["id-a"].Data["baz"].(float64) != float64(1) {
		t.Fatal("failed to drill deep for data")
	}
}

func TestGetNodes(t *testing.T) {
	c := getTestClient(t)
	nodesRequest := mock.MakeNodesRequest()
	nodes, err := c.GetNodes(nodesRequest.Env, nodesRequest.Nodes)
	if err != nil {
		t.Fatal(err)
	}
	testNode, ok := nodes["test"]
	if !ok {
		t.Fatal("that should be a node")
	}
	testData, ok := testNode.Item.Data["foo"]
	if !ok {
		t.Fatal("where is foo")
	}
	if testData != "bar" {
		t.Fatal("testData should have bennd bar not", testData)
	}

}

func TestGetContent(t *testing.T) {
	c := getTestClient(t)
	request := mock.MakeValidContentRequest()
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

// not very meaningful yet
func BenchmarkServerAndClient(b *testing.B) {
	var wg sync.WaitGroup
	stats := make([]int, 100)
	for group := 0; group < 100; group++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			c := getTestClient(b)
			request := mock.MakeValidContentRequest()
			for i := 0; i < 1000; i++ {
				response, err := c.GetContent(request)
				if err != nil {
					b.Fatal("unexpected err", err)
				}
				if request.URI != response.URI {
					b.Fatal("uri mismatch")
				}
				if response.Status != content.StatusOk {
					b.Fatal("unexpected status")
				}
				stats[g] = i
			}
		}(group)

	}
	// Wait for all HTTP fetches to complete.
	wg.Wait()
	b.Log(stats)
}
