package handler

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/foomo/contentserver/pkg/repo"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	httputils "github.com/foomo/keel/utils/net/http"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	HTTP struct {
		l    *zap.Logger
		path string
		repo *repo.Repo
	}
	HTTPOption func(*HTTP)
)

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

// NewHTTP returns a shiny new web server
func NewHTTP(l *zap.Logger, repo *repo.Repo, opts ...HTTPOption) http.Handler {
	inst := &HTTP{
		l:    l.Named("http"),
		path: "/contentserver",
		repo: repo,
	}

	for _, opt := range opts {
		opt(inst)
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func WithPath(v string) HTTPOption {
	return func(o *HTTP) {
		o.path = v
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputils.ServerError(h.l, w, r, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	if r.Body == nil {
		httputils.BadRequestServerError(h.l, w, r, errors.New("empty request body"))
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		httputils.BadRequestServerError(h.l, w, r, errors.Wrap(err, "failed to read incoming request"))
		return
	}

	route := Route(strings.TrimPrefix(r.URL.Path, h.path+"/"))
	if route == RouteGetRepo {
		h.repo.WriteRepoBytes(w)
		w.Header().Set("Content-Type", "application/json")
		return
	}

	reply, errReply := h.handleRequest(h.repo, route, bytes, "webserver")
	if errReply != nil {
		http.Error(w, errReply.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(reply)
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (h *HTTP) handleRequest(r *repo.Repo, handler Route, jsonBytes []byte, source string) ([]byte, error) {
	start := time.Now()

	reply, err := h.executeRequest(r, handler, jsonBytes, source)
	result := "success"
	if err != nil {
		result = "error"
	}

	metrics.ServiceRequestCounter.WithLabelValues(string(handler), result, source).Inc()
	metrics.ServiceRequestDuration.WithLabelValues(string(handler), result, source).Observe(time.Since(start).Seconds())

	return reply, err
}

func (h *HTTP) executeRequest(r *repo.Repo, handler Route, jsonBytes []byte, source string) (replyBytes []byte, err error) {
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
	metrics.ContentRequestCounter.WithLabelValues(source).Inc()

	// handle and process
	switch handler {
	// case HandlerGetRepo: // This case is handled prior to handleRequest being called.
	// since the resulting bytes are written directly in to the http.ResponseWriter / net.Connection
	case RouteGetURIs:
		getURIRequest := &requests.URIs{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &getURIRequest), func() {
			reply = r.GetURIs(getURIRequest.Dimension, getURIRequest.IDs)
		})
	case RouteGetContent:
		contentRequest := &requests.Content{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &contentRequest), func() {
			reply, apiErr = r.GetContent(contentRequest)
		})
	case RouteGetNodes:
		nodesRequest := &requests.Nodes{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &nodesRequest), func() {
			reply = r.GetNodes(nodesRequest)
		})
	case RouteUpdate:
		updateRequest := &requests.Update{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &updateRequest), func() {
			reply = r.Update()
		})
	default:
		reply = responses.NewError(1, "unknown handler: "+string(handler))
	}

	// error handling
	if jsonErr != nil {
		h.l.Error("could not read incoming json", zap.Error(jsonErr))
		reply = responses.NewError(2, "could not read incoming json "+jsonErr.Error())
	} else if apiErr != nil {
		h.l.Error("an API error occurred", zap.Error(apiErr))
		reply = responses.NewError(3, "internal error "+apiErr.Error())
	}

	return h.encodeReply(reply)
}

// encodeReply takes an interface and encodes it as JSON
// it returns the resulting JSON and a marshalling error
func (h *HTTP) encodeReply(reply interface{}) (bytes []byte, err error) {
	bytes, err = json.Marshal(map[string]interface{}{
		"reply": reply,
	})
	if err != nil {
		h.l.Error("could not encode reply", zap.Error(err))
	}
	return
}
