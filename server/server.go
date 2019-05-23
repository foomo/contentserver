package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo"
	jsoniter "github.com/json-iterator/go"

	// profiling
	_ "net/http/pprof"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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

// Run - let it run and enjoy on a socket near you
func Run(server string, address string, varDir string) error {
	return RunServerSocketAndWebServer(server, address, "", "", varDir)
}

func RunServerSocketAndWebServer(
	server string,
	address string,
	webserverAddress string,
	webserverPath string,
	varDir string,
) error {
	if address == "" && webserverAddress == "" {
		return errors.New("one of the addresses needs to be set")
	}
	log.Record("building repo with content from " + server)
	r := repo.NewRepo(server, varDir)

	// start initial update and handle error
	go func() {
		resp := r.Update()
		if !resp.Success {
			log.Error("failed to update: ", resp)
			os.Exit(1)
		}
	}()

	// update can run in bg
	chanErr := make(chan error)

	if address != "" {
		log.Notice("starting socketserver on: ", address)
		go runSocketServer(r, address, chanErr)
	}
	if webserverAddress != "" {
		log.Notice("starting webserver on: ", webserverAddress)
		go runWebserver(r, webserverAddress, webserverPath, chanErr)
	}
	return <-chanErr
}

func runWebserver(
	r *repo.Repo,
	address string,
	path string,
	chanErr chan error,
) {
	chanErr <- http.ListenAndServe(address, NewWebServer(path, r))
}

func runSocketServer(
	repo *repo.Repo,
	address string,
	chanErr chan error,
) {
	// create socket server
	s := newSocketServer(repo)

	// listen on socket
	ln, errListen := net.Listen("tcp", address)
	if errListen != nil {
		errListenSocket := errors.New("RunSocketServer: could not start the on \"" + address + "\" - error: " + fmt.Sprint(errListen))
		log.Error(errListenSocket)
		chanErr <- errListenSocket
		return
	}

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
