package client

import (
	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/server"
)

// Client a content server client
type Client struct {
	Server string
}

func (c *Client) call(handler string, request interface{}, response interface{}) error {
	return nil
}

// Update tell the server to update itself
func (c *Client) Update() (response *responses.Update, err error) {
	response = &responses.Update{}
	err = c.call(server.HandlerUpdate, &requests.Update{}, response)
	return
}

// GetContent request site content
func (c *Client) GetContent(request *requests.Content) (response *content.SiteContent, err error) {
	response = &content.SiteContent{}
	err = c.call(server.HandlerContent, request, response)
	return
}
