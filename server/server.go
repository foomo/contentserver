package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/requests"
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

func (s *socketServer) handle(handler string, jsonBytes []byte) (replyBytes []byte, err error) {

	var reply interface{}
	var apiErr error
	var jsonErr error

	processIfJSONIsOk := func(err error, processingFunc func()) {
		if err != nil {
			jsonErr = err
			return
		}
		processingFunc()
	}

	switch handler {
	case "getURIs":
		getURIRequest := &requests.URIs{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &getURIRequest), func() {
			reply = s.repo.GetURIs(getURIRequest.Dimension, getURIRequest.Ids)
		})
	case "content":
		contentRequest := &requests.Content{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &contentRequest), func() {
			reply, apiErr = s.repo.GetContent(contentRequest)
		})
	case "getNodes":
		nodesRequest := &requests.Nodes{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &nodesRequest), func() {
			reply = s.repo.GetNodes(nodesRequest)
		})
	case "update":
		updateRequest := &requests.Update{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &updateRequest), func() {
			reply = s.repo.Update()
		})
	case "getRepo":
		repoRequest := &requests.Repo{}
		processIfJSONIsOk(json.Unmarshal(jsonBytes, &repoRequest), func() {
			reply = s.repo.GetRepo()
		})
	default:
		err = errors.New(log.Error("  can not handle this one " + handler))
		errorResponse := responses.NewError(1, "unknown handler")
		reply = errorResponse
	}
	return s.reply(reply, jsonErr, apiErr)
}

func (s *socketServer) reply(reply interface{}, jsonErr error, apiErr error) (replyBytes []byte, err error) {
	if jsonErr != nil {
		err = jsonErr
		log.Error("  could not read incoming json:", jsonErr)
		errorResponse := responses.NewError(2, "could not read incoming json "+jsonErr.Error())
		reply = errorResponse
	} else if apiErr != nil {
		log.Error("  an API error occured:", apiErr)
		err = apiErr
		reply = responses.NewError(3, "internal error "+apiErr.Error())
	}
	encodedBytes, jsonReplyErr := json.MarshalIndent(map[string]interface{}{
		"reply": reply,
	}, "", " ")
	if jsonReplyErr != nil {
		err = jsonReplyErr
		log.Error("  could not encode reply " + fmt.Sprint(jsonReplyErr))
	} else {
		replyBytes = encodedBytes
	}
	return replyBytes, err
}

func extractHandlerAndJSONLentgh(header string) (handler string, jsonLength int) {
	headerParts := strings.Split(header, ":")
	jsonLength, _ = strconv.Atoi(headerParts[1])
	return headerParts[0], jsonLength
}

func (s *socketServer) execute(conn net.Conn, handler string, jsonBytes []byte) {
	s.stats.countRequest()
	log.Record("socket.handleSocketRequest(%d): %s", s.stats.requests, handler)
	if log.SelectedLevel == log.LevelDebug {
		log.Debug("  incoming json buffer:", string(jsonBytes))
	}
	reply, handlingError := s.handle(handler, jsonBytes)
	if handlingError != nil {
		log.Error("socket.handleConnection handlingError :", handlingError)
		if reply == nil {
			log.Error("giving up with nil reply")
			conn.Close()
			return
		}
	}
	headerBytes := []byte(strconv.Itoa(len(reply)))
	reply = append(headerBytes, reply...)
	log.Debug("  replying: " + string(reply))
	_, writeError := conn.Write(reply)
	if writeError != nil {
		log.Error("socket.handleConnection: could not write my reply: " + fmt.Sprint(writeError))
		return
	}
	log.Debug("  replied. waiting for next request on open connection")
}

func (s *socketServer) handleConnection(conn net.Conn) {
	log.Debug("socket.handleConnection")
	var headerBuffer [1]byte
	header := ""
	for {
		// let us read with 1 byte steps on conn until we find "{"
		_, readErr := conn.Read(headerBuffer[0:])
		if readErr != nil {
			log.Debug("  looks like the client closed the connection - this is my readError: " + fmt.Sprint(readErr))
			return
		}
		// read next byte
		current := headerBuffer[0:]
		if string(current) == "{" {
			// reset header
			header = ""
			// json has started
			handler, jsonLength := extractHandlerAndJSONLentgh(header)
			log.Debug(fmt.Sprintf("  found json with %d bytes", jsonLength))
			if jsonLength > 0 {
				// let us try to read some json
				jsonBytes := make([]byte, jsonLength)
				// that is "{"
				jsonBytes[0] = 123
				_, jsonReadErr := conn.Read(jsonBytes[1:])
				if jsonReadErr != nil {
					log.Error("  could not read json - giving up with this client connection" + fmt.Sprint(jsonReadErr))
					return
				}
				if log.SelectedLevel == log.LevelDebug {
					log.Debug("  read json: " + string(jsonBytes))
				}
				s.execute(conn, handler, jsonBytes)
				return
			}
			log.Error("can not read empty json")
			conn.Close()
			return
		}
		// adding to header byte by byte
		header += string(headerBuffer[0:])
	}
}

// Run - let it run and enjoy on a socket near you
func Run(server string, address string, varDir string) error {
	log.Record("building repo with content from " + server)
	s := &socketServer{
		stats: newStats(),
		repo:  repo.NewRepo(server, varDir),
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		err = errors.New("RunSocketServer: could not start the on \"" + address + "\" - error: " + fmt.Sprint(err))
		// failed to create socket
		log.Error(err)
		return err
	}
	// there we go
	log.Record("RunSocketServer: started to listen on " + address)
	// update can run in bg
	go s.repo.Update()
	for {
		// this blocks until connection or error
		conn, err := ln.Accept()
		if err != nil {
			log.Error("RunSocketServer: could not accept connection" + fmt.Sprint(err))
			continue
		}
		// a goroutine handles conn so that the loop can accept other connections
		go s.handleConnection(conn)
	}
}
