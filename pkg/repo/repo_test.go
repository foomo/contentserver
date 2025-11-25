package repo

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/foomo/contentserver/pkg/repo/mock"
	"github.com/foomo/contentserver/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func NewTestRepo(ctx context.Context, l *zap.Logger, url, varDir string) *Repo {
	h, err := NewHistory(l, HistoryWithHistoryLimit(2), HistoryWithHistoryDir(varDir))
	if err != nil {
		panic(err)
	}
	r := New(l, url, h)
	go r.Start(ctx) //nolint:errcheck
	time.Sleep(100 * time.Millisecond)
	return r
}

func assertRepoIsEmpty(t *testing.T, r *Repo, empty bool) {
	t.Helper()
	if empty {
		if len(r.Directory()) > 0 {
			t.Fatal("directory should have been empty, but is not")
		}
	} else {
		if len(r.Directory()) == 0 {
			t.Fatal("directory is empty, but should have been not")
		}
	}
}

func TestLoad404(t *testing.T) {
	var (
		l                  = zaptest.NewLogger(t)
		mockServer, varDir = mock.GetMockData(t)
		url                = mockServer.URL + "/repo-no-have"
		r                  = NewTestRepo(t.Context(), l, url, varDir)
	)

	response := r.Update(t.Context())
	if response.Success {
		t.Fatal("can not get a repo, if the server responds with a 404")
	}
}

func TestLoadBrokenRepo(t *testing.T) {
	var (
		l                  = zaptest.NewLogger(t)
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-broken-json.json"
		r                  = NewTestRepo(t.Context(), l, server, varDir)
	)

	response := r.Update(t.Context())
	if response.Success {
		t.Fatal("how could we load a broken json")
	}
}

func TestLoadRepo(t *testing.T) {
	var (
		l                  = zaptest.NewLogger(t)
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-ok.json"
		r                  = NewTestRepo(t.Context(), l, server, varDir)
	)
	assertRepoIsEmpty(t, r, false)

	response := r.Update(t.Context())
	assertRepoIsEmpty(t, r, false)

	if !response.Success {
		t.Fatal("could not load valid repo")
	}
	if response.Stats.OwnRuntime > response.Stats.RepoRuntime {
		t.Fatal("how could all take less time, than me alone")
	}
	if response.Stats.RepoRuntime < 0.05 {
		t.Fatal("the server was too fast")
	}

	// see what happens if we try to start it up again
	// nr := NewTestRepo(l, server, varDir)
	// assertRepoIsEmpty(t, nr, false)
}

func BenchmarkLoadRepo(b *testing.B) {
	var (
		l                  = zaptest.NewLogger(b)
		t                  = &testing.T{}
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-ok.json"
		r                  = NewTestRepo(b.Context(), l, server, varDir)
	)

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		response := r.Update(b.Context())
		if len(r.Directory()) == 0 {
			b.Fatal("directory is empty, but should have been not")
		}

		if !response.Success {
			b.Fatal("could not load valid repo")
		}
	}
}

func TestLoadRepoDuplicateUris(t *testing.T) {
	var (
		l                  = zaptest.NewLogger(t)
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-duplicate-uris.json"
		r                  = NewTestRepo(t.Context(), l, server, varDir)
	)

	response := r.Update(t.Context())
	require.False(t, response.Success, "there are duplicates, this repo update should have failed")

	assert.Contains(t, response.ErrorMessage, "update dimension")
}

func TestDimensionHygiene(t *testing.T) {
	l := zaptest.NewLogger(t)

	mockServer, varDir := mock.GetMockData(t)
	server := mockServer.URL + "/repo-two-dimensions.json"
	r := NewTestRepo(t.Context(), l, server, varDir)

	response := r.Update(t.Context())
	require.True(t, response.Success, "well those two dimension should be fine")

	r.url = mockServer.URL + "/repo-ok.json"
	response = r.Update(t.Context())
	require.True(t, response.Success, "it is called repo ok")

	assert.Lenf(t, r.Directory(), 1, "directory hygiene failed")
}

