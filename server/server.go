package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/responses"
)

// Handler type
type Handler string

const (
	// HandlerGetURIs get uris, many at once, to keep it fast
	HandlerGetURIs Handler = "getURIs"
	// HandlerGetContent get (site) content
	HandlerGetContent = "getContent"
	// HandlerGetNodes get nodes
	HandlerGetNodes = "getNodes"
	// HandlerUpdate update repo
	HandlerUpdate = "update"
	// HandlerGetRepo get the whole repo
	HandlerGetRepo = "getRepo"
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
		for {
			select {
			case <-s.chanCount:
				s.requests++
				s.chanCount <- 1
			}
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

// Run - let it run and enjoy on a socket near you

func Run(server string, address string, varDir string) error {
	return RunServerSocketAndWebServer(server, address, "", varDir)
}

func RunServerSocketAndWebServer(
	server string,
	address string,
	webserverAdresss string,
	varDir string,
) error {
	if address == "" && webserverAdresss == "" {
		return errors.New("one of the addresses needs to be set")
	}
	log.Record("building repo with content from " + server)
	r := repo.NewRepo(server, varDir)
	go r.Update()
	// update can run in bg
	chanErr := make(chan error)
	if address != "" {
		go runSocketServer(r, address, chanErr)
	}
	if webserverAdresss != "" {
		go runWebserver(r, webserverAdresss, chanErr)
	}
	return <-chanErr
}

func runWebserver(
	r *repo.Repo,
	address string,
	chanErr chan error,
) {
	s := &webServer{
		r: r,
	}
	chanErr <- http.ListenAndServe(address, s)
}

func runSocketServer(
	repo *repo.Repo,
	address string,
	chanErr chan error,
) {
	s := &socketServer{
		stats: newStats(),
		repo:  repo,
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		err = errors.New("RunSocketServer: could not start the on \"" + address + "\" - error: " + fmt.Sprint(err))
		// failed to create socket
		log.Error(err)
		chanErr <- err
		return
	}
	// there we go
	log.Record("RunSocketServer: started to listen on " + address)
	for {
		// this blocks until connection or error
		conn, err := ln.Accept()
		if err != nil {
			log.Error("RunSocketServer: could not accept connection" + fmt.Sprint(err))
			continue
		}
		log.Debug("new connection")
		// a goroutine handles conn so that the loop can accept other connections
		go func() {
			log.Debug("accepted connection")
			s.handleConnection(conn)
			conn.Close()
			// log.Debug("connection closed")
		}()
	}
}
