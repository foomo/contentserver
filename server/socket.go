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
	switch handler {
	case "getURIs":
		getURIRequest := &requests.URIs{}
		jsonErr = json.Unmarshal(jsonBuffer, &getURIRequest)
		log.Debug("  getURIRequest: " + fmt.Sprint(getURIRequest))
		uris := s.repo.GetURIs(getURIRequest.Dimension, getURIRequest.Ids)
		log.Debug("    resolved: " + fmt.Sprint(uris))
		reply = uris
		break
	case "content":
		contentRequest := &requests.Content{}
		jsonErr = json.Unmarshal(jsonBuffer, &contentRequest)
		log.Debug("contentRequest:", contentRequest)
		content, apiErr := s.repo.GetContent(contentRequest)
		log.Debug(apiErr)
		reply = content
		break
	case "getNodes":
		nodesRequest := &requests.Nodes{}
		jsonErr = json.Unmarshal(jsonBuffer, &nodesRequest)
		log.Debug("  nodesRequest: " + fmt.Sprint(nodesRequest))
		nodesMap := s.repo.GetNodes(nodesRequest)
		reply = nodesMap
		break
	case "update":
		updateRequest := &requests.Update{}
		jsonErr = json.Unmarshal(jsonBuffer, &updateRequest)
		log.Debug("  updateRequest: " + fmt.Sprint(updateRequest))
		updateResponse := s.repo.Update()
		reply = updateResponse
		break
	case "getRepo":
		repoRequest := &requests.Repo{}
		jsonErr = json.Unmarshal(jsonBuffer, &repoRequest)
		log.Debug("  getRepoRequest: " + fmt.Sprint(repoRequest))
		repoResponse := s.repo.GetRepo()
		reply = repoResponse
		break
	default:
		err = errors.New(log.Error("  can not handle this one " + handler))
		errorResponse := responses.NewError(1, "unknown handler")
		reply = errorResponse
	}
	if jsonErr != nil {
		log.Error("  could not read incoming json: " + fmt.Sprint(jsonErr))
		errorResponse := responses.NewError(2, "could not read incoming json "+jsonErr.Error())
		reply = errorResponse
	} else if apiErr != nil {
		reply = responses.NewError(3, "internal error "+apiErr.Error())
	}
	encodedBytes, jsonErr := json.MarshalIndent(map[string]interface{}{"reply": reply}, "", " ")
	if jsonErr != nil {
		err = jsonErr
		log.Error("  could not encode reply " + fmt.Sprint(jsonErr))
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
		_, readErr := conn.Read(headerBuffer[0:])
		if readErr != nil {
			log.Debug("  looks like the client closed the connection - this is my readError: " + fmt.Sprint(readErr))
			return
		}
		// read next byte
		current := string(headerBuffer[0:])
		if current == "{" {
			// json has started
			headerParts := strings.Split(header, ":")
			header = ""
			requestHandler := headerParts[0]
			jsonLength, _ := strconv.Atoi(headerParts[1])
			log.Debug(fmt.Sprintf("  found json with %d bytes", jsonLength))
			if jsonLength > 0 {
				jsonBuffer := make([]byte, jsonLength)
				jsonBuffer[0] = 123
				_, jsonReadErr := conn.Read(jsonBuffer[1:])
				if jsonReadErr != nil {
					log.Error("  could not read json - giving up with this client connection" + fmt.Sprint(jsonReadErr))
					return
				}
				log.Debug("  read json: " + string(jsonBuffer))
				reply, handlingError := s.handleSocketRequest(requestHandler, jsonBuffer)
				if handlingError != nil {
					log.Error("socket.handleConnection: handlingError " + fmt.Sprint(handlingError))
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

// RunSocketServer - let it run and enjoy on a socket near you
func RunSocketServer(server string, address string, varDir string) {
	log.Record("building repo with content from " + server)
	s := &socketServer{
		stats: newStats(),
		repo:  repo.NewRepo(server, varDir),
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		// failed to create socket
		log.Error("RunSocketServer: could not start the on \"" + address + "\" - error: " + fmt.Sprint(err))
	} else {
		// there we go
		log.Record("RunSocketServer: started to listen on " + address)
		s.repo.Update()
		for {
			conn, err := ln.Accept() // this blocks until connection or error
			if err != nil {
				log.Error("RunSocketServer: could not accept connection" + fmt.Sprint(err))
				continue
			}
			// a goroutine handles conn so that the loop can accept other connections
			go s.handleConnection(conn)
		}
	}
}
