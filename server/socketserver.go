package server

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/responses"
)

// simple internal request counter
type stats struct {
	requests  int64
	chanCount chan int
}

func newStats() *stats {
	s := &stats{
		requests:  0,
		chanCount: make(chan int),
	}
	go func() {
		for _ = range s.chanCount {
			s.requests++
			s.chanCount <- 1
		}
	}()
	return s
}

func (s *stats) countRequest() {
	s.chanCount <- 1
	<-s.chanCount
}

type socketServer struct {
	stats *stats
	repo  *repo.Repo
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
	s.stats.countRequest()
	log.Notice("socketServer.execute: ", s.stats.requests, ", ", handler)
	if log.SelectedLevel == log.LevelDebug {
		log.Debug("  incoming json buffer:", string(jsonBytes))
	}
	reply, handlingError := handleRequest(s.repo, handler, jsonBytes)
	if handlingError != nil {
		log.Error("socketServer.execute handlingError :", handlingError)
	}
	return reply
}

func (s *socketServer) writeResponse(conn net.Conn, reply []byte) {
	headerBytes := []byte(strconv.Itoa(len(reply)))
	reply = append(headerBytes, reply...)
	log.Debug("  replying: " + string(reply))
	n, writeError := conn.Write(reply)
	if writeError != nil {
		log.Error("socketServer.writeResponse: could not write my reply: " + fmt.Sprint(writeError))
		return
	}
	if n < len(reply) {
		log.Error(fmt.Sprintf("socketServer.writeResponse: write too short %q instead of %q", n, len(reply)))
		return
	}
	log.Debug("  replied. waiting for next request on open connection")

}

func (s *socketServer) handleConnection(conn net.Conn) {
	log.Debug("socketServer.handleConnection")
	var headerBuffer [1]byte
	header := ""
	i := 0
	for {
		i++
		// fmt.Println("---->", i)
		// let us read with 1 byte steps on conn until we find "{"
		_, readErr := conn.Read(headerBuffer[0:])
		if readErr != nil {
			log.Debug("  looks like the client closed the connection: ", readErr)
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
				log.Error("invalid request could not read header", headerErr)
				encodedErr, encodingErr := encodeReply(responses.NewError(4, "invalid header "+headerErr.Error()))
				if encodingErr == nil {
					s.writeResponse(conn, encodedErr)
				} else {
					log.Error("could not respond to invalid request", encodingErr)
				}
				return
			}
			log.Debug(fmt.Sprintf("  found json with %d bytes", jsonLength))
			if jsonLength > 0 {
				// let us try to read some json
				jsonBytes := make([]byte, jsonLength)
				// that is "{"
				jsonBytes[0] = 123
				jsonLengthCurrent := 1
				readRound := 0
				for jsonLengthCurrent < jsonLength {
					readRound++
					readLength, jsonReadErr := conn.Read(jsonBytes[jsonLengthCurrent:jsonLength])
					if jsonReadErr != nil {
						//@fixme we need to force a read timeout (SetReadDeadline?), if expected jsonLength is lower than really sent bytes (e.g. if client implements protocol wrong)
						//@todo should we check for io.EOF here
						log.Error("  could not read json - giving up with this client connection" + fmt.Sprint(jsonReadErr))
						return
					}
					jsonLengthCurrent += readLength
					log.Debug(fmt.Sprintf("  read so far %d of %d bytes in read cycle %d", jsonLengthCurrent, jsonLength, readRound))
				}

				if log.SelectedLevel == log.LevelDebug {
					log.Debug("  read json: " + string(jsonBytes))
				}
				s.writeResponse(conn, s.execute(handler, jsonBytes))
				// note: connection remains open
				continue
			}
			log.Error("can not read empty json")
			return
		}
		// adding to header byte by byte
		header += string(headerBuffer[0:])
	}
}
