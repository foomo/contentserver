package server

import (
	"time"

	"go.uber.org/zap"

	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/status"
	"github.com/prometheus/client_golang/prometheus"
)

func handleRequest(r *repo.Repo, handler Handler, jsonBytes []byte, source string) (replyBytes []byte, err error) {

	var (
		reply             interface{}
		apiErr            error
		jsonErr           error
		start             = time.Now()
		processIfJSONIsOk = func(err error, processingFunc func()) {
			if err != nil {
				jsonErr = err
				return
			}
			processingFunc()
		}
	)

	// handle and process
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
		reply = responses.NewError(1, "unknown handler: "+string(handler))
	}
	addMetrics(metrics, handler, start, jsonErr, apiErr, source)

	// error handling
	if jsonErr != nil {
		Log.Error("could not read incoming json", zap.Error(jsonErr))
		reply = responses.NewError(2, "could not read incoming json "+jsonErr.Error())
	} else if apiErr != nil {
		Log.Error("an API error occured", zap.Error(apiErr))
		reply = responses.NewError(3, "internal error "+apiErr.Error())
	}

	return encodeReply(reply)
}

func addMetrics(metrics *status.Metrics, handlerName Handler, start time.Time, errJSON error, errAPI error, source string) {

	var (
		duration = time.Since(start)
		s        = "succeeded"
	)
	if errJSON != nil || errAPI != nil {
		s = "failed"
	}

	metrics.ServiceRequestCounter.With(prometheus.Labels{
		status.MetricLabelHandler: string(handlerName),
		status.MetricLabelStatus:  s,
		status.MetricLabelSource:  source,
	}).Inc()

	metrics.ServiceRequestDuration.With(prometheus.Labels{
		status.MetricLabelHandler: string(handlerName),
		status.MetricLabelStatus:  s,
		status.MetricLabelSource:  source,
	}).Observe(float64(duration.Seconds()))
}

// encodeReply takes an interface and encodes it as JSON
// it returns the resulting JSON and a marshalling error
func encodeReply(reply interface{}) (replyBytes []byte, err error) {
	replyBytes, err = json.Marshal(map[string]interface{}{
		"reply": reply,
	})
	if err != nil {
		Log.Error("could not encode reply", zap.Error(err))
	}
	return
}
