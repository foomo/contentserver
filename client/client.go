package client

import (
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/server"
)

// Client a content server client
type Client struct {
	t transport
}

func NewClient(
	server string,
	connectionPoolSize int,
	waitTimeout time.Duration,
) (c *Client, err error) {
	c = &Client{
		t: newSocketTransport(server, connectionPoolSize, waitTimeout),
	}
	return
}

func NewHTTPClient(server string) (c *Client, err error) {
	c = &Client{
		t: newHTTPTransport(server),
	}
	return
}

// Update tell the server to update itself
func (c *Client) Update() (response *responses.Update, err error) {
	response = &responses.Update{}
	err = c.t.call(server.HandlerUpdate, &requests.Update{}, response)
	return
}

// GetContent request site content
func (c *Client) GetContent(request *requests.Content) (response *content.SiteContent, err error) {
	response = &content.SiteContent{}
	err = c.t.call(server.HandlerGetContent, request, response)
	return
}

// GetURIs resolve uris for ids in a dimension
func (c *Client) GetURIs(dimension string, IDs []string) (uriMap map[string]string, err error) {
	uriMap = map[string]string{}
	err = c.t.call(
		server.HandlerGetURIs,
		&requests.URIs{
			Dimension: dimension,
			IDs:       IDs,
		},
		&uriMap,
	)
	return
}

// GetNodes request nodes
func (c *Client) GetNodes(env *requests.Env, nodes map[string]*requests.Node) (nodesResponse map[string]*content.Node, err error) {
	r := &requests.Nodes{
		Env:   env,
		Nodes: nodes,
	}
	nodesResponse = map[string]*content.Node{}
	err = c.t.call(server.HandlerGetNodes, r, &nodesResponse)
	return
}

// GetRepo get the whole repo
func (c *Client) GetRepo() (response map[string]*content.RepoNode, err error) {
	response = map[string]*content.RepoNode{}
	err = c.t.call(server.HandlerGetRepo, &requests.Repo{}, &response)
	return
}

func (c *Client) ShutDown() {
	c.t.shutdown()
}
