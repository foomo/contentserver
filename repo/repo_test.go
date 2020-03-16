package repo

import (
	"strings"
	"testing"
	"time"

	. "github.com/foomo/contentserver/logger"
	_ "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo/mock"
	"github.com/foomo/contentserver/requests"
	"github.com/stretchr/testify/require"
)

func init() {
	SetupLogging(true, "contentserver_repo_test.log")
}

func NewTestRepo(server, varDir string) *Repo {

	r := NewRepo(server, varDir)

	// because the travis CI VMs are very slow,
	// we need to add some delay to allow the server to startup
	time.Sleep(1 * time.Second)

	return r
}

func assertRepoIsEmpty(t *testing.T, r *Repo, empty bool) {
	if empty {
		if len(r.Directory) > 0 {
			t.Fatal("directory should have been empty, but is not")
		}
	} else {
		if len(r.Directory) == 0 {
			t.Fatal("directory is empty, but should have been not")
		}
	}
}

func TestLoad404(t *testing.T) {
	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-no-have"
		r                  = NewTestRepo(server, varDir)
	)

	response := r.Update()
	if response.Success {
		t.Fatal("can not get a repo, if the server responds with a 404")
	}
}

func TestLoadBrokenRepo(t *testing.T) {
	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-broken-json.json"
		r                  = NewTestRepo(server, varDir)
	)

	response := r.Update()
	if response.Success {
		t.Fatal("how could we load a broken json")
	}
}

func TestLoadRepo(t *testing.T) {

	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-ok.json"
		r                  = NewTestRepo(server, varDir)
	)
	assertRepoIsEmpty(t, r, true)

	response := r.Update()
	assertRepoIsEmpty(t, r, false)

	if !response.Success {
		t.Fatal("could not load valid repo")
	}
	if response.Stats.OwnRuntime > response.Stats.RepoRuntime {
		t.Fatal("how could all take less time, than me alone")
	}
	if response.Stats.RepoRuntime < float64(0.05) {
		t.Fatal("the server was too fast")
	}

	// see what happens if we try to start it up again
	nr := NewTestRepo(server, varDir)
	assertRepoIsEmpty(t, nr, false)
}

func BenchmarkLoadRepo(b *testing.B) {

	var (
		t                  = &testing.T{}
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-ok.json"
		r                  = NewTestRepo(server, varDir)
	)
	if len(r.Directory) > 0 {
		b.Fatal("directory should have been empty, but is not")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		response := r.Update()
		if len(r.Directory) == 0 {
			b.Fatal("directory is empty, but should have been not")
		}

		if !response.Success {
			b.Fatal("could not load valid repo")
		}
	}
}

func TestLoadRepoDuplicateUris(t *testing.T) {

	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-duplicate-uris.json"
		r                  = NewTestRepo(server, varDir)
	)

	response := r.Update()
	if response.Success {
		t.Fatal("there are duplicates, this repo update should have failed")
	}
	if !strings.Contains(response.ErrorMessage, "update dimension") {
		t.Fatal("error message not as expected: " + response.ErrorMessage)
	}
}

func TestDimensionHygiene(t *testing.T) {

	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-two-dimensions.json"
		r                  = NewTestRepo(server, varDir)
	)

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

	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + path
		r                  = NewTestRepo(server, varDir)
		response           = r.Update()
	)
	if !response.Success {
		t.Fatal("well those two dimension should be fine")
	}
	return r
}

func TestGetNodes(t *testing.T) {
	var (
		r            = getTestRepo("/repo-two-dimensions.json", t)
		nodesRequest = mock.MakeNodesRequest()
		nodes        = r.GetNodes(nodesRequest)
		testNode, ok = nodes["test"]
	)
	if !ok {
		t.Fatal("wtf that should be a node")
	}
	testData, ok := testNode.Item.Data["foo"]
	t.Log("testData", testData)
	if !ok {
		t.Fatal("failed to fetch test data")
	}
}

func TestGetNodesExposeHidden(t *testing.T) {
	var (
		r            = getTestRepo("/repo-ok-exposehidden.json", t)
		nodesRequest = mock.MakeNodesRequest()
	)
	nodesRequest.Nodes["test"].ExposeHiddenNodes = true
	nodes := r.GetNodes(nodesRequest)
	testNode, ok := nodes["test"]
	if !ok {
		t.Fatal("wtf that should be a node")
	}
	_, ok = testNode.Item.Data["foo"]
	if !ok {
		t.Fatal("failed to fetch test data")
	}
	require.Equal(t, 2, len(testNode.Nodes))
}
func TestResolveContent(t *testing.T) {

	var (
		r                = getTestRepo("/repo-two-dimensions.json", t)
		contentRequest   = mock.MakeValidContentRequest()
		siteContent, err = r.GetContent(contentRequest)
	)

	if siteContent.URI != contentRequest.URI {
		t.Fatal("failed to resolve uri")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinkIds(t *testing.T) {

	var (
		mockServer, varDir = mock.GetMockData(t)
		server             = mockServer.URL + "/repo-link-ok.json"
		r                  = NewTestRepo(server, varDir)
		response           = r.Update()
	)

	if !response.Success {
		t.Fatal("those links should have been fine")
	}

	r.server = mockServer.URL + "/repo-link-broken.json"
	response = r.Update()

	if response.Success {
		t.Fatal("I do not think so")
	}

}

func TestInvalidRequest(t *testing.T) {

	r := getTestRepo("/repo-two-dimensions.json", t)

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

	//rNodesValidID := mock.MakeValidContentRequest()
	//rNodesValidID.Nodes["id-root"].Id = ""
	//tests["nodes must have a valid id"] = rNodesValidID

	for comment, req := range tests {
		if r.validateContentRequest(req) == nil {
			t.Fatal(comment, "should have failed")
		}
	}
}
