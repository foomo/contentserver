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
func (c *Client) Update() (*responses.Update, error) {
	type serverResponse struct {
		Reply *responses.Update
	}
	resp := serverResponse{}
	if err := c.t.call(server.HandlerUpdate, &requests.Update{}, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

// GetContent request site content
func (c *Client) GetContent(request *requests.Content) (*content.SiteContent, error) {
	type serverResponse struct {
		Reply *content.SiteContent
	}
	resp := serverResponse{}
	if err := c.t.call(server.HandlerGetContent, request, &resp); err != nil {
		return nil, err
	}

	return resp.Reply, nil
}

// GetURIs resolve uris for ids in a dimension
func (c *Client) GetURIs(dimension string, IDs []string) (map[string]string, error) {
	type serverResponse struct {
		Reply map[string]string
	}

	resp := serverResponse{}
	if err := c.t.call(server.HandlerGetURIs, &requests.URIs{Dimension: dimension, IDs: IDs}, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

// GetNodes request nodes
func (c *Client) GetNodes(env *requests.Env, nodes map[string]*requests.Node) (map[string]*content.Node, error) {
	r := &requests.Nodes{
		Env:   env,
		Nodes: nodes,
	}
	type serverResponse struct {
		Reply map[string]*content.Node
	}
	resp := serverResponse{}
	if err := c.t.call(server.HandlerGetNodes, r, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

// GetRepo get the whole repo
func (c *Client) GetRepo() (map[string]*content.RepoNode, error) {
	type serverResponse struct {
		Reply map[string]*content.RepoNode
	}
	resp := serverResponse{}
	if err := c.t.call(server.HandlerGetRepo, &requests.Repo{}, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

func (c *Client) ShutDown() {
	c.t.shutdown()
}
