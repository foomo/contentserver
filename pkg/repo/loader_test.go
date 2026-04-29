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

const (
	testPollPath = "/poll"
	testRepoPath = "/repo"
	testRepoBody = `{"dimension_foo":{"id":"id-root","uri":"/","name":"root","nodes":{},"index":[]}}`
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

	// The poll server returns a URL that points at itself + testRepoPath.
	// The testRepoPath endpoint serves the actual JSON repo content.
	var (
		pollCallCount int
		repoCallCount int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPollPath:
			pollCallCount++
			// No ETag header — legacy server behaviour.
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("http://" + r.Host + testRepoPath)) //nolint:gosec // r.Host is the test server's local address, not user input
		case testRepoPath:
			repoCallCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testRepoBody))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+testPollPath)

	// Start the background channel-routing goroutines that update() requires.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)          //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	// First call — no ETag on wire, repo must be fetched and loaded.
	_, err := r.update(ctx)
	require.NoError(t, err)
	assert.Empty(t, r.lastETag, "lastETag must stay empty when server sends no ETag")
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
		pollCallCount       int
		receivedIfNoneMatch string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPollPath:
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
			_, _ = w.Write([]byte("http://" + r.Host + testRepoPath)) //nolint:gosec // r.Host is the test server's local address, not user input

		case testRepoPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testRepoBody))

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+testPollPath)

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
		case testPollPath:
			callCount++
			switch callCount {
			case 1:
				// First call: return ETag v1 + body URL.
				w.Header().Set("ETag", etagV1)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("http://" + r.Host + testRepoPath + "?v=1")) //nolint:gosec // r.Host is the test server's local address, not user input
			case 2:
				// Second call: repo has changed — return 200 with new ETag v2 +
				// a different body URL to prevent URL-in-body skip.
				w.Header().Set("ETag", etagV2)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("http://" + r.Host + testRepoPath + "?v=2")) //nolint:gosec // r.Host is the test server's local address, not user input
			}
		case testRepoPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testRepoBody))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+testPollPath)

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

// TestUpdate_NonOKNon304_ReturnsErrorAndPreservesETag verifies the failure
// path: when the poll endpoint returns something other than 200 or 304 (e.g.
// 500), update() must return an error and must NOT clobber a previously
// captured ETag — so the next attempt can still send a valid If-None-Match.
func TestUpdate_NonOKNon304_ReturnsErrorAndPreservesETag(t *testing.T) {
	t.Parallel()

	const etagV1 = `"v1"`
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPollPath:
			callCount++
			switch callCount {
			case 1:
				// First call: 200 + ETag — stores etagV1.
				w.Header().Set("ETag", etagV1)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("http://" + r.Host + testRepoPath)) //nolint:gosec // r.Host is the test server's local address, not user input
			default:
				// Subsequent calls: 500 — must NOT overwrite lastETag.
				http.Error(w, "boom", http.StatusInternalServerError)
			}
		case testRepoPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testRepoBody))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+testPollPath)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)          //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	// First call — primes lastETag to etagV1.
	_, err := r.update(ctx)
	require.NoError(t, err)
	require.Equal(t, etagV1, r.lastETag)

	// Second call — server replies 500. update() must return an error and
	// MUST preserve the previously captured ETag so the next retry can still
	// send a valid If-None-Match.
	_, err = r.update(ctx)
	require.Error(t, err, "non-200/304 must surface as an error")
	assert.Equal(t, etagV1, r.lastETag, "lastETag must be preserved across error responses")
}

// TestUpdate_ETagThenAbsent_PreservesLast verifies that when the server emits
// an ETag once and subsequently returns 200 with no ETag header, the loader
// keeps the previously captured value (so a future ETag-aware response from
// the same server can short-circuit again). This pins the "do not clear on
// missing header" design choice in update().
func TestUpdate_ETagThenAbsent_PreservesLast(t *testing.T) {
	t.Parallel()

	const etagV1 = `"v1"`
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPollPath:
			callCount++
			switch callCount {
			case 1:
				// First call: 200 + ETag + body URL.
				w.Header().Set("ETag", etagV1)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("http://" + r.Host + testRepoPath + "?v=1")) //nolint:gosec // r.Host is the test server's local address, not user input
			default:
				// Subsequent calls: 200 with NO ETag header, different body URL
				// to bypass the URL-in-body skip and reach the ETag-update line.
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("http://" + r.Host + testRepoPath + "?v=2")) //nolint:gosec // r.Host is the test server's local address, not user input
			}
		case testRepoPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testRepoBody))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+testPollPath)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)          //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	// First call — stores etagV1.
	_, err := r.update(ctx)
	require.NoError(t, err)
	require.Equal(t, etagV1, r.lastETag)

	// Second call — server omits ETag. update() must NOT clear lastETag; we
	// still want to send If-None-Match: etagV1 on the next request in case
	// the server re-enables ETag emission.
	_, err = r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, etagV1, r.lastETag, "lastETag must be preserved when a 200 response omits ETag")
}

// TestUpdate_NoIfNoneMatchOnFirstCall verifies that the very first poll
// request does not include an If-None-Match header (lastETag is empty at
// startup).
func TestUpdate_NoIfNoneMatchOnFirstCall(t *testing.T) {
	t.Parallel()

	var firstRequestHadIfNoneMatch bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPollPath:
			if r.Header.Get("If-None-Match") != "" {
				firstRequestHadIfNoneMatch = true
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("http://" + r.Host + testRepoPath)) //nolint:gosec // r.Host is the test server's local address, not user input
		case testRepoPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testRepoBody))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+testPollPath)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go r.UpdateRoutine(ctx)          //nolint:errcheck
	go r.DimensionUpdateRoutine(ctx) //nolint:errcheck

	_, err := r.update(ctx)
	require.NoError(t, err)
	assert.False(t, firstRequestHadIfNoneMatch, "first call must NOT include an If-None-Match header")
}
