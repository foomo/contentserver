package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo"
	"github.com/foomo/contentserver/status"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var (
	json    = jsoniter.ConfigCompatibleWithStandardLibrary
	metrics = status.NewMetrics()
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
	webserverAddress string,
	webserverPath string,
	varDir string,
) error {
	if address == "" && webserverAddress == "" {
		return errors.New("one of the addresses needs to be set")
	}
	Log.Info("building repo with content", zap.String("server", server))

	r := repo.NewRepo(server, varDir)

	// start initial update and handle error
	go func() {
		resp := r.Update()
		if !resp.Success {
			Log.Error("failed to update",
				zap.String("error", resp.ErrorMessage),
				zap.Int("NumberOfNodes", resp.Stats.NumberOfNodes),
				zap.Int("NumberOfURIs", resp.Stats.NumberOfURIs),
				zap.Float64("OwnRuntime", resp.Stats.OwnRuntime),
				zap.Float64("RepoRuntime", resp.Stats.RepoRuntime),
			)
			os.Exit(1)
		}
	}()

	// update can run in bg
	chanErr := make(chan error)

	if address != "" {
		Log.Info("starting socketserver", zap.String("address", address))
		go runSocketServer(r, address, chanErr)
	}
	if webserverAddress != "" {
		Log.Info("starting webserver", zap.String("webserverAddress", webserverAddress))
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
		Log.Error("runSocketServer: could not start",
			zap.String("address", address),
			zap.Error(errListen),
		)
		chanErr <- errors.New("runSocketServer: could not start the on \"" + address + "\" - error: " + fmt.Sprint(errListen))
		return
	}

	Log.Info("runSocketServer: started listening", zap.String("address", address))
	for {
		// this blocks until connection or error
		conn, err := ln.Accept()
		if err != nil {
			Log.Error("runSocketServer: could not accept connection", zap.Error(err))
			continue
		}

		// a goroutine handles conn so that the loop can accept other connections
		go func() {
			Log.Debug("accepted connection", zap.String("source", conn.RemoteAddr().String()))
			s.handleConnection(conn)
			conn.Close()
			// log.Debug("connection closed")
		}()
	}
}
