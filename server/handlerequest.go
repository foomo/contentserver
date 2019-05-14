package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/status"
	"github.com/prometheus/client_golang/prometheus"
)

func handleRequest(r *repo.Repo, handler Handler, jsonBytes []byte, metrics *status.Metrics) (replyBytes []byte, err error) {

	// variables
	var reply interface{}
	var apiErr error
	var jsonErr error

	start := time.Now()

	// helper processor
	processIfJSONIsOk := func(err error, processingFunc func()) {
		if err != nil {
			jsonErr = err
			return
		}
		processingFunc()
	}

	// handle and process
	switch handler {
	case HandlerGetURIs:
		getURIRequest := &requests.URIs{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &getURIRequest), func() {
			reply = r.GetURIs(getURIRequest.Dimension, getURIRequest.IDs)
		})
		addMetrics(metrics, HandlerGetURIs, start, jsonErr, apiErr)
	case HandlerGetContent:
		contentRequest := &requests.Content{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &contentRequest), func() {
			reply, apiErr = r.GetContent(contentRequest)
		})
		addMetrics(metrics, HandlerGetContent, start, jsonErr, apiErr)
	case HandlerGetNodes:
		nodesRequest := &requests.Nodes{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &nodesRequest), func() {
			reply = r.GetNodes(nodesRequest)
		})
		addMetrics(metrics, HandlerGetNodes, start, jsonErr, apiErr)
	case HandlerUpdate:
		updateRequest := &requests.Update{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &updateRequest), func() {
			reply = r.Update()
		})
		addMetrics(metrics, HandlerUpdate, start, jsonErr, apiErr)
	case HandlerGetRepo:
		repoRequest := &requests.Repo{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &repoRequest), func() {
			reply = r.GetRepo()
		})
		addMetrics(metrics, HandlerGetRepo, start, jsonErr, apiErr)
	default:
		err = errors.New(log.Error("  can not handle this one " + handler))
		errorResponse := responses.NewError(1, "unknown handler")
		reply = errorResponse
		addMetrics(metrics, "default", start, jsonErr, apiErr)
	}

	// error handling
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

func addMetrics(metrics *status.Metrics, handlerName Handler, start time.Time, errJSON error, errAPI error) {

	duration := time.Since(start)

	s := "succeeded"
	if errJSON != nil || errAPI != nil {
		s = "failed"
	}

	metrics.ServiceRequestCounter.With(prometheus.Labels{
		status.MetricLabelHandler: string(handlerName),
		status.MetricLabelStatus:  s,
	}).Inc()

	metrics.ServiceRequestDuration.With(prometheus.Labels{
		status.MetricLabelHandler: string(handlerName),
		status.MetricLabelStatus:  s,
	}).Observe(float64(duration.Nanoseconds()))
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
