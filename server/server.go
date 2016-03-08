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

// there should be sth. built in ?!
// anyway this ony concatenates two "ByteArrays"
func concat(a []byte, b []byte) []byte {
	newslice := make([]byte, len(a)+len(b))
	copy(newslice, a)
	copy(newslice[len(a):], b)
	return newslice
}

func (s *socketServer) handleSocketRequest(handler string, jsonBuffer []byte) (replyBytes []byte, err error) {
	s.stats.countRequest()
	var reply interface{}
	var apiErr error
	var jsonErr error
	log.Record(fmt.Sprintf("socket.handleSocketRequest(%d): %s %s", s.stats.requests, handler, string(jsonBuffer)))

	ifJSONIsFine := func(err error, processingFunc func()) {
		if err != nil {
			jsonErr = err
			return
		}
		processingFunc()
	}

	switch handler {
	case "getURIs":
		getURIRequest := &requests.URIs{}
		ifJSONIsFine(json.Unmarshal(jsonBuffer, &getURIRequest), func() {
			log.Debug("  getURIRequest: " + fmt.Sprint(getURIRequest))
			uris := s.repo.GetURIs(getURIRequest.Dimension, getURIRequest.Ids)
			log.Debug("    resolved: " + fmt.Sprint(uris))
			reply = uris
		})
	case "content":
		contentRequest := &requests.Content{}
		ifJSONIsFine(json.Unmarshal(jsonBuffer, &contentRequest), func() {
			log.Debug("contentRequest:", contentRequest)
			content, contentAPIErr := s.repo.GetContent(contentRequest)
			apiErr = contentAPIErr
			reply = content
		})
	case "getNodes":
		nodesRequest := &requests.Nodes{}
		ifJSONIsFine(json.Unmarshal(jsonBuffer, &nodesRequest), func() {
			log.Debug("  nodesRequest: " + fmt.Sprint(nodesRequest))
			nodesMap := s.repo.GetNodes(nodesRequest)
			reply = nodesMap
		})
	case "update":
		updateRequest := &requests.Update{}
		ifJSONIsFine(json.Unmarshal(jsonBuffer, &updateRequest), func() {
			log.Debug("  updateRequest: " + fmt.Sprint(updateRequest))
			updateResponse := s.repo.Update()
			reply = updateResponse
		})
	case "getRepo":
		repoRequest := &requests.Repo{}
		ifJSONIsFine(json.Unmarshal(jsonBuffer, &repoRequest), func() {
			log.Debug("  getRepoRequest: " + fmt.Sprint(repoRequest))
			repoResponse := s.repo.GetRepo()
			reply = repoResponse
		})
	default:
		err = errors.New(log.Error("  can not handle this one " + handler))
		errorResponse := responses.NewError(1, "unknown handler")
		reply = errorResponse
	}
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
	encodedBytes, jsonReplyErr := json.MarshalIndent(map[string]interface{}{"reply": reply}, "", " ")
	if jsonReplyErr != nil {
		err = jsonReplyErr
		log.Error("  could not encode reply " + fmt.Sprint(jsonReplyErr))
	} else {
		replyBytes = encodedBytes
	}
	return replyBytes, err
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
			// json has started
			headerParts := strings.Split(header, ":")
			header = ""
			requestHandler := headerParts[0]
			jsonLength, _ := strconv.Atoi(headerParts[1])
			log.Debug(fmt.Sprintf("  found json with %d bytes", jsonLength))
			if jsonLength > 0 {
				// let us try to read some json
				jsonBuffer := make([]byte, jsonLength)
				// that is "{"
				jsonBuffer[0] = 123
				_, jsonReadErr := conn.Read(jsonBuffer[1:])
				if jsonReadErr != nil {
					log.Error("  could not read json - giving up with this client connection" + fmt.Sprint(jsonReadErr))
					return
				}
				if log.SelectedLevel == log.LevelDebug {
					log.Debug("  read json: " + string(jsonBuffer))
				}

				// execution time
				reply, handlingError := s.handleSocketRequest(requestHandler, jsonBuffer)
				if handlingError != nil {
					log.Error("socket.handleConnection handlingError :", handlingError)
					if reply == nil {
						log.Error("giving up with nil reply")
						conn.Close()
						return
					}
				}
				headerBytes := []byte(strconv.Itoa(len(reply)))
				reply = concat(headerBytes, reply)
				log.Debug("  replying: " + string(reply))
				_, writeError := conn.Write(reply)
				if writeError != nil {
					log.Error("socket.handleConnection: could not write my reply: " + fmt.Sprint(writeError))
					return
				}
				log.Debug("  replied. waiting for next request on open connection")
			} else {
				log.Error("can not read empty json")
				conn.Close()
				return
			}
		} else {
			// adding to header byte by byte
			header += string(headerBuffer[0:])
		}
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
		log.Error(err.Error)
		return err
	}
	// there we go
	log.Record("RunSocketServer: started to listen on " + address)
	s.repo.Update()
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
