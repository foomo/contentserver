package server

import (
	"context"
	"time"

	"go.uber.org/zap"

	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/status"
)

func handleRequest(ctx context.Context, r *repo.Repo, handler Handler, jsonBytes []byte, source string) (replyBytes []byte, err error) {

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
	status.M.ContentRequestCounter.WithLabelValues(source).Inc()

	// handle and process
	switch handler {
	// case HandlerGetRepo: // This case is handled prior to handleRequest being called.
	// since the resulting bytes are written directly in to the http.ResponseWriter / net.Connection
	case HandlerGetURIs:
		getURIRequest := &requests.URIs{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &getURIRequest), func() {
			reply = r.GetURIs(getURIRequest.Dimension, getURIRequest.IDs)
		})
	case HandlerGetContent:
		contentRequest := &requests.Content{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &contentRequest), func() {
			reply, apiErr = r.GetContent(ctx, contentRequest)
		})
	case HandlerGetNodes:
		nodesRequest := &requests.Nodes{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &nodesRequest), func() {
			reply = r.GetNodes(ctx, nodesRequest)
		})
	case HandlerUpdate:
		updateRequest := &requests.Update{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &updateRequest), func() {
			reply = r.Update()
		})

	default:
		reply = responses.NewError(1, "unknown handler: "+string(handler))
	}
	addMetrics(handler, start, jsonErr, apiErr, source)

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

func addMetrics(handlerName Handler, start time.Time, errJSON error, errAPI error, source string) {

	var (
		duration = time.Since(start)
		s        = "succeeded"
	)
	if errJSON != nil || errAPI != nil {
		s = "failed"
	}

	status.M.ServiceRequestCounter.WithLabelValues(string(handlerName), s, source).Inc()
	status.M.ServiceRequestDuration.WithLabelValues(string(handlerName), s, source).Observe(float64(duration.Seconds()))
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
