package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/foomo/contentserver/requests"
	"go.uber.org/zap"

	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/foomo/contentserver/pkg/repo"
	"github.com/foomo/contentserver/responses"
)

const sourceSocketServer = "socketserver"

type Socket struct {
	l    *zap.Logger
	repo *repo.Repo
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

// NewSocket returns a shiny new socket server
func NewSocket(l *zap.Logger, repo *repo.Repo) *Socket {
	inst := &Socket{
		l:    l.Named("socket"),
		repo: repo,
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (h *Socket) Serve(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				if !errors.Is(err, io.EOF) {
					h.l.Error("panic in handle connection", zap.Error(err))
				}
			} else {
				h.l.Error("panic in handle connection", zap.String("error", fmt.Sprint(r)))
			}
		}
	}()

	// h.l.Debug("socketServer.handleConnection")
	metrics.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Inc()

	var (
		headerBuffer [1]byte
		header       = ""
	)
	for {
		// let us read with 1 byte steps on conn until we find "{"
		_, readErr := conn.Read(headerBuffer[0:])
		if errors.Is(readErr, io.EOF) {
			// client closed the connection
			metrics.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()
			return
		} else if readErr != nil {
			h.l.Error("failed to read from connection", zap.Error(readErr))
			return
		}
		// read next byte
		current := headerBuffer[0:]
		if string(current) == "{" {
			start := time.Now()
			// json has started
			handler, jsonLength, headerErr := h.extractHandlerAndJSONLentgh(header)
			// reset header
			header = ""

			if headerErr != nil {
				h.l.Warn("invalid request could not read header", zap.Error(headerErr))
				observeRequest(handler, sourceSocketServer, start, true)

				encodedErr, encodingErr := h.encodeReply(responses.NewError(4, "invalid header "+headerErr.Error()))
				if encodingErr == nil {
					h.writeResponse(conn, encodedErr)
				} else {
					h.l.Error("could not respond to invalid request", zap.Error(encodingErr))
				}

				return
			}

			h.l.Debug("found json", zap.Int("length", jsonLength))

			if jsonLength > 0 {
				var (
					// let us try to read some json
					jsonBytes         = make([]byte, jsonLength)
					jsonLengthCurrent = 1
					readRound         = 0
				)

				// that is "{"
				jsonBytes[0] = 123

				for jsonLengthCurrent < jsonLength {
					readRound++

					readLength, jsonReadErr := conn.Read(jsonBytes[jsonLengthCurrent:jsonLength])
					if jsonReadErr != nil {
						// @fixme we need to force a read timeout (SetReadDeadline?), if expected jsonLength is lower than really sent bytes (e.g. if client implements protocol wrong)
						// @todo should we check for io.EOF here
						h.l.Warn("could not read json - giving up with this client connection", zap.Error(jsonReadErr))
						observeRequest(handler, sourceSocketServer, start, true)
						metrics.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()

						return
					}

					jsonLengthCurrent += readLength
					h.l.Debug("read cycle status",
						zap.Int("jsonLengthCurrent", jsonLengthCurrent),
						zap.Int("jsonLength", jsonLength),
						zap.Int("readRound", readRound),
					)
				}

				h.l.Debug("read json", zap.Int("length", len(jsonBytes)))

				h.writeResponse(conn, h.execute(handler, jsonBytes))
				// note: connection remains open
				continue
			}

			h.l.Warn("can not read empty json")
			observeRequest(handler, sourceSocketServer, start, true)
			metrics.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()

			return
		}
		// adding to header byte by byte
		header += string(headerBuffer[0:])
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (h *Socket) extractHandlerAndJSONLentgh(header string) (route Route, jsonLength int, err error) {
	headerParts := strings.Split(header, ":")
	if len(headerParts) != 2 {
		return "", 0, errors.New("invalid header")
	}

	jsonLength, err = strconv.Atoi(headerParts[1])
	if err != nil {
		err = fmt.Errorf("could not parse length in header: %q", header)
	}

	return Route(headerParts[0]), jsonLength, err
}

func (h *Socket) execute(route Route, jsonBytes []byte) (reply []byte) {
	h.l.Debug("incoming json buffer", zap.Int("length", len(jsonBytes)))

	if route == RouteGetRepo {
		start := time.Now()

		var b bytes.Buffer

		err := h.repo.WriteRepoBytes(context.Background(), &b)
		observeRequest(route, sourceSocketServer, start, err != nil)

		if err != nil {
			h.l.Error("failed to write repo bytes", zap.Error(err))
			errorReply, _ := h.encodeReply(responses.NewError(5, "failed to get repo: "+err.Error()))

			return errorReply
		}

		return b.Bytes()
	}

	reply, handlingError := h.handleRequest(h.repo, route, jsonBytes, sourceSocketServer)
	if handlingError != nil {
		h.l.Error("socketServer.execute failed", zap.Error(handlingError))
	}

	return reply
}

func (h *Socket) writeResponse(conn net.Conn, reply []byte) {
	headerBytes := []byte(strconv.Itoa(len(reply)))
	reply = append(headerBytes, reply...)
	h.l.Debug("replying", zap.String("reply", string(reply)))

	n, writeError := conn.Write(reply)
	if writeError != nil {
		h.l.Error("socketServer.writeResponse: could not write reply", zap.Error(writeError))
		return
	}

	if n < len(reply) {
		h.l.Error("socketServer.writeResponse: write too short",
			zap.Int("got", n),
			zap.Int("expected", len(reply)),
		)

		return
	}

	h.l.Debug("replied. waiting for next request on open connection")
}

func (h *Socket) handleRequest(r *repo.Repo, route Route, jsonBytes []byte, source string) ([]byte, error) {
	start := time.Now()

	reply, failed, err := h.executeRequest(r, route, jsonBytes, source)

	observeRequest(route, source, start, failed || err != nil)

	return reply, err
}

func (h *Socket) executeRequest(r *repo.Repo, route Route, jsonBytes []byte, source string) (replyBytes []byte, failed bool, err error) {
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
	// case RouteGetRepo: // This case is handled prior to handleRequest being called.
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
			updateReply := r.Update(context.Background())
			failed = !updateReply.Success
			reply = updateReply
		})

	default:
		failed = true

		h.l.Warn("unknown handler", zap.String("route", string(route)))
		reply = responses.NewError(1, "unknown handler: "+string(route))
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
func (h *Socket) encodeReply(reply any) (replyBytes []byte, err error) {
	replyBytes, err = json.Marshal(map[string]any{
		"reply": reply,
	})
	if err != nil {
		h.l.Error("could not encode reply", zap.Error(err))
	}

	return
}
