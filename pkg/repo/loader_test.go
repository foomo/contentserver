package repo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
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

func TestPollRoutineLogsSuccessfulVersion(t *testing.T) {
	// t.Parallel()
	core, logs := observer.New(zap.InfoLevel)
	r := New(zap.New(core), "http://example.test/repo", nil, WithPoll(true), WithPollInterval(time.Millisecond))
	r.version = `"v1"`

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	handledUpdate := make(chan struct{}, 1)
	stopResponder := make(chan struct{})

	responderDone := make(chan struct{})
	go func() {
		defer close(responderDone)

		for {
			select {
			case resChan := <-r.updateInProgressChannel:
				resChan <- updateResponse{}

				select {
				case handledUpdate <- struct{}{}:
				default:
				}
			case <-stopResponder:
				return
			}
		}
	}()

	pollDone := make(chan error, 1)
	go func() {
		pollDone <- r.PollRoutine(ctx)
	}()

	select {
	case <-handledUpdate:
	case <-time.After(time.Second):
		t.Fatal("poll routine did not request an update")
	}

	cancel()

	select {
	case err := <-pollDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("poll routine did not stop after context cancellation")
	}

	close(stopResponder)
	<-responderDone

	entries := logs.FilterMessage("update success").All()
	require.NotEmpty(t, entries)
	assert.Equal(t, `"v1"`, entries[0].ContextMap()["revision"])
}

// TestUpdate_NoETag_BackwardCompat verifies that when the poll server never
// sends an ETag header the loader behaves exactly as it did before this
// change: it fetches the body URL, loads the repo, and on a second call with
// the same poll response it skips via the URL-in-body comparison.
func TestUpdate_NoETag_BackwardCompat(t *testing.T) {
	// t.Parallel()

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

	// First call — no ETag on wire, repo must be fetched and loaded. With the
	// unified version field, version falls back to the URL returned in the body.
	_, err := r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+testRepoPath, r.version, "version must equal the body URL when server sends no ETag")
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
	// t.Parallel()
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
	assert.Equal(t, etagV1, r.version, "ETag must be stored after first successful poll")
	assert.Equal(t, 1, pollCallCount)

	// Second call: loader must send If-None-Match; server replies 304.
	_, err = r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, etagV1, receivedIfNoneMatch, "If-None-Match must equal the stored ETag")
	assert.Equal(t, 2, pollCallCount)
	assert.Equal(t, etagV1, r.version, "version must remain unchanged after 304")
}

// TestUpdate_ETagChange verifies that when the server returns a new ETag on a
// 200 response the loader updates version and fetches the new repo content.
func TestUpdate_ETagChange(t *testing.T) {
	// t.Parallel()
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
	assert.Equal(t, etagV1, r.version)

	// Second call — server returns new ETag v2; loader must update version.
	_, err = r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, etagV2, r.version, "version must be updated to the new value after 200 response")
}

// TestUpdate_NonOKNon304_ReturnsErrorAndPreservesETag verifies the failure
// path: when the poll endpoint returns something other than 200 or 304 (e.g.
// 500), update() must return an error and must NOT clobber a previously
// captured ETag — so the next attempt can still send a valid If-None-Match.
func TestUpdate_NonOKNon304_ReturnsErrorAndPreservesETag(t *testing.T) {
	// t.Parallel()
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
				// Subsequent calls: 500 — must NOT overwrite version.
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

	// First call — primes version to etagV1.
	_, err := r.update(ctx)
	require.NoError(t, err)
	require.Equal(t, etagV1, r.version)

	// Second call — server replies 500. update() must return an error and
	// MUST preserve the previously captured ETag so the next retry can still
	// send a valid If-None-Match.
	_, err = r.update(ctx)
	require.Error(t, err, "non-200/304 must surface as an error")
	assert.Equal(t, etagV1, r.version, "version must be preserved across error responses")
}

// TestUpdate_ETagThenAbsent_FallsBackToURL verifies that when the server emits
// an ETag once and subsequently returns 200 with no ETag header, the unified
// version field transitions to the body URL. This pins the "ETag if present,
// else URL" rule for the version field.
func TestUpdate_ETagThenAbsent_FallsBackToURL(t *testing.T) {
	// t.Parallel()
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
	require.Equal(t, etagV1, r.version)

	// Second call — server omits ETag. version must fall back to the body URL.
	_, err = r.update(ctx)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+testRepoPath+"?v=2", r.version, "version must fall back to body URL when ETag is absent")
}

// TestUpdate_VersionNotCommittedOnLoadFailure pins the contract that a
// poll response's ETag MUST NOT be committed to version until the catalogue
// body has been fully fetched and loaded. Without this guarantee, a transient
// failure downloading the body URL would leave version pointing at content
// the loader never actually loaded — the next poll's If-None-Match would
// elicit a 304 and the loader would silently log "up to date" without ever
// recovering. This is a regression test for that ordering bug.
func TestUpdate_VersionNotCommittedOnLoadFailure(t *testing.T) {
	// t.Parallel()
	const etagV1 = `"v1"`

	var pollCallCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPollPath:
			pollCallCount++
			// Poll always returns 200 + ETag + a URL pointing at the repo
			// endpoint, which deliberately fails on every call below.
			w.Header().Set("ETag", etagV1)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("http://" + r.Host + testRepoPath)) //nolint:gosec // r.Host is the test server's local address, not user input
		case testRepoPath:
			// Simulate a transient catalogue download failure (CDN blip,
			// transient 5xx) — exactly the scenario where the loader must
			// not poison version.
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := newMinimalRepo(t, srv.URL+testPollPath)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// First call — poll returns 200+ETag, but the body fetch fails. version
	// MUST stay empty so the next poll re-attempts (no silent 304 short-circuit).
	_, err := r.update(ctx)
	require.Error(t, err, "catalogue download failure must surface as an error")
	assert.Empty(t, r.version, "version must NOT be committed when the catalogue load failed")
	require.Equal(t, 1, pollCallCount)

	// Second call — must again attempt the poll and the body fetch (no 304
	// short-circuit), proving the bug pattern of perma-staleness is gone.
	_, err = r.update(ctx)
	require.Error(t, err)
	assert.Empty(t, r.version, "version must still be empty after a second failed load")
	assert.Equal(t, 2, pollCallCount, "poll endpoint must be hit again — no 304 short-circuit when version was never committed")
}

// TestUpdate_NoIfNoneMatchOnFirstCall verifies that the very first poll
// request does not include an If-None-Match header (version is empty at
// startup).
func TestUpdate_NoIfNoneMatchOnFirstCall(t *testing.T) {
	// t.Parallel()
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
