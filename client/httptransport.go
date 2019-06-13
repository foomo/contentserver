package client

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/foomo/contentserver/server"
)

type httpTransport struct {
	client   *http.Client
	endpoint string
}

func newHTTPTransport(server string) transport {
	return &httpTransport{
		endpoint: server,
		client:   http.DefaultClient,
	}
}

func (ht *httpTransport) shutdown() {
	// nothing to do here
}

func (ht *httpTransport) call(handler server.Handler, request interface{}, response interface{}) error {
	requestBytes, errMarshal := json.Marshal(request)
	if errMarshal != nil {
		return errMarshal
	}
	req, errNewRequest := http.NewRequest(
		http.MethodPost,
		ht.endpoint+"/"+string(handler),
		bytes.NewReader(requestBytes),
	)
	if errNewRequest != nil {
		return errNewRequest
	}
	httpResponse, errDo := ht.client.Do(req)
	if errDo != nil {
		return errDo
	}
	if httpResponse.StatusCode != http.StatusOK {
		return errors.New("non 200 reply")
	}
	if httpResponse.Body == nil {
		return errors.New("empty response body")
	}
	defer httpResponse.Body.Close()
	return json.NewDecoder(httpResponse.Body).Decode(&serverResponse{Reply: response})
}
