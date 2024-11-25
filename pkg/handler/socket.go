package handler

import (
	"bytes"
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

	h.l.Debug("socketServer.handleConnection")
	metrics.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Inc()

	var (
		headerBuffer [1]byte
		header       = ""
		i            = 0
	)
	for {
		i++
		// fmt.Println("---->", i)
		// let us read with 1 byte steps on conn until we find "{"
		_, readErr := conn.Read(headerBuffer[0:])
		if readErr != nil {
			h.l.Debug("looks like the client closed the connection", zap.Error(readErr))
			metrics.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()
			return
		}
		// read next byte
		current := headerBuffer[0:]
		if string(current) == "{" {
			// json has started
			handler, jsonLength, headerErr := h.extractHandlerAndJSONLentgh(header)
			// reset header
			header = ""
			if headerErr != nil {
				h.l.Error("invalid request could not read header", zap.Error(headerErr))
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
						h.l.Error("could not read json - giving up with this client connection", zap.Error(jsonReadErr))
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
			h.l.Error("can not read empty json")
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
		var (
			b bytes.Buffer
		)
		h.repo.WriteRepoBytes(&b)
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

	reply, err := h.executeRequest(r, route, jsonBytes, source)
	result := "success"
	if err != nil {
		result = "error"
	}

	metrics.ServiceRequestCounter.WithLabelValues(string(route), result, source).Inc()
	metrics.ServiceRequestDuration.WithLabelValues(string(route), result, source).Observe(time.Since(start).Seconds())

	return reply, err
}

func (h *Socket) executeRequest(r *repo.Repo, route Route, jsonBytes []byte, source string) (replyBytes []byte, err error) {
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
	switch route {
	// case RouteGetRepo: // This case is handled prior to handleRequest being called.
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
		reply = responses.NewError(1, "unknown handler: "+string(route))
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
func (h *Socket) encodeReply(reply interface{}) (replyBytes []byte, err error) {
	replyBytes, err = json.Marshal(map[string]interface{}{
		"reply": reply,
	})
	if err != nil {
		h.l.Error("could not encode reply", zap.Error(err))
	}
	return
}
