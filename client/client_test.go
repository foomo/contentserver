package client

import (
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/foomo/contentserver/content"
	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo/mock"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/server"
)

const pathContentserver = "/contentserver"

var (
	testServerSocketAddr    string
	testServerWebserverAddr string
)

func init() {
	SetupLogging(true, "contentserver_client_test.log")
}

func dump(t *testing.T, v interface{}) {
	jsonBytes, err := json.MarshalIndent(v, "", "	")
	if err != nil {
		t.Fatal("could not dump v", v, "err", err)
		return
	}
	t.Log(string(jsonBytes))
}

func getFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func getAvailableAddr() string {
	return "127.0.0.1:" + strconv.Itoa(getFreePort())
}

func initTestServer(t testing.TB) (socketAddr, webserverAddr string) {
	socketAddr = getAvailableAddr()
	webserverAddr = getAvailableAddr()
	testServer, varDir := mock.GetMockData(t)

	go func() {
		err := server.RunServerSocketAndWebServer(
			testServer.URL+"/repo-two-dimensions.json",
			socketAddr,
			webserverAddr,
			pathContentserver,
			varDir,
		)
		if err != nil {
			t.Fatal("test server crashed: ", err)
		}
	}()
	socketClient, errClient := NewClient(socketAddr, 1, time.Duration(time.Millisecond*100))
	if errClient != nil {
		panic(errClient)
	}
	i := 0
	for {
		time.Sleep(time.Millisecond * 100)
		r, err := socketClient.GetRepo()
		if err != nil {
			continue
		}
		if r["dimension_foo"].Nodes["id-a"].Data["baz"].(float64) == float64(1) {
			break
		}
		if i > 100 {
			panic("this is taking too long")
		}
		i++
	}
	return
}

func getTestClients(t testing.TB) (socketClient *Client, httpClient *Client) {
	if testServerSocketAddr == "" {
		socketAddr, webserverAddr := initTestServer(t)
		testServerSocketAddr = socketAddr
		testServerWebserverAddr = webserverAddr
	}
	socketClient, errClient := NewClient(testServerSocketAddr, 25, time.Duration(time.Millisecond*100))
	if errClient != nil {
		t.Log(errClient)
		t.Fail()
	}
	httpClient, errHTTPClient := NewHTTPClient("http://" + testServerWebserverAddr + pathContentserver)
	if errHTTPClient != nil {
		t.Log(errHTTPClient)
		t.Fail()
	}
	return
}

func testWithClients(t *testing.T, testFunc func(c *Client)) {
	socketClient, httpClient := getTestClients(t)
	defer socketClient.ShutDown()
	defer httpClient.ShutDown()
	testFunc(socketClient)
	testFunc(httpClient)
}

func TestUpdate(t *testing.T) {
	testWithClients(t, func(c *Client) {
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
	})
}

func TestGetURIs(t *testing.T) {
	testWithClients(t, func(c *Client) {
		defer c.ShutDown()
		request := mock.MakeValidURIsRequest()
		uriMap, err := c.GetURIs(request.Dimension, request.IDs)
		if err != nil {
			t.Fatal(err)
		}
		if uriMap[request.IDs[0]] != "/a" {
			t.Fatal(uriMap)
		}
	})
}

func TestGetRepo(t *testing.T) {
	testWithClients(t, func(c *Client) {
		r, err := c.GetRepo()
		if err != nil {
			t.Fatal(err)
		}
		if r["dimension_foo"].Nodes["id-a"].Data["baz"].(float64) != float64(1) {
			t.Fatal("failed to drill deep for data")
		}
	})
}

func TestGetNodes(t *testing.T) {
	testWithClients(t, func(c *Client) {
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
	})
}

func TestGetContent(t *testing.T) {
	testWithClients(t, func(c *Client) {
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
	})
}

func BenchmarkSocketClientAndServerGetContent(b *testing.B) {
	socketClient, _ := getTestClients(b)
	benchmarkServerAndClientGetContent(b, 30, 100, socketClient)

}
func BenchmarkWebClientAndServerGetContent(b *testing.B) {
	_, httpClient := getTestClients(b)
	benchmarkServerAndClientGetContent(b, 30, 100, httpClient)
}

func benchmarkServerAndClientGetContent(b *testing.B, numGroups, numCalls int, client GetContentClient) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		benchmarkClientAndServerGetContent(b, numGroups, numCalls, client)
		dur := time.Since(start)
		totalCalls := numGroups * numCalls
		b.Log("requests per second", int(float64(totalCalls)/(float64(dur)/float64(1000000000))), dur, totalCalls)
	}
}

type GetContentClient interface {
	GetContent(request *requests.Content) (response *content.SiteContent, err error)
}

func benchmarkClientAndServerGetContent(b testing.TB, numGroups, numCalls int, client GetContentClient) {
	var wg sync.WaitGroup
	wg.Add(numGroups)
	for group := 0; group < numGroups; group++ {
		go func(g int) {
			defer wg.Done()
			request := mock.MakeValidContentRequest()
			for i := 0; i < numCalls; i++ {
				response, err := client.GetContent(request)
				if err == nil {
					if request.URI != response.URI {
						b.Fatal("uri mismatch")
					}
					if response.Status != content.StatusOk {
						b.Fatal("unexpected status")
					}
				}
			}
		}(group)
	}
	// Wait for all HTTP fetches to complete.
	wg.Wait()
}
