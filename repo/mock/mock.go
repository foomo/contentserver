package mock

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/foomo/contentserver/requests"
)

// GetMockData mock data to run a repo
func GetMockData(t testing.TB) (server *httptest.Server, varDir string) {

	_, filename, _, _ := runtime.Caller(0)
	mockDir := path.Dir(filename)

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Millisecond * 50)
		mockFilename := path.Join(mockDir, req.URL.Path[1:])
		http.ServeFile(w, req, mockFilename)
	}))
	varDir, err := ioutil.TempDir("", "content-server-test")
	if err != nil {
		panic(err)
	}
	return server, varDir
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
			"id-root": &requests.Node{
				ID:         "id-root",
				Dimension:  dimensions[0],
				MimeTypes:  []string{"application/x-node"},
				Expand:     true,
				DataFields: []string{},
			},
		},
	}

}
