package client

import "github.com/foomo/contentserver/server"

type transport interface {
	call(handler server.Handler, request interface{}, response interface{}) error
	shutdown()
}
