package client

import (
	"context"

	"github.com/foomo/contentserver/pkg/handler"
)

type Transport interface {
	Call(ctx context.Context, route handler.Route, request interface{}, response interface{}) error
	Close()
}
