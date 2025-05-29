package client_test

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/foomo/contentserver/client"
	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/handler"
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
	r := repo.New(l,
		testRepoServer.URL+"/repo-two-dimensions.json",
		repo.NewHistory(l,
			repo.HistoryWithHistoryDir(varDir),
		),
	)
	up := make(chan bool, 1)
	r.OnLoaded(func() {
		up <- true
	})
	go r.Start(tb.Context()) //nolint:errcheck
	<-up
	return r
}

func TestGetRepo_UpdateFails(t *testing.T) {
	t.Helper()
	t.Parallel()

	l := zaptest.NewLogger(t)

	// Step 1: create a working repo and let it populate history
	testRepoServer, varDir := mock.GetMockData(t)
	historyDir := varDir

	workingRepo := repo.New(l,
		testRepoServer.URL+"/repo-two-dimensions.json",
		repo.NewHistory(l, repo.HistoryWithHistoryDir(historyDir)),
	)
	go func() {
		if err := workingRepo.Start(t.Context()); err != nil {
			t.Errorf("workingRepo.Start failed: %v", err)
		}
	}()

	// Give it time to persist JSON
	time.Sleep(300 * time.Millisecond)

	// Step 2: create a new repo with a broken upstream, but reusing the same history
	brokenRepo := repo.New(l,
		"http://localhost:9999/this-will-fail-non-existent.json", // force failure
		repo.NewHistory(l, repo.HistoryWithHistoryDir(historyDir)),
	)
	go func() {
		if err := brokenRepo.Start(t.Context()); err != nil {
			t.Errorf("brokenRepo.Start failed: %v", err)
		}
	}()

	// Step 3: serve it
	server := httptest.NewServer(handler.NewHTTP(l, brokenRepo))
	defer server.Close()

	client, err := client.NewHTTPClient(server.URL + pathContentserver)
	require.NoError(t, err)

	// Step 4: trigger GetRepo, which will log warning and fall back to existing file
	result, err := client.GetRepo(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, result, "expected fallback repo content from WriteRepoBytes")
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
