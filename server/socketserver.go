package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/contentserver/status"
)

const sourceSocketServer = "socketserver"

type socketServer struct {
	repo repoer
}

// newSocketServer returns a shiny new socket server
func newSocketServer(repo repoer) *socketServer {
	return &socketServer{
		repo: repo,
	}
}

func extractHandlerAndJSONLength(header string) (handler Handler, jsonLength int, err error) {
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

	var buf bytes.Buffer
	buf.Grow(len(jsonBytes))
	if err := handleRequest(s.repo, handler, bytes.NewReader(jsonBytes), &buf, sourceSocketServer); err != nil {
		Log.Error("socketServer.execute failed", zap.Error(err))
	}
	return buf.Bytes()
}

func (s *socketServer) writeResponse(w io.Writer, reply []byte) {
	headerBytes := make([]byte, 0, len(reply)+12) // 12 is the approx length of header to store the size of json, next line.
	headerBytes = strconv.AppendInt(headerBytes, int64(len(reply)), 10)
	reply = append(headerBytes, reply...)
	if Log.Core().Enabled(zap.DebugLevel) {
		// only log when debug level is really enabled as this costs lots of performance.
		Log.Debug("replying", zap.String("reply", string(reply)))
	}
	n, writeError := w.Write(reply)
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
		header       bytes.Buffer
		i            = 0
	)
	for {
		i++
		// fmt.Println("---->", i)
		// let us read with 1 byte steps on conn until we find "{"
		_, readErr := conn.Read(headerBuffer[:])
		if readErr != nil {
			Log.Debug("looks like the client closed the connection", zap.Error(readErr))
			status.M.NumSocketsGauge.WithLabelValues(conn.RemoteAddr().String()).Dec()
			return
		}
		// read next byte
		if headerBuffer[0] == '{' {
			// json has started
			handler, jsonLength, headerErr := extractHandlerAndJSONLength(header.String())
			// reset header
			header.Reset()
			if headerErr != nil {
				Log.Error("invalid request could not read header", zap.Error(headerErr))
				var buf bytes.Buffer
				if err := encodeReply(&buf, responses.NewErrorf(4, "invalid header %s", headerErr)); err == nil {
					s.writeResponse(conn, buf.Bytes())
				} else {
					Log.Error("could not respond to invalid request", zap.Error(err))
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

				jsonBytes[0] = '{'

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
		header.WriteByte(headerBuffer[0])
	}
}
