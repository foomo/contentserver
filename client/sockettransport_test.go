package client_test

import (
	"testing"
	"time"

	"github.com/foomo/contentserver/client"
	"github.com/foomo/contentserver/pkg/handler"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/net/nettest"
)

func BenchmarkSocketClientAndServerGetContent(b *testing.B) {
	l := zaptest.NewLogger(b)
	server := initSocketRepoServer(b, l)
	socketClient := newSocketClient(b, server)
	benchmarkServerAndClientGetContent(b, 30, 100, socketClient)
}

func newSocketClient(tb testing.TB, address string) *client.Client {
	tb.Helper()
	return client.New(client.NewSocketTransport(address, 25, 100*time.Millisecond))
}

func initSocketRepoServer(tb testing.TB, l *zap.Logger) string {
	tb.Helper()
	r := initRepo(tb, l)
	h := handler.NewSocket(l, r)

	// listen on socket
	ln, err := nettest.NewLocalListener("tcp")

	require.NoError(tb, err)

	go func() {
		for {
			// this blocks until connection or error
			conn, err := ln.Accept()
			if err != nil {
				tb.Error("runSocketServer: could not accept connection", err.Error())
				continue
			}

			// a goroutine handles conn so that the loop can accept other connections
			go func() {
				l.Debug("accepted connection", zap.String("source", conn.RemoteAddr().String()))
				h.Serve(conn)
				require.NoError(tb, conn.Close())
			}()
		}
	}()

	return ln.Addr().String()
}
