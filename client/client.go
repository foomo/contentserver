package client

import (
	"context"
	"errors"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/handler"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
)

var (
	ErrEmptyServerURL   = errors.New("empty contentserver url provided")
	ErrInvalidServerURL = errors.New("invalid contentserver url provided")
)

// Client a content server client
type Client struct {
	t Transport
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func New(transport Transport) *Client {
	return &Client{
		t: transport,
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

// Update tell the server to update itself
func (c *Client) Update(ctx context.Context) (*responses.Update, error) {
	type serverResponse struct {
		Reply *responses.Update
	}
	resp := serverResponse{}
	if err := c.t.Call(ctx, handler.RouteUpdate, &requests.Update{}, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

// GetContent request site content
func (c *Client) GetContent(ctx context.Context, request *requests.Content) (*content.SiteContent, error) {
	type serverResponse struct {
		Reply *content.SiteContent
	}
	resp := serverResponse{}
	if err := c.t.Call(ctx, handler.RouteGetContent, request, &resp); err != nil {
		return nil, err
	}

	return resp.Reply, nil
}

// GetURIs resolve uris for ids in a dimension
func (c *Client) GetURIs(ctx context.Context, dimension string, ids []string) (map[string]string, error) {
	type serverResponse struct {
		Reply map[string]string
	}

	resp := serverResponse{}
	if err := c.t.Call(ctx, handler.RouteGetURIs, &requests.URIs{Dimension: dimension, IDs: ids}, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

// GetNodes request nodes
func (c *Client) GetNodes(ctx context.Context, env *requests.Env, nodes map[string]*requests.Node) (map[string]*content.Node, error) {
	r := &requests.Nodes{
		Env:   env,
		Nodes: nodes,
	}
	type serverResponse struct {
		Reply map[string]*content.Node
	}
	resp := serverResponse{}
	if err := c.t.Call(ctx, handler.RouteGetNodes, r, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

// GetRepo get the whole repo
func (c *Client) GetRepo(ctx context.Context) (map[string]*content.RepoNode, error) {
	type serverResponse struct {
		Reply map[string]*content.RepoNode
	}
	resp := serverResponse{}
	if err := c.t.Call(ctx, handler.RouteGetRepo, &requests.Repo{}, &resp); err != nil {
		return nil, err
	}
	return resp.Reply, nil
}

func (c *Client) Close() {
	c.t.Close()
}
