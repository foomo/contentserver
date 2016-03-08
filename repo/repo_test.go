package repo

import (
	"strings"
	"testing"

	"github.com/foomo/contentserver/repo/mock"
	"github.com/foomo/contentserver/requests"
)

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
	mockServer, varDir := mock.GetMockData(t)
	server := mockServer.URL + "/repo-no-have"
	r := NewRepo(server, varDir)
	response := r.Update()
	if response.Success {
		t.Fatal("can not get a repo, if the server responds with a 404")
	}
}

func TestLoadBrokenRepo(t *testing.T) {
	mockServer, varDir := mock.GetMockData(t)
	server := mockServer.URL + "/repo-broken-json.json"
	r := NewRepo(server, varDir)
	response := r.Update()
	if response.Success {
		t.Fatal("how could we load a broken json")
	}
}

func TestLoadRepo(t *testing.T) {

	mockServer, varDir := mock.GetMockData(t)
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
	mockServer, varDir := mock.GetMockData(t)
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
	mockServer, varDir := mock.GetMockData(t)
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

func getTestRepo(path string, t *testing.T) *Repo {
	mockServer, varDir := mock.GetMockData(t)
	server := mockServer.URL + path
	r := NewRepo(server, varDir)
	response := r.Update()
	if !response.Success {
		t.Fatal("well those two dimension should be fine")
	}
	return r
}

func TestResolveContent(t *testing.T) {
	r := getTestRepo("/repo-two-dimensions.json", t)

	contentRequest := makeValidRequest()

	siteContent, err := r.GetContent(contentRequest)
	if siteContent.URI != contentRequest.URI {
		t.Fatal("failed to resolve uri")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinkIds(t *testing.T) {
	mockServer, varDir := mock.GetMockData(t)
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

func makeValidRequest() *requests.Content {
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

func TestInvalidRequest(t *testing.T) {

	r := getTestRepo("/repo-two-dimensions.json", t)

	if r.validateContentRequest(makeValidRequest()) != nil {
		t.Fatal("failed validation a valid request")
	}

	tests := map[string]*requests.Content{}

	rEmptyURI := makeValidRequest()
	rEmptyURI.URI = ""
	tests["empty uri"] = rEmptyURI

	rEmptyEnv := makeValidRequest()
	rEmptyEnv.Env = nil
	tests["empty env"] = rEmptyEnv

	rEmptyEnvDimensions := makeValidRequest()
	rEmptyEnvDimensions.Env.Dimensions = []string{}
	tests["empty env dimensions"] = rEmptyEnvDimensions

	//rNodesValidID := makeValidRequest()
	//rNodesValidID.Nodes["id-root"].Id = ""
	//tests["nodes must have a valid id"] = rNodesValidID

	for comment, req := range tests {
		if r.validateContentRequest(req) == nil {
			t.Fatal(comment, "should have failed")
		}
	}
}