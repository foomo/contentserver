package mock

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"testing"
	"time"
)

// GetMockData mock data to run a repo
func GetMockData(t *testing.T) (server *httptest.Server, varDir string) {

	_, filename, _, _ := runtime.Caller(0)
	mockDir := path.Dir(filename)

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Millisecond * 50)
		mockFilename := path.Join(mockDir, req.URL.Path[1:])
		fmt.Println("----------------------------------->", mockFilename)
		http.ServeFile(w, req, mockFilename)
	}))
	varDir, err := ioutil.TempDir("", "content-server-test")
	if err != nil {
		panic(err)
	}
	return server, varDir
}
