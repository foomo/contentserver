package handler

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/foomo/contentserver/pkg/repo"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/keel/log"
	httplog "github.com/foomo/keel/net/http/log"
	"github.com/foomo/keel/telemetry"
	"github.com/pkg/errors"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.uber.org/zap"
)

type (
	HTTP struct {
		l        *zap.Logger
		repo     *repo.Repo
		basePath string
	}
	HTTPOption func(*HTTP)
)

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

// NewHTTP returns a shiny new web server
func NewHTTP(l *zap.Logger, repo *repo.Repo, opts ...HTTPOption) http.Handler {
	inst := &HTTP{
		l:        l.Named("http"),
		basePath: "/contentserver",
		repo:     repo,
	}

	for _, opt := range opts {
		opt(inst)
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func WithBasePath(v string) HTTPOption {
	return func(o *HTTP) {
		o.basePath = v
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	route := Route(strings.TrimPrefix(r.URL.Path, h.basePath+"/"))
	if r.Method != http.MethodPost {
		h.rejectRequest(w, r, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		observeRequest(route, "webserver", start, true)

		return
	}

	if r.Body == nil {
		h.rejectRequest(w, r, http.StatusBadRequest, errors.New("empty request body"))
		observeRequest(route, "webserver", start, true)

		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.rejectRequest(w, r, http.StatusBadRequest, errors.Wrap(err, "failed to read incoming request"))
		observeRequest(route, "webserver", start, true)

		return
	}

	if route == RouteGetRepo {
		w.Header().Set("Content-Type", "application/json")

		err := h.repo.WriteRepoBytes(r.Context(), w)
		observeRequest(route, "webserver", start, err != nil)

		if err != nil {
			h.l.Error("failed to write repo bytes", zap.Error(err))
			http.Error(w, "failed to get repo", http.StatusInternalServerError)
		}

		return
	}

	reply, errReply := h.handleRequest(r.Context(), h.repo, route, bytes, "webserver")
	if errReply != nil {
		http.Error(w, errReply.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// reply is produced by encodeReply -> json.Marshal (jsoniter ConfigCompatibleWithStandardLibrary),
	// which HTML-escapes <, >, & by default. Combined with the explicit JSON content type and the
	// nosniff header above, this is safe to write back to the client. gosec G705 cannot follow taint
	// through the third-party JSON encoder and reports a false positive.
	_, _ = w.Write(reply) //nolint:gosec // see comment above
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (h *HTTP) rejectRequest(w http.ResponseWriter, r *http.Request, code int, err error) {
	telemetry.Ctx(r.Context()).RecordError(err)

	if labeler, ok := httplog.LabelerFromRequest(r); ok {
		labeler.Add(log.FErrorType(err), log.FError(errors.Wrap(err, "http server error")))
	} else {
		l := log.WithHTTPRequest(log.WithError(h.l, err), r)
		l.Warn("http server error", log.Attribute(semconv.HTTPResponseStatusCode(code)))
	}

	http.Error(w, http.StatusText(code), code)
}

func (h *HTTP) handleRequest(ctx context.Context, r *repo.Repo, route Route, jsonBytes []byte, source string) ([]byte, error) {
	start := time.Now()

	reply, failed, err := h.executeRequest(ctx, r, route, jsonBytes, source)

	observeRequest(route, source, start, failed || err != nil)

	return reply, err
}

func (h *HTTP) executeRequest(ctx context.Context, r *repo.Repo, route Route, jsonBytes []byte, source string) (replyBytes []byte, failed bool, err error) {
	var (
		reply             any
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
	switch route {
	// case HandlerGetRepo: // This case is handled prior to handleRequest being called.
	// since the resulting bytes are written directly in to the http.ResponseWriter / net.Connection
	case RouteGetURIs:
		getURIRequest := &requests.URIs{}
		processIfJSONIsOk(decodeRequest(jsonBytes, &getURIRequest), func() {
			reply = r.GetURIs(getURIRequest.Dimension, getURIRequest.IDs)
		})
	case RouteGetContent:
		contentRequest := &requests.Content{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &contentRequest), func() {
			reply, apiErr = r.GetContent(contentRequest)
			if contentRequest != nil {
				failed = invalidNodes(contentRequest.Nodes)
			}
		})
	case RouteGetNodes:
		nodesRequest := &requests.Nodes{}
		processIfJSONIsOk(decodeRequest(jsonBytes, &nodesRequest), func() {
			if jsonErr = validateNodesRequest(nodesRequest); jsonErr != nil {
				return
			}

			failed = invalidNodes(nodesRequest.Nodes)
			reply = r.GetNodes(nodesRequest)
		})
	case RouteUpdate:
		updateRequest := &requests.Update{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &updateRequest), func() {
			updateReply := r.Update(ctx)
			failed = !updateReply.Success
			reply = updateReply
		})
	default:
		failed = true

		h.l.Warn("unknown route", zap.String("route", string(route)))
		reply = responses.NewError(1, "unknown route: "+string(route))
	}

	// error handling
	if jsonErr != nil {
		failed = true

		h.l.Warn("could not read incoming json", zap.Error(jsonErr))
		reply = responses.NewError(2, "could not read incoming json "+jsonErr.Error())
	} else if apiErr != nil {
		failed = true

		if repo.IsInvalidRequest(apiErr) {
			h.l.Warn("an API error occurred", zap.Error(apiErr))
		} else {
			h.l.Error("an API error occurred", zap.Error(apiErr))
		}

		reply = responses.NewError(3, "internal error "+apiErr.Error())
	}

	replyBytes, err = h.encodeReply(reply)

	return replyBytes, failed, err
}

// encodeReply takes an interface and encodes it as JSON
// it returns the resulting JSON and a marshalling error
func (h *HTTP) encodeReply(reply any) (bytes []byte, err error) {
	bytes, err = json.Marshal(map[string]any{
		"reply": reply,
	})
	if err != nil {
		h.l.Error("could not encode reply", zap.Error(err))
	}

	return
}
