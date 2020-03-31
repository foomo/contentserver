package client

import (
	"bytes"
	"errors"
	"io/ioutil"
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
		bytes.NewBuffer(requestBytes),
	)
	if errNewRequest != nil {
		return errNewRequest
	}
	httpResponse, errDo := ht.client.Do(req)
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
	responseBytes, errRead := ioutil.ReadAll(httpResponse.Body)
	if errRead != nil {
		return errRead
	}
	return json.Unmarshal(responseBytes, response)
}
