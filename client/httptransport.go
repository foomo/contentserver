package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/foomo/contentserver/pkg/handler"
	"github.com/foomo/contentserver/pkg/utils"
)

type (
	HTTPTransport struct {
		httpClient *http.Client
		endpoint   string
	}
	HTTPTransportOption func(*HTTPTransport)
)

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

// NewHTTPTransport will create a new http transport for the given server and client.
// Caution: the provided server url is not validated!
func NewHTTPTransport(server string, opts ...HTTPTransportOption) *HTTPTransport {
	inst := &HTTPTransport{
		endpoint:   server,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(inst)
	}

	return inst
}

// NewHTTPClient constructs a new client to talk to the contentserver.
// It returns an error if the provided url is empty or invalid.
func NewHTTPClient(url string) (c *Client, err error) {
	if url == "" {
		return nil, ErrEmptyServerURL
	}

	// validate url
	if !utils.IsValidUrl(url) {
		return nil, ErrInvalidServerURL
	}

	return New(NewHTTPTransport(url)), nil
}

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func HTTPTransportWithHTTPClient(v *http.Client) HTTPTransportOption {
	return func(o *HTTPTransport) {
		o.httpClient = v
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (t *HTTPTransport) Call(ctx context.Context, route handler.Route, request interface{}, response interface{}) error {
	requestBytes, errMarshal := json.Marshal(request)
	if errMarshal != nil {
		return errMarshal
	}
	req, errNewRequest := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		t.endpoint+"/"+string(route),
		bytes.NewBuffer(requestBytes),
	)
	if errNewRequest != nil {
		return errNewRequest
	}
	httpResponse, errDo := t.httpClient.Do(req)
	if errDo != nil {
		return errDo
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return errors.New("non 200 reply")
	}
	if httpResponse.Body == nil {
		return errors.New("empty response body")
	}
	responseBytes, errRead := io.ReadAll(httpResponse.Body)
	if errRead != nil {
		return errRead
	}
	return json.Unmarshal(responseBytes, response)
}

func (t *HTTPTransport) Close() {
	// nothing to do here
}
