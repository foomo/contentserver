package client_test

import (
	"sync"
	"testing"
	"time"

	"github.com/foomo/contentserver/client"
	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/repo"
	"github.com/foomo/contentserver/pkg/repo/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestUpdate(t *testing.T) {
	testWithClients(t, func(t *testing.T, c *client.Client) {
		t.Helper()
		t.Parallel()
		response, err := c.Update(t.Context())
		require.NoError(t, err)
		require.True(t, response.Success, "update has to return .Sucesss true")
		assert.Greater(t, response.Stats.OwnRuntime, 0.0)
		assert.Greater(t, response.Stats.RepoRuntime, 0.0)
	})
}

func TestGetURIs(t *testing.T) {
	testWithClients(t, func(t *testing.T, c *client.Client) {
		t.Helper()
		t.Parallel()
		request := mock.MakeValidURIsRequest()
		uriMap, err := c.GetURIs(t.Context(), request.Dimension, request.IDs)
		time.Sleep(100 * time.Millisecond)
		require.NoError(t, err)
		assert.Equal(t, "/a", uriMap[request.IDs[0]])
	})
}

func TestGetRepo(t *testing.T) {
	testWithClients(t, func(t *testing.T, c *client.Client) {
		t.Helper()
		t.Parallel()
		r, err := c.GetRepo(t.Context())
		require.NoError(t, err)
		if assert.NotEmpty(t, r, "received empty JSON from GetRepo") {
			assert.InDelta(t, 1.0, r["dimension_foo"].Nodes["id-a"].Data["baz"].(float64), 0, "failed to drill deep for data") //nolint:forcetypeassert
		}
	})
}

func TestGetNodes(t *testing.T) {
	testWithClients(t, func(t *testing.T, c *client.Client) {
		t.Helper()
		t.Parallel()
		nodesRequest := mock.MakeNodesRequest()
		nodes, err := c.GetNodes(t.Context(), nodesRequest.Env, nodesRequest.Nodes)
		require.NoError(t, err)
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
	testWithClients(t, func(t *testing.T, c *client.Client) {
		t.Helper()
		t.Parallel()
		request := mock.MakeValidContentRequest()
		response, err := c.GetContent(t.Context(), request)
		require.NoError(t, err)
		assert.Equal(t, request.URI, response.URI)
		assert.Equal(t, content.StatusOk, response.Status)
	})
}

func benchmarkServerAndClientGetContent(b *testing.B, numGroups, numCalls int, client GetContentClient) {
	b.Helper()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		benchmarkClientAndServerGetContent(b, numGroups, numCalls, client)
		dur := time.Since(start)
		totalCalls := numGroups * numCalls
		b.Log("requests per second", int(float64(totalCalls)/(float64(dur)/float64(1000000000))), dur, totalCalls)
	}
}

func benchmarkClientAndServerGetContent(tb testing.TB, numGroups, numCalls int, client GetContentClient) {
	tb.Helper()
	var wg sync.WaitGroup
	wg.Add(numGroups)
	for group := 0; group < numGroups; group++ {
		go func() {
			defer wg.Done()
			request := mock.MakeValidContentRequest()
			for i := 0; i < numCalls; i++ {
				response, err := client.GetContent(tb.Context(), request)
				if err == nil {
					if request.URI != response.URI {
						tb.Fatal("uri mismatch")
					}
					if response.Status != content.StatusOk {
						tb.Fatal("unexpected status")
					}
				}
			}
		}()
	}
	// Wait for all HTTP fetches to complete.
	wg.Wait()
}

func testWithClients(t *testing.T, testFunc func(t *testing.T, c *client.Client)) {
	t.Helper()
	t.Run("http", func(t *testing.T) {
		l := zaptest.NewLogger(t)
		s := initHTTPRepoServer(t, l)
		c := newHTTPClient(t, s)
		defer func() {
			s.Close()
			c.Close()
		}()
		testFunc(t, c)
	})
	t.Run("socket", func(t *testing.T) {
		l := zaptest.NewLogger(t)
		s := initSocketRepoServer(t, l)
		c := newSocketClient(t, s.Addr().String())
		defer func() {
			s.Close()
			c.Close()
		}()
		testFunc(t, c)
	})
}

func initRepo(tb testing.TB, l *zap.Logger) *repo.Repo {
	tb.Helper()
	testRepoServer, varDir := mock.GetMockData(tb)
	h, err := repo.NewHistory(l,
		repo.HistoryWithHistoryDir(varDir),
	)
	if err != nil {
		tb.Fatal(err)
	}
	r := repo.New(l,
		testRepoServer.URL+"/repo-two-dimensions.json",
		h,
	)
	up := make(chan bool, 1)
	r.OnLoaded(func() {
		up <- true
	})
	go r.Start(tb.Context()) //nolint:errcheck
	<-up
	return r
}

// func dump(t *testing.T, v interface{}) {
// 	t.Helper()
// 	jsonBytes, err := json.MarshalIndent(v, "", "	")
// 	if err != nil {
// 		t.Fatal("could not dump v", v, "err", err)
// 		return
// 	}
// 	t.Log(string(jsonBytes))
// }
