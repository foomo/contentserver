package server

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/status"
)

const sourceSocketServer = "socketserver"

type socketServer struct {
	repo    *repo.Repo
	metrics *status.Metrics
}

// newSocketServer returns a shiny new socket server
func newSocketServer(repo *repo.Repo) *socketServer {
	return &socketServer{
		repo: repo,
	}
}

func extractHandlerAndJSONLentgh(header string) (handler Handler, jsonLength int, err error) {
	headerParts := strings.Split(header, ":")
	if len(headerParts) != 2 {
		return "", 0, errors.New("invalid header")
	}
	jsonLength, err = strconv.Atoi(headerParts[1])
	if err != nil {
		err = fmt.Errorf("could not parse length in header: %q", header)
	}
	return Handler(headerParts[0]), jsonLength, err
}

func (s *socketServer) execute(handler Handler, jsonBytes []byte) (reply []byte) {
	Log.Debug("incoming json buffer", zap.Int("length", len(jsonBytes)))

	if handler == HandlerGetRepo {

		var (
			b     bytes.Buffer
			start = time.Now()
		)
		s.repo.WriteRepoBytes(&b)
		addMetrics(handler, start, nil, nil, sourceSocketServer)
		return b.Bytes()
	}

	reply, handlingError := handleRequest(s.repo, handler, jsonBytes, sourceSocketServer)
	if handlingError != nil {
		Log.Error("socketServer.execute failed", zap.Error(handlingError))
	}
	return reply
}

func (s *socketServer) writeResponse(conn net.Conn, reply []byte) {
	headerBytes := []byte(strconv.Itoa(len(reply)))
	reply = append(headerBytes, reply...)
	Log.Debug("replying", zap.String("reply", string(reply)))
	n, writeError := conn.Write(reply)
	if writeError != nil {
		Log.Error("socketServer.writeResponse: could not write reply", zap.Error(writeError))
		return
	}
	if n < len(reply) {
		Log.Error("socketServer.writeResponse: write too short",
			zap.Int("got", n),
			zap.Int("expected", len(reply)),
		)
		return
	}
	Log.Debug("replied. waiting for next request on open connection")
}

func (s *socketServer) handleConnection(conn net.Conn) {
	Log.Debug("socketServer.handleConnection")
	status.M.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Inc()

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
			Log.Debug("looks like the client closed the connection", zap.Error(readErr))
			status.M.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()
			return
		}
		// read next byte
		current := headerBuffer[0:]
		if string(current) == "{" {
			// json has started
			handler, jsonLength, headerErr := extractHandlerAndJSONLentgh(header)
			// reset header
			header = ""
			if headerErr != nil {
				Log.Error("invalid request could not read header", zap.Error(headerErr))
				encodedErr, encodingErr := encodeReply(responses.NewError(4, "invalid header "+headerErr.Error()))
				if encodingErr == nil {
					s.writeResponse(conn, encodedErr)
				} else {
					Log.Error("could not respond to invalid request", zap.Error(encodingErr))
				}
				return
			}
			Log.Debug("found json", zap.Int("length", jsonLength))
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
						//@fixme we need to force a read timeout (SetReadDeadline?), if expected jsonLength is lower than really sent bytes (e.g. if client implements protocol wrong)
						//@todo should we check for io.EOF here
						Log.Error("could not read json - giving up with this client connection", zap.Error(jsonReadErr))
						status.M.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()
						return
					}
					jsonLengthCurrent += readLength
					Log.Debug("read cycle status",
						zap.Int("jsonLengthCurrent", jsonLengthCurrent),
						zap.Int("jsonLength", jsonLength),
						zap.Int("readRound", readRound),
					)
				}

				Log.Debug("read json", zap.Int("length", len(jsonBytes)))

				s.writeResponse(conn, s.execute(handler, jsonBytes))
				// note: connection remains open
				continue
			}
			Log.Error("can not read empty json")
			status.M.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()
			return
		}
		// adding to header byte by byte
		header += string(headerBuffer[0:])
	}
}
