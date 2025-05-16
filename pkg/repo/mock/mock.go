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

// GetMockData mock data to run a repo
func GetMockData(tb testing.TB) (*httptest.Server, string) {
	tb.Helper()
	_, filename, _, _ := runtime.Caller(0)
	mockDir := path.Dir(filename)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Millisecond * 50)
		mockFilename := path.Join(mockDir, req.URL.Path[1:])
		http.ServeFile(w, req, mockFilename)
	}))

	return server, tb.TempDir()
}

// MakeNodesRequest a request to get some nodes
func MakeNodesRequest() *requests.Nodes {
	return &requests.Nodes{
		Env: &requests.Env{
			Dimensions: []string{"dimension_foo"},
		},
		Nodes: map[string]*requests.Node{
			"test": {
				ID:         "id-root",
				Dimension:  "dimension_foo",
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
		Dimension: "dimension_foo",
		IDs:       []string{"id-a", "id-b"},
	}
}

// MakeValidContentRequest a mock content request
func MakeValidContentRequest() *requests.Content {
	dimensions := []string{"dimension_foo"}
	return &requests.Content{
		URI: "/a",
		Env: &requests.Env{
			Dimensions: dimensions,
			Groups:     []string{},
		},
		Nodes: map[string]*requests.Node{
			"id-root": {
				ID:         "id-root",
				Dimension:  dimensions[0],
				MimeTypes:  []string{"application/x-node"},
				Expand:     true,
				DataFields: []string{},
			},
		},
	}
}