func getTestRepo(t *testing.T, path string) *Repo {
	t.Helper()
	l := zaptest.NewLogger(t)

	mockServer, varDir := mock.GetMockData(t)
	server := mockServer.URL + path
	r := NewTestRepo(t.Context(), l, server, varDir)
	response := r.Update(t.Context())

	require.True(t, response.Success, "well those two dimension should be fine")

	return r
}

func TestGetNodes(t *testing.T) {
	r := getTestRepo(t, "/repo-two-dimensions.json")
	nodesRequest := mock.MakeNodesRequest()
	nodes := r.GetNodes(nodesRequest)
	testNode, ok := nodes["test"]

	require.True(t, ok, "should be a node")

	testData, ok := testNode.Item.Data["foo"]
	require.True(t, ok, "failed to fetch test data")

	t.Log("testData", testData)
}

func TestGetNodesExposeHidden(t *testing.T) {
	r := getTestRepo(t, "/repo-ok-exposehidden.json")
	nodesRequest := mock.MakeNodesRequest()
	nodesRequest.Nodes["test"].ExposeHiddenNodes = true
	nodes := r.GetNodes(nodesRequest)

	testNode, ok := nodes["test"]
	require.True(t, ok, "should be a node")

	_, ok = testNode.Item.Data["foo"]
	require.True(t, ok, "failed to fetch test data")

	require.Len(t, testNode.Nodes, 2)
}

func TestResolveContent(t *testing.T) {
	r := getTestRepo(t, "/repo-two-dimensions.json")
	contentRequest := mock.MakeValidContentRequest()
	siteContent, err := r.GetContent(contentRequest)
	require.NoError(t, err)
	assert.Equal(t, contentRequest.URI, siteContent.URI, "failed to resolve uri")
}

func TestLinkIds(t *testing.T) {
	l := zaptest.NewLogger(t)

	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-link-ok.json"
		r                  = NewTestRepo(t.Context(), l, server, varDir)
		response           = r.Update(t.Context())
	)

	if !response.Success {
		t.Fatal("those links should have been fine")
	}

	r.url = mockServer.URL + "/repo-link-broken.json"
	response = r.Update(t.Context())

	if response.Success {
		t.Fatal("I do not think so")
	}
}

func TestInvalidRequest(t *testing.T) {
	r := getTestRepo(t, "/repo-two-dimensions.json")

	if r.validateContentRequest(mock.MakeValidContentRequest()) != nil {
		t.Fatal("failed validation a valid request")
	}

	tests := map[string]*requests.Content{}

	rEmptyURI := mock.MakeValidContentRequest()
	rEmptyURI.URI = ""
	tests["empty uri"] = rEmptyURI

	rEmptyEnv := mock.MakeValidContentRequest()
	rEmptyEnv.Env = nil
	tests["empty env"] = rEmptyEnv

	rEmptyEnvDimensions := mock.MakeValidContentRequest()
	rEmptyEnvDimensions.Env.Dimensions = []string{}
	tests["empty env dimensions"] = rEmptyEnvDimensions

	// rNodesValidID := mock.MakeValidContentRequest()
	// rNodesValidID.Nodes["id-root"].Id = ""
	// tests["nodes must have a valid id"] = rNodesValidID

	for comment, req := range tests {
		if r.validateContentRequest(req) == nil {
			t.Fatal(comment, "should have failed")
		}
	}
}

func TestWriteRepoBytesRace(t *testing.T) {
	var (
		l                  = zaptest.NewLogger(t)
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-ok.json"
		r                  = NewTestRepo(t.Context(), l, server, varDir)
	)

	response := r.Update(t.Context())
	require.True(t, response.Success, "should load repo successfully")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					var buf bytes.Buffer
					_ = r.WriteRepoBytes(ctx, &buf)
				}
			}
		}()
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					newBuf := bytes.NewBufferString(`{"test":"data"}`)
					r.SetJSONBuffer(newBuf)
				}
			}
		}()
	}
	wg.Wait()
}
