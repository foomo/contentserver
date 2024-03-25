package client_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/foomo/contentserver/client"
	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/handler"
	"github.com/foomo/contentserver/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const pathContentserver = "/contentserver"

func TestInvalidHTTPClientInit(t *testing.T) {
	c, err := client.NewHTTPClient("")
	assert.Nil(t, c)
	assert.Error(t, err)

	c, err = client.NewHTTPClient("bogus")
	assert.Nil(t, c)
	assert.Error(t, err)

	c, err = client.NewHTTPClient("htt:/notaurl")
	assert.Nil(t, c)
	assert.Error(t, err)

	c, err = client.NewHTTPClient("htts://notaurl")
	assert.Nil(t, c)
	assert.Error(t, err)

	c, err = client.NewHTTPClient("/path/segment/only")
	assert.Nil(t, c)
	assert.Error(t, err)
}

func BenchmarkWebClientAndServerGetContent(b *testing.B) {
	l := zaptest.NewLogger(b)
	server := initHTTPRepoServer(b, l)
	httpClient := newHTTPClient(b, server)
	benchmarkServerAndClientGetContent(b, 30, 100, httpClient)
}

type GetContentClient interface {
	GetContent(ctx context.Context, request *requests.Content) (response *content.SiteContent, err error)
}

func newHTTPClient(tb testing.TB, server *httptest.Server) *client.Client {
	tb.Helper()
	c, err := client.NewHTTPClient(server.URL + pathContentserver)
	require.NoError(tb, err)
	return c
}

func initHTTPRepoServer(tb testing.TB, l *zap.Logger) *httptest.Server {
	tb.Helper()
	r := initRepo(tb, l)
	return httptest.NewServer(handler.NewHTTP(l, r))
}
