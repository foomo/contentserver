package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/server"
)

type serverResponse struct {
	Reply interface{}
}

// Client a content server client
type Client struct {
	Server string
	conn   net.Conn
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
	err = c.call(server.HandlerGetContent, request, response)
	return
}

// GetURIs resolve uris for ids in a dimension
func (c *Client) GetURIs(dimension string, IDs []string) (uriMap map[string]string, err error) {
	uriMap = map[string]string{}
	err = c.call(
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
	err = c.call(server.HandlerGetNodes, r, &nodesResponse)
	return
}

// GetRepo get the whole repo
func (c *Client) GetRepo() (response map[string]*content.RepoNode, err error) {
	response = map[string]*content.RepoNode{}
	err = c.call(server.HandlerGetRepo, &requests.Repo{}, &response)
	return
}

// func (c *Client) closeConnection() error {
// 	if c.conn != nil {
// 		err := c.conn.Close()
// 		if err != nil {
// 			return err
// 		}
// 		c.conn = nil
// 	}
// 	return nil
// }

// func (c *Client) getConnection() (conn net.Conn, err error) {
// 	// we need some pooling here
// 	return
// }

func (c *Client) call(handler server.Handler, request interface{}, response interface{}) error {
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("could not marshal request : %q", err)
	}
	conn, err := net.Dial("tcp", c.Server)
	if err != nil {
		return fmt.Errorf("can not call server - connection error: %q", err)
	}
	defer conn.Close()
	// write header result will be like handler:2{}
	jsonBytes = append([]byte(fmt.Sprintf("%s:%d", handler, len(jsonBytes))), jsonBytes...)

	// send request
	written := 0
	l := len(jsonBytes)
	for written < l {
		n, err := conn.Write(jsonBytes[written:])
		if err != nil {
			return fmt.Errorf("failed to send request: %q", err)
		}
		written += n
	}

	// read response
	responseBytes := []byte{}
	buf := make([]byte, 4096)
	responseLength := 0
	for {
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("an error occured while reading the response: %q", err)
		}
		if n == 0 {
			break
		}
		responseBytes = append(responseBytes, buf[0:n]...)
		if responseLength == 0 {
			for index, byte := range responseBytes {
				if byte == 123 {
					// opening bracket
					responseLength, err = strconv.Atoi(string(responseBytes[0:index]))
					if err != nil {
						return errors.New("could not read response length: " + err.Error())
					}
					responseBytes = responseBytes[index:]
					break
				}
			}
		}
		if responseLength > 0 && len(responseBytes) == responseLength {
			break
		}
	}
	// unmarshal response
	responseJSONErr := json.Unmarshal(responseBytes, &serverResponse{Reply: response})
	if responseJSONErr != nil {
		// is it an error ?
		remoteErr := responses.Error{}
		remoteErrJSONErr := json.Unmarshal(responseBytes, remoteErr)
		if remoteErrJSONErr == nil {
			return remoteErr
		}
		return fmt.Errorf("could not unmarshal response : %q %q", remoteErrJSONErr, string(responseBytes))
	}
	return nil
}
