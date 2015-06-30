package repo

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/foomo/contentserver/server/requests"
)

func getMockData(t *testing.T) (server *httptest.Server, varDir string) {

	_, filename, _, _ := runtime.Caller(1)
	mockDir := path.Join(path.Dir(filename), "mock")

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

func assertRepoIsEmpty(t *testing.T, r *Repo, empty bool) {
	if empty {
		if len(r.Directory) > 0 {
			t.Fatal("directory should have been empty, but is not")
		}
	} else {
		if len(r.Directory) == 0 {
			t.Fatal("directory should not have been empty, but it is")
		}
	}
}

func TestLoad404(t *testing.T) {
	mockServer, varDir := getMockData(t)
	server := mockServer.URL + "/repo-no-have"
	r := NewRepo(server, varDir)
	response := r.Update()
	if response.Success {
		t.Fatal("can not get a repo, if the server responds with a 404")
	}
}

func TestLoadBrokenRepo(t *testing.T) {
	mockServer, varDir := getMockData(t)
	server := mockServer.URL + "/repo-broken-json.json"
	r := NewRepo(server, varDir)
	response := r.Update()
	if response.Success {
		t.Fatal("how could we load a broken json")
	}
}

func TestLoadRepo(t *testing.T) {

	mockServer, varDir := getMockData(t)
	server := mockServer.URL + "/repo-ok.json"
	r := NewRepo(server, varDir)
	assertRepoIsEmpty(t, r, true)
	response := r.Update()
	assertRepoIsEmpty(t, r, false)
	if response.Success == false {
		t.Fatal("could not load valid repo")
	}
	if response.Stats.OwnRuntime > response.Stats.RepoRuntime {
		t.Fatal("how could all take less time, than me alone")
	}
	if response.Stats.RepoRuntime < float64(0.05) {
		t.Fatal("the server was too fast")
	}

	// see what happens if we try to start it up again
	nr := NewRepo(server, varDir)
	assertRepoIsEmpty(t, nr, false)
}

func TestLoadRepoDuplicateUris(t *testing.T) {
	mockServer, varDir := getMockData(t)
	server := mockServer.URL + "/repo-duplicate-uris.json"
	r := NewRepo(server, varDir)
	response := r.Update()
	if response.Success {
		t.Fatal("there are duplicates, this repo update should have failed")
	}
	if !strings.Contains(response.ErrorMessage, "update dimension") {
		t.Fatal("error message not as expected")
	}
}

func TestDimensionHygiene(t *testing.T) {
	mockServer, varDir := getMockData(t)
	server := mockServer.URL + "/repo-two-dimensions.json"
	r := NewRepo(server, varDir)
	response := r.Update()
	if !response.Success {
		t.Fatal("well those two dimension should be fine")
	}
	r.server = mockServer.URL + "/repo-ok.json"
	response = r.Update()
	if !response.Success {
		t.Fatal("wtf it is called repo ok")
	}
	if len(r.Directory) != 1 {
		t.Fatal("directory hygiene failed")
	}
}

func TestResolveContent(t *testing.T) {
	mockServer, varDir := getMockData(t)
	server := mockServer.URL + "/repo-two-dimensions.json"
	r := NewRepo(server, varDir)
	response := r.Update()
	if !response.Success {
		t.Fatal("well those two dimension should be fine")
	}
	dimensions := []string{"dimension_foo"}
	contentRequest := &requests.Content{
		URI: "/a",
		Env: &requests.Env{
			Dimensions: dimensions,
			Groups:     []string{},
		},
		Nodes: map[string]*requests.Node{
			"id-root": &requests.Node{
				Id:         "id-root",
				Dimension:  dimensions[0],
				MimeTypes:  []string{"application/x-node"},
				Expand:     true,
				DataFields: []string{},
			},
		},
	}
	siteContent := r.GetContent(contentRequest)
	if siteContent.URI != contentRequest.URI {
		t.Fatal("failed to resolve uri")
	}
}

func TestLinkIds(t *testing.T) {
	mockServer, varDir := getMockData(t)
	server := mockServer.URL + "/repo-link-ok.json"
	r := NewRepo(server, varDir)
	response := r.Update()
	if !response.Success {
		t.Fatal("those links should have been fine")
	}

	r.server = mockServer.URL + "/repo-link-broken.json"
	response = r.Update()

	if response.Success {
		t.Fatal("I do not think so")
	}

}
