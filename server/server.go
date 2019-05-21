package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/repo"

	// profiling
	_ "net/http/pprof"
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

// Run - let it run and enjoy on a socket near you
func Run(server string, address string, varDir string) error {
	return RunServerSocketAndWebServer(server, address, "", "", varDir)
}

func RunServerSocketAndWebServer(
	server string,
	address string,
	webserverAdresss string,
	webserverPath string,
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
		go runWebserver(r, webserverAdresss, webserverPath, chanErr)
	}
	return <-chanErr
}

func runWebserver(
	r *repo.Repo,
	address string,
	path string,
	chanErr chan error,
) {
	s, errNew := NewWebServer(path, r)
	if errNew != nil {
		chanErr <- errNew
		return
	}
	chanErr <- http.ListenAndServe(address, s)
}

func runSocketServer(
	repo *repo.Repo,
	address string,
	chanErr chan error,
) {
	// create socket server
	s, errSocketServer := newSocketServer(repo)
	if errSocketServer != nil {
		log.Error(errSocketServer)
		chanErr <- errSocketServer
		return
	}

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
