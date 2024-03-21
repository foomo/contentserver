package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/foomo/contentserver/pkg/handler"
	"github.com/foomo/contentserver/responses"
)

type connReturn struct {
	conn net.Conn
	err  error
}

type SocketTransport struct {
	connPool *connectionPool
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func NewSocketTransport(url string, connectionPoolSize int, waitTimeout time.Duration) *SocketTransport {
	return &SocketTransport{
		connPool: newConnectionPool(url, connectionPoolSize, waitTimeout),
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (t *SocketTransport) Call(ctx context.Context, route handler.Route, request interface{}, response interface{}) error {
	if t.connPool.chanDrainPool == nil {
		return errors.New("connection pool has been drained, client is dead")
	}
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("could not marshal request : %q", err)
	}
	netChan := make(chan net.Conn)
	t.connPool.chanConnGet <- netChan
	conn := <-netChan
	if conn == nil {
		return errors.New("could not get a connection")
	}
	returnConn := func(err error) {
		t.connPool.chanConnReturn <- connReturn{
			conn: conn,
			err:  err,
		}
	}
	// write header result will be like handler:2{}
	jsonBytes = append([]byte(fmt.Sprintf("%s:%d", route, len(jsonBytes))), jsonBytes...)

	// send request
	var (
		written = 0
		l       = len(jsonBytes)
	)
	for written < l {
		n, err := conn.Write(jsonBytes[written:])
		if err != nil {
			returnConn(err)
			return fmt.Errorf("failed to send request: %q", err)
		}
		written += n
	}

	// read response
	var (
		responseBytes  = []byte{}
		buf            = make([]byte, 4096)
		responseLength = 0
	)
	for {
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			returnConn(err)
			return fmt.Errorf("an error occured while reading the response: %q", err)
		}
		if n == 0 {
			break
		}
		responseBytes = append(responseBytes, buf[0:n]...)
		if responseLength == 0 {
			for index, byte := range responseBytes {
				if byte == 123 {
					// opening bracket
					responseLength, err = strconv.Atoi(string(responseBytes[0:index]))
					if err != nil {
						returnConn(err)
						return errors.New("could not read response length: " + err.Error())
					}
					responseBytes = responseBytes[index:]
					break
				}
			}
		}
		if responseLength > 0 && len(responseBytes) == responseLength {
			break
		}
	}

	// unmarshal response
	errResponse := json.Unmarshal(responseBytes, response)
	if errResponse != nil {
		// is it an error ?
		var (
			remoteErr        = responses.Error{}
			remoteErrJSONErr = json.Unmarshal(responseBytes, &remoteErr)
		)
		if remoteErrJSONErr == nil {
			returnConn(remoteErrJSONErr)
			return remoteErr
		}
		return fmt.Errorf("could not unmarshal response : %q %q", remoteErrJSONErr, string(responseBytes))
	}
	returnConn(nil)
	return nil
}

func (t *SocketTransport) Close() {
	if t.connPool.chanDrainPool != nil {
		t.connPool.chanDrainPool <- 1
	}
}
