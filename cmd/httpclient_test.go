package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// keel's transports stop waiting for response headers after 5s (internal) or
// 10s (external). A cold contentserverexport poll takes longer than that, so
// the repository timeout has to govern the header wait as well.
func TestNewRepositoryHTTPClientWaitsForSlowHeaders(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		internal    bool
		headerDelay time.Duration
	}{
		"internal": {internal: true, headerDelay: 6 * time.Second},
		"external": {internal: false, headerDelay: 11 * time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-time.After(tc.headerDelay):
					w.WriteHeader(http.StatusOK)
				case <-r.Context().Done():
				}
			}))
			t.Cleanup(srv.Close)

			client := newRepositoryHTTPClient(tc.internal, 30*time.Second)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)

			t.Cleanup(func() { _ = resp.Body.Close() })
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}
