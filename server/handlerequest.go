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

	var reply interface{}
	var apiErr error
	var jsonErr error

	processIfJSONIsOk := func(err error, processingFunc func()) {
		if err != nil {
			jsonErr = err
			return
		}
		processingFunc()
	}

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
	if jsonErr != nil {
		err = jsonErr
		log.Error("  could not read incoming json:", jsonErr)
		errorResponse := responses.NewError(2, "could not read incoming json "+jsonErr.Error())
		reply = errorResponse
	} else if apiErr != nil {
		log.Error("  an API error occured:", apiErr)
		err = apiErr
		reply = responses.NewError(3, "internal error "+apiErr.Error())
	}
	return encodeReply(reply)
}

func encodeReply(reply interface{}) (replyBytes []byte, err error) {
	encodedBytes, jsonReplyErr := json.MarshalIndent(map[string]interface{}{
		"reply": reply,
	}, "", " ")
	if jsonReplyErr != nil {
		err = jsonReplyErr
		log.Error("  could not encode reply " + fmt.Sprint(jsonReplyErr))
	} else {
		replyBytes = encodedBytes
	}
	return replyBytes, err
}
