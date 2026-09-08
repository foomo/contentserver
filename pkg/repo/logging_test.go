package repo

import (
	stdjson "encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/foomo/contentserver/requests"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLookupLogLevelsPreserveContent(t *testing.T) {
	var baseline []byte

	for _, tc := range []struct {
		name     string
		missing  zapcore.Level
		resolved zapcore.Level
		global   zapcore.Level
		defaults bool
	}{
		{"defaults", zap.ErrorLevel, zap.InfoLevel, zap.DebugLevel, true},
		{"debug", zap.DebugLevel, zap.DebugLevel, zap.DebugLevel, false},
		{"info", zap.InfoLevel, zap.InfoLevel, zap.DebugLevel, false},
		{"warn", zap.WarnLevel, zap.WarnLevel, zap.DebugLevel, false},
		{"error", zap.ErrorLevel, zap.ErrorLevel, zap.DebugLevel, false},
		{"debug filtered at info", zap.DebugLevel, zap.DebugLevel, zap.InfoLevel, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(tc.global)

			var opts []Option
			if !tc.defaults {
				opts = append(opts, WithLogLevelMissingNode(tc.missing), WithLogLevelResolved(tc.resolved))
			}

			r := New(zap.New(core), "", nil, opts...)
			require.NoError(t, r._updateDimension("en", &content.RepoNode{ID: "root", URI: "/"}))

			before := testutil.ToFloat64(metrics.InvalidNodeTreeRequests.WithLabelValues())
			reply, err := r.GetContent(&requests.Content{
				URI: "/",
				Env: &requests.Env{Dimensions: []string{"en"}},
				Nodes: map[string]*requests.Node{
					"missing": {ID: "absent"},
				},
			})
			require.NoError(t, err)
			encoded, err := stdjson.Marshal(reply)
			require.NoError(t, err)

			if baseline == nil {
				baseline = encoded
			}

			assert.Equal(t, baseline, encoded)
			assert.InDelta(t, before+1, testutil.ToFloat64(metrics.InvalidNodeTreeRequests.WithLabelValues()), 0)

			for message, level := range map[string]zapcore.Level{
				"Invalid tree node requested": tc.missing,
				"Content resolved":            tc.resolved,
			} {
				entries := logs.FilterMessage(message).All()
				if level < tc.global {
					assert.Empty(t, entries)
				} else {
					require.Len(t, entries, 1)
					assert.Equal(t, level, entries[0].Level)
				}
			}
		})
	}
}

func TestMissingNodeSettingDoesNotChangeOtherValidation(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	r := New(zap.New(core), "", nil, WithLogLevelMissingNode(zap.DebugLevel))
	before := testutil.ToFloat64(metrics.InvalidNodeTreeRequests.WithLabelValues())

	r.GetNodes(&requests.Nodes{
		Env: &requests.Env{},
		Nodes: map[string]*requests.Node{
			"bad":               {ID: ""},
			"missing dimension": {ID: "root", Dimension: "absent"},
		},
	})
	assert.Empty(t, logs.FilterMessage("Invalid tree node requested").All())
	assert.InDelta(t, before, testutil.ToFloat64(metrics.InvalidNodeTreeRequests.WithLabelValues()), 0)
	require.Len(t, logs.FilterMessage("invalid node request").All(), 1)
	assert.Equal(t, zap.WarnLevel, logs.FilterMessage("invalid node request").All()[0].Level)
	require.Len(t, logs.FilterMessage("could not get dimension root node").All(), 1)
	assert.Equal(t, zap.ErrorLevel, logs.FilterMessage("could not get dimension root node").All()[0].Level)
}

func TestUpdateRoutineLogOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		pollStatus int
		version    string
		body       string
		changed    bool
		wantError  bool
	}{
		{"applied revision", http.StatusOK, "old", testRepoBody, true, false},
		{"unchanged URL", http.StatusOK, "http://example.test/repo", "", false, false},
		{"not modified", http.StatusNotModified, "old", "", false, false},
		{"poll failure", http.StatusBadGateway, "old", "", false, true},
		{"invalid JSON", http.StatusOK, "old", "invalid", false, true},
		{"invalid dimension", http.StatusOK, "old", `{"en":{"id":"root","uri":"/","linkId":"missing"}}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			r := newMinimalRepo(t, "http://example.test/poll")
			r.l = zap.New(core)
			r.version = tc.version
			r.loaded.Store(true)
			r.httpClient = &http.Client{Transport: updateMetricsTransport(func(req *http.Request) (*http.Response, error) {
				status, body := http.StatusOK, tc.body
				if req.URL.Path == testPollPath {
					status, body = tc.pollStatus, "http://example.test/repo"
				}

				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			startMetricsUpdateRoutines(t, r)

			response := make(chan updateResponse, 1)
			select {
			case r.updateInProgressChannel <- response:
			case <-time.After(5 * time.Second):
				t.Fatal("update routine did not accept request")
			}

			select {
			case result := <-response:
				assert.Equal(t, tc.changed, result.changed)
				assert.Equal(t, tc.wantError, result.err != nil)
			case <-time.After(5 * time.Second):
				t.Fatal("update routine did not complete")
			}

			if tc.wantError {
				entries := logs.FilterLevelExact(zap.ErrorLevel).All()
				require.Len(t, entries, 1)
				assert.Equal(t, "update failed", entries[0].Message)
				assert.NotEmpty(t, entries[0].ContextMap()["run_id"])
			} else if tc.changed {
				entries := logs.FilterMessage("update success").All()
				require.Len(t, entries, 1)
				assert.Equal(t, zap.InfoLevel, entries[0].Level)
				assert.Equal(t, "http://example.test/repo", entries[0].ContextMap()["revision"])
			} else {
				assert.Empty(t, logs.FilterLevelExact(zap.InfoLevel).All())
				entries := logs.FilterMessage("repo is up to date").All()
				require.Len(t, entries, 1)
				assert.Equal(t, zap.DebugLevel, entries[0].Level)
			}
		})
	}
}

func TestRejectedUpdateLog(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	r := New(zap.New(core), "", nil)
	reply := r.Update(t.Context())
	assert.False(t, reply.Success)
	assert.Empty(t, reply.ErrorMessage)
	assert.Empty(t, logs.FilterLevelExact(zap.ErrorLevel).All())
	entries := logs.FilterLevelExact(zap.WarnLevel).All()
	require.Len(t, entries, 1)
	assert.Equal(t, "update request rejected: update already in progress", entries[0].Message)
}

func TestPublicUpdateDoesNotDuplicateFailureLog(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	r := newMinimalRepo(t, "http://example.test/poll")
	r.l = zap.New(core)
	require.NoError(t, r.history.Add(t.Context(), []byte(testRepoBody)))

	r.httpClient = &http.Client{Transport: updateMetricsTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	startMetricsUpdateRoutines(t, r)

	require.Eventually(t, func() bool {
		response := r.Update(t.Context())
		return !response.Success && response.ErrorMessage != ""
	}, 5*time.Second, time.Millisecond)

	entries := logs.FilterLevelExact(zap.ErrorLevel).All()
	require.Len(t, entries, 1)
	assert.Equal(t, "update failed", entries[0].Message)
	assert.Len(t, logs.FilterMessage("Successfully restored current repository from local history").All(), 1)
}
