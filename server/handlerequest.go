package server

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
)

func handleRequest(r *repo.Repo, handler Handler, jsonBytes []byte) (replyBytes []byte, err error) {

	var (
		reply             interface{}
		apiErr            error
		jsonErr           error
		processIfJSONIsOk = func(err error, processingFunc func()) {
			if err != nil {
				jsonErr = err
				return
			}
			processingFunc()
		}
	)

	switch handler {
	case HandlerGetURIs:
		getURIRequest := &requests.URIs{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &getURIRequest), func() {
			reply = r.GetURIs(getURIRequest.Dimension, getURIRequest.IDs)
		})
	case HandlerGetContent:
		contentRequest := &requests.Content{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &contentRequest), func() {
			reply, apiErr = r.GetContent(contentRequest)
		})
	case HandlerGetNodes:
		nodesRequest := &requests.Nodes{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &nodesRequest), func() {
			reply = r.GetNodes(nodesRequest)
		})
	case HandlerUpdate:
		updateRequest := &requests.Update{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &updateRequest), func() {
			reply = r.Update()
		})
	case HandlerGetRepo:
		repoRequest := &requests.Repo{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &repoRequest), func() {
			reply = r.GetRepo()
		})
	default:
		err = errors.New(log.Error("  can not handle this one " + handler))
		errorResponse := responses.NewError(1, "unknown handler")
		reply = errorResponse
	}

	// check for errors and set reply
	if jsonErr != nil {
		log.Error("  could not read incoming json:", jsonErr)
		err = jsonErr
		reply = responses.NewError(2, "could not read incoming json "+jsonErr.Error())
	} else if apiErr != nil {
		log.Error("  an API error occured:", apiErr)
		err = apiErr
		reply = responses.NewError(3, "internal error "+apiErr.Error())
	}

	return encodeReply(reply)
}

// encodeReply takes an interface and encodes it as JSON
// it returns the resulting JSON and a marshalling error
func encodeReply(reply interface{}) (replyBytes []byte, err error) {

	// @TODO: why use marshal indent here???
	replyBytes, err = json.MarshalIndent(map[string]interface{}{
		"reply": reply,
	}, "", " ")
	if err != nil {
		log.Error("  could not encode reply " + fmt.Sprint(err))
	}
	return
}
