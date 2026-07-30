package mock

import (
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/foomo/contentserver/requests"
)

const (
	dimensionFoo = "dimension_foo"
	idRoot       = "id-root"
)

// GetMockData mock data to run a repo
func GetMockData(tb testing.TB) (*httptest.Server, string) {
	tb.Helper()

	_, filename, _, _ := runtime.Caller(0)
	mockDir := path.Dir(filename)
	fileServer := http.FileServer(http.Dir(mockDir))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Millisecond * 50)
		fileServer.ServeHTTP(w, req)
	}))

	go func() {
		<-tb.Context().Done()
		server.Close()
	}()

	return server, tb.TempDir()
}

// MakeNodesRequest a request to get some nodes
func MakeNodesRequest() *requests.Nodes {
	return &requests.Nodes{
		Env: &requests.Env{
			Dimensions: []string{dimensionFoo},
		},
		Nodes: map[string]*requests.Node{
			"test": {
				ID:         idRoot,
				Dimension:  dimensionFoo,
				MimeTypes:  []string{},
				Expand:     true,
				DataFields: []string{"foo"},
			},
		},
	}
}

// MakeValidURIsRequest URIs reuqest
func MakeValidURIsRequest() *requests.URIs {
	return &requests.URIs{
		Dimension: dimensionFoo,
		IDs:       []string{"id-a", "id-b"},
	}
}

// MakeValidContentRequest a mock content request
func MakeValidContentRequest() *requests.Content {
	dimensions := []string{dimensionFoo}

	return &requests.Content{
		URI: "/a",
		Env: &requests.Env{
			Dimensions: dimensions,
			Groups:     []string{},
		},
		Nodes: map[string]*requests.Node{
			idRoot: {
				ID:         idRoot,
				Dimension:  dimensions[0],
				MimeTypes:  []string{"application/x-node"},
				Expand:     true,
				DataFields: []string{},
			},
		},
	}
}
