package cmd

import (
	"net/http"
	"time"

	keelhttp "github.com/foomo/keel/net/http"
)

// newRepositoryHTTPClient builds the client used to fetch and poll the
// repository. keel's transports give up on response headers after a few
// seconds, which is shorter than a cold repository export takes to build, so
// the repository timeout has to bound the header wait as well as the request.
func newRepositoryHTTPClient(internal bool, timeout time.Duration) *http.Client {
	newClient := keelhttp.NewExternalHTTPClient
	if internal {
		newClient = keelhttp.NewInternalHTTPClient
	}

	return newClient(
		keelhttp.HTTPClientWithTimeout(timeout),
		keelhttp.HTTPClientWithResponseHeaderTimeout(timeout),
		keelhttp.HTTPClientWithTelemetry(),
	)
}
