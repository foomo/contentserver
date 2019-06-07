package server

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"go.uber.org/zap"
)

// Test if we still maintain the contract with the upstream package
var _ repoer = (*repo.Repo)(nil)
var _ repoer = (*mockRepoer)(nil)

type mockRepoer struct {
	getContentErr error
}

func (mockRepoer) GetURIs(dimension string, ids []string) map[string]string {
	return map[string]string{"uri1": "value1"}
}

func (m mockRepoer) GetContent(*requests.Content) (*content.SiteContent, error) {
	if m.getContentErr != nil {
		return nil, m.getContentErr
	}
	return &content.SiteContent{
		URI: "uri1",
	}, nil
}

func (mockRepoer) GetNodes(*requests.Nodes) map[string]*content.Node {
	return map[string]*content.Node{
		"node1": {
			Index: []string{"index1"},
		},
	}
}

func (mockRepoer) Update() *responses.Update {
	return &responses.Update{Success: true}
}

func (mockRepoer) WriteRepoBytes(io.Writer) {}

func objectToJsonReader(t *testing.T, o interface{}) io.Reader {
	data, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(data)
}

func TestHandleRequest_WebServer(t *testing.T) {
	logger.Log = zap.New(nil)

	runTest := func(handler Handler, reqObject interface{}, want string) func(*testing.T) {
		return func(t *testing.T) {
			rdr := objectToJsonReader(t, reqObject)
			var wr bytes.Buffer
			if err := handleRequest(mockRepoer{}, handler, rdr, &wr, "webserver"); err != nil {
				t.Error(err)
			}
			if have := wr.String(); have != want {
				t.Errorf("\nhave: %q\nwant: %q", have, want)
			}
		}
	}

	t.Run("unknown handler", runTest(
		"xx",
		new(requests.URIs),
		`{"reply":{"status":500,"code":1,"message":"unknown handler: xx"}}`+"\n"),
	)

	t.Run("HandlerGetURIs", runTest(
		HandlerGetURIs,
		&requests.URIs{
			IDs:       []string{"a"},
			Dimension: "dim1",
		},
		"{\"reply\":{\"uri1\":\"value1\"}}\n"),
	)

	t.Run("HandlerGetContent Ok", runTest(
		HandlerGetContent,
		&requests.Content{
			URI:        "uri2",
			DataFields: []string{"field1"},
		},
		"{\"reply\":{\"status\":0,\"URI\":\"uri1\",\"dimension\":\"\",\"mimeType\":\"\",\"item\":null,\"data\":null,\"path\":null,\"URIs\":null,\"nodes\":null}}\n"),
	)

	t.Run("HandlerGetContent Err", func(t *testing.T) {
		rdr := objectToJsonReader(t, &requests.Content{
			URI:        "uri2",
			DataFields: []string{"field1"},
		}, )
		var wr bytes.Buffer
		if err := handleRequest(mockRepoer{
			getContentErr: errors.New("upssss an error"),
		}, HandlerGetContent, rdr, &wr, "webserver"); err != nil {
			t.Error(err)
		}
		const want = "{\"reply\":{\"status\":500,\"code\":3,\"message\":\"internal error: upssss an error\"}}\n"
		if have := wr.String(); have != want {
			t.Errorf("\nhave: %q\nwant: %q", have, want)
		}
	})

	t.Run("HandlerGetNodes", runTest(
		HandlerGetNodes,
		&requests.Nodes{
			Nodes: map[string]*requests.Node{
				"node1": {
					ID: "id11",
				},
			},
		},
		"{\"reply\":{\"node1\":{\"item\":null,\"nodes\":null,\"index\":[\"index1\"]}}}\n"),
	)

	t.Run("HandlerUpdate", runTest(
		HandlerUpdate,
		requests.Update{},
		"{\"reply\":{\"success\":true,\"errorMessage\":\"\",\"stats\":{\"numberOfNodes\":0,\"numberOfURIs\":0,\"repoRuntime\":0,\"ownRuntime\":0}}}\n"),
	)

}
