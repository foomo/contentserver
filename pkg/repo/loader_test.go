package repo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// newMinimalRepo creates a Repo wired up for poll mode against the given URL,
// with a temporary history dir. It does NOT start the background routines so
// tests can call update() directly and observe the state.
func newMinimalRepo(t *testing.T, url string) *Repo {
	t.Helper()
	l := zaptest.NewLogger(t)
	h, err := NewHistory(l, HistoryWithHistoryLimit(2), HistoryWithHistoryDir(t.TempDir()))
	require.NoError(t, err)
	return New(l, url, h, WithPoll(true))
}

// TestUpdate_NoETag_BackwardCompat verifies that when the poll server never
// sends an ETag header the loader behaves exactly as it did before this
// change: it fetches the body URL, loads the repo, and on a second call with
// the same poll response it skips via the URL-in-body comparison.
func TestUpdate_NoETag_BackwardCompat(t *testing.T) {
	t.Parallel()

	// The poll server returns a URL that points at itself + "/repo".
	// The "/repo" endpoint serves the actual JSON repo content.
	var (
		pollCallCount int
		repoCallCount int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/poll":
			pollCallCount++
			// No ETag header — legacy server behaviour.
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("http://" + r.Host + "/repo"))
		case "/repo":
			repoCallCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"dimension_foo":{"id":"id-root","uri":"/","name":"root","nodes":{},"index":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+"/poll")

	// Start the background channel-routing goroutines that update() requires.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)   //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	// First call — no ETag on wire, repo must be fetched and loaded.
	_, err := r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, "", r.lastETag, "lastETag must stay empty when server sends no ETag")
	assert.Equal(t, 1, pollCallCount)
	assert.Equal(t, 1, repoCallCount)

	// Second call — same URL returned by poll server, URL-in-body comparison
	// must fire and skip the repo fetch.
	_, err = r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, pollCallCount, "poll endpoint must be hit again")
	assert.Equal(t, 1, repoCallCount, "repo endpoint must NOT be hit again (URL-in-body skip)")
}

// TestUpdate_ETagSetThenNotModified verifies the happy-path ETag flow:
// first call stores the ETag; second call sends If-None-Match and the server
// replies 304, causing the loader to skip the body read.
func TestUpdate_ETagSetThenNotModified(t *testing.T) {
	t.Parallel()

	const etagV1 = `"v1"`
	var (
		pollCallCount         int
		receivedIfNoneMatch   string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/poll":
			pollCallCount++
			inm := r.Header.Get("If-None-Match")
			if inm != "" {
				receivedIfNoneMatch = inm
			}

			if inm == etagV1 {
				// Confirm the ETag — content unchanged.
				w.Header().Set("ETag", etagV1)
				w.WriteHeader(http.StatusNotModified)
				return
			}

			// First call: return 200 + ETag + URL body.
			w.Header().Set("ETag", etagV1)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("http://" + r.Host + "/repo"))

		case "/repo":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"dimension_foo":{"id":"id-root","uri":"/","name":"root","nodes":{},"index":[]}}`))

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+"/poll")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)          //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	// First call: 200 + ETag; loader stores the ETag.
	_, err := r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, etagV1, r.lastETag, "ETag must be stored after first successful poll")
	assert.Equal(t, 1, pollCallCount)

	// Second call: loader must send If-None-Match; server replies 304.
	_, err = r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, etagV1, receivedIfNoneMatch, "If-None-Match must equal the stored ETag")
	assert.Equal(t, 2, pollCallCount)
	assert.Equal(t, etagV1, r.lastETag, "lastETag must remain unchanged after 304")
}

// TestUpdate_ETagChange verifies that when the server returns a new ETag on a
// 200 response the loader updates lastETag and fetches the new repo content.
func TestUpdate_ETagChange(t *testing.T) {
	t.Parallel()

	const (
		etagV1 = `"v1"`
		etagV2 = `"v2"`
	)
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/poll":
			callCount++
			switch callCount {
			case 1:
				// First call: return ETag v1 + body URL.
				w.Header().Set("ETag", etagV1)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("http://" + r.Host + "/repo?v=1"))
			case 2:
				// Second call: repo has changed — return 200 with new ETag v2 +
				// a different body URL to prevent URL-in-body skip.
				w.Header().Set("ETag", etagV2)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("http://" + r.Host + "/repo?v=2"))
			}
		case "/repo":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"dimension_foo":{"id":"id-root","uri":"/","name":"root","nodes":{},"index":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+"/poll")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)          //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	// First call — stores v1.
	_, err := r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, etagV1, r.lastETag)

	// Second call — server returns new ETag v2; loader must update lastETag.
	_, err = r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, etagV2, r.lastETag, "lastETag must be updated to the new value after 200 response")
}

// TestUpdate_NoIfNoneMatchOnFirstCall verifies that the very first poll
// request does not include an If-None-Match header (lastETag is empty at
// startup).
func TestUpdate_NoIfNoneMatchOnFirstCall(t *testing.T) {
	t.Parallel()

	var firstRequestHadIfNoneMatch bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/poll":
			if r.Header.Get("If-None-Match") != "" {
				firstRequestHadIfNoneMatch = true
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("http://" + r.Host + "/repo"))
		case "/repo":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"dimension_foo":{"id":"id-root","uri":"/","name":"root","nodes":{},"index":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+"/poll")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)          //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	_, err := r.update(ctx)
	require.NoError(t, err)
	assert.False(t, firstRequestHadIfNoneMatch, "first call must NOT include an If-None-Match header")
}
