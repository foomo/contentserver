package repo

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type updateMetricsTransport func(*http.Request) (*http.Response, error)

func (f updateMetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func resetUpdateTimestamps(t *testing.T) {
	t.Helper()
	metrics.LastSuccessfulUpdateTimestamp.Set(0)
	metrics.LastFailedUpdateTimestamp.Set(0)
	t.Cleanup(func() {
		metrics.LastSuccessfulUpdateTimestamp.Set(0)
		metrics.LastFailedUpdateTimestamp.Set(0)
	})
}

func startMetricsUpdateRoutines(t *testing.T, r *Repo) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 2)
	go func() { done <- r.UpdateRoutine(ctx) }()
	go func() { done <- r.DimensionUpdateRoutine(ctx) }()

	t.Cleanup(func() {
		cancel()

		for range 2 {
			select {
			case err := <-done:
				assert.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Error("update routine did not stop")
			}
		}
	})
}

func runMetricsUpdate(t *testing.T, r *Repo) error {
	t.Helper()

	response := make(chan updateResponse, 1)
	select {
	case r.updateInProgressChannel <- response:
	case <-time.After(5 * time.Second):
		t.Fatal("update routine did not accept request")
	}

	select {
	case result := <-response:
		return result.err
	case <-time.After(5 * time.Second):
		t.Fatal("update routine did not complete")
		return nil
	}
}

func TestUpdateRoutineTimestamps(t *testing.T) {
	resetUpdateTimestamps(t)
	r := newMinimalRepo(t, "http://example.test/poll")
	startMetricsUpdateRoutines(t, r)

	for _, tc := range []struct {
		name       string
		pollStatus int
		repoStatus int
		body       string
		wantError  string
	}{
		{"initial poll failure", 500, 200, testRepoBody, "could not poll"},
		{"repeated poll failure", 500, 200, testRepoBody, "could not poll"},
		{"download failure", 200, 500, "", "bad response code"},
		{"parsing failure", 200, 200, "invalid json", "failed to deserialize"},
		{"dimension failure", 200, 200, `{"en":{"id":"root","uri":"/","linkId":"missing"}}`, "failed to update dimension"},
		{"successful load", 200, 200, testRepoBody, ""},
		{"failure after success", 500, 200, testRepoBody, "could not poll"},
		{"304 recovery", 304, 500, "", ""},
		{"failure before unchanged version", 500, 200, testRepoBody, "could not poll"},
		{"unchanged version recovery", 200, 500, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r.httpClient = &http.Client{Transport: updateMetricsTransport(func(req *http.Request) (*http.Response, error) {
				status, body := tc.repoStatus, tc.body
				if req.URL.Path == testPollPath {
					status, body = tc.pollStatus, "http://example.test/repo"
				}

				return &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			})}
			successBefore := testutil.ToFloat64(metrics.LastSuccessfulUpdateTimestamp)
			failureBefore := testutil.ToFloat64(metrics.LastFailedUpdateTimestamp)
			completedBefore := testutil.ToFloat64(metrics.UpdatesCompletedCounter.WithLabelValues())
			failedBefore := testutil.ToFloat64(metrics.UpdatesFailedCounter.WithLabelValues())
			start := float64(time.Now().UnixNano()) / 1e9
			err := runMetricsUpdate(t, r)
			end := float64(time.Now().UnixNano()) / 1e9
			success := testutil.ToFloat64(metrics.LastSuccessfulUpdateTimestamp)
			failure := testutil.ToFloat64(metrics.LastFailedUpdateTimestamp)

			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				assert.InDelta(t, successBefore, success, 0)
				assert.Greater(t, failure, failureBefore)
				assert.Greater(t, failure, success)
				assert.GreaterOrEqual(t, failure, start)
				assert.LessOrEqual(t, failure, end)
				assert.InDelta(t, failedBefore+1, testutil.ToFloat64(metrics.UpdatesFailedCounter.WithLabelValues()), 0)
				assert.InDelta(t, completedBefore, testutil.ToFloat64(metrics.UpdatesCompletedCounter.WithLabelValues()), 0)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, failureBefore, failure, 0)
				assert.Greater(t, success, successBefore)
				assert.Greater(t, success, failure)
				assert.GreaterOrEqual(t, success, start)
				assert.LessOrEqual(t, success, end)
				assert.InDelta(t, completedBefore+1, testutil.ToFloat64(metrics.UpdatesCompletedCounter.WithLabelValues()), 0)
				assert.InDelta(t, failedBefore, testutil.ToFloat64(metrics.UpdatesFailedCounter.WithLabelValues()), 0)
			}
		})
	}
}

func TestUpdateTimestampsIgnoreRestoreAndRejectedRequests(t *testing.T) {
	resetUpdateTimestamps(t)
	r := newMinimalRepo(t, "http://example.test/poll")

	_, err := r.tryUpdate()
	require.ErrorIs(t, err, ErrUpdateRejected)
	assert.Zero(t, testutil.ToFloat64(metrics.LastSuccessfulUpdateTimestamp))
	assert.Zero(t, testutil.ToFloat64(metrics.LastFailedUpdateTimestamp))

	require.NoError(t, r.history.Add(t.Context(), []byte(testRepoBody)))
	startMetricsUpdateRoutines(t, r)
	require.NoError(t, r.tryToRestoreCurrent(t.Context()))
	assert.Contains(t, r.Directory(), "dimension_foo")
	assert.Zero(t, testutil.ToFloat64(metrics.LastSuccessfulUpdateTimestamp))
	assert.Zero(t, testutil.ToFloat64(metrics.LastFailedUpdateTimestamp))
}

type failingMetricsStorage struct{ Storage }

func (s failingMetricsStorage) Write(context.Context, string, []byte) error {
	return io.ErrClosedPipe
}

func TestUpdateRoutineHistoryFailureStillRecordsSuccess(t *testing.T) {
	resetUpdateTimestamps(t)
	r := newMinimalRepo(t, "http://example.test/repo")
	r.poll = false
	r.history.storage = failingMetricsStorage{r.history.storage}
	r.httpClient = &http.Client{Transport: updateMetricsTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(testRepoBody)),
		}, nil
	})}
	startMetricsUpdateRoutines(t, r)

	historyBefore := testutil.ToFloat64(metrics.HistoryPersistFailedCounter.WithLabelValues())

	require.NoError(t, runMetricsUpdate(t, r))
	assert.Positive(t, testutil.ToFloat64(metrics.LastSuccessfulUpdateTimestamp))
	assert.Zero(t, testutil.ToFloat64(metrics.LastFailedUpdateTimestamp))
	assert.InDelta(t, historyBefore+1, testutil.ToFloat64(metrics.HistoryPersistFailedCounter.WithLabelValues()), 0)
}
