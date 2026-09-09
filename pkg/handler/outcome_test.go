package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/foomo/contentserver/pkg/repo"
	"github.com/foomo/contentserver/responses"
	"github.com/foomo/keel/net/http/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestOutcomes(t *testing.T) {
	for _, source := range []string{"webserver", sourceSocketServer} {
		t.Run(source, func(t *testing.T) {
			for _, tc := range []struct {
				name    string
				route   Route
				payload string
				failed  bool
				warning bool
				message string
			}{
				{"valid URIs", RouteGetURIs, `{}`, false, false, `"reply":{}`},
				{"invalid JSON", RouteGetURIs, `{`, true, true, "could not read incoming json"},
				{"invalid content", RouteGetContent, `{}`, true, true, "request URI must not be empty"},
				{"missing environment", RouteGetContent, `{"URI":"/a"}`, true, true, "request.Env must not be nil"},
				{"unknown route", "unknown", `{}`, true, true, "unknown"},
				{"rejected update", RouteUpdate, `{}`, true, true, `"success":false`},
				{"missing content", RouteGetContent, `{"URI":"/missing","env":{"dimensions":["test"]}}`, false, false, `"status":404`},
				{"empty nodes", RouteGetNodes, `{}`, false, false, `"reply":{}`},
				{"invalid node ID", RouteGetNodes, `{"nodes":{"invalid":{}}}`, true, true, `"reply":{}`},
				{"invalid node name", RouteGetNodes, `{"nodes":{"":null}}`, true, true, `"reply":{}`},
				{"missing node", RouteGetNodes, `{"nodes":{"missing":{"id":"missing","dimension":"test"}},"env":{}}`, false, false, `"missing":null`},
				{"null node", RouteGetNodes, `{"nodes":{"missing":null},"env":{}}`, true, true, "must not be nil"},
				{"node without environment", RouteGetNodes, `{"nodes":{"missing":{"id":"missing"}}}`, true, true, "must not be nil"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					core, logs := observer.New(zap.DebugLevel)
					l := zap.New(core)
					r := repo.New(l, "", nil, repo.WithLogLevelMissingNode(zap.DebugLevel))
					r.SetDirectory(map[string]*repo.Dimension{"test": {Directory: map[string]*content.RepoNode{}, URIDirectory: map[string]*content.RepoNode{}}})

					beforeSuccess := requestCount(tc.route, "success", source)
					beforeError := requestCount(tc.route, "error", source)

					status := "success"
					if tc.failed {
						status = "error"
					}

					beforeDuration := durationCount(t, tc.route, status, source)
					reply := executeTransport(t, source, l, r, tc.route, tc.payload)
					require.Contains(t, string(reply), tc.message)

					if tc.failed {
						require.Equal(t, beforeError+1, requestCount(tc.route, "error", source))
						require.Equal(t, beforeSuccess, requestCount(tc.route, "success", source))
					} else {
						require.Equal(t, beforeSuccess+1, requestCount(tc.route, "success", source))
						require.Equal(t, beforeError, requestCount(tc.route, "error", source))
					}

					require.Empty(t, logs.FilterLevelExact(zap.ErrorLevel).All())

					if tc.warning {
						require.NotEmpty(t, logs.FilterLevelExact(zap.WarnLevel).All())
					}

					require.Equal(t, beforeDuration+1, durationCount(t, tc.route, status, source))
				})
			}
		})
	}
}

func TestEncodingFailure(t *testing.T) {
	for _, source := range []string{"webserver", sourceSocketServer} {
		t.Run(source, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			l := zap.New(core)
			r := repo.New(l, "", nil)
			node := &content.RepoNode{ID: "test", URI: "/a", Data: map[string]any{"invalid": make(chan bool)}}
			r.SetDirectory(map[string]*repo.Dimension{"test": {Directory: map[string]*content.RepoNode{"test": node}, URIDirectory: map[string]*content.RepoNode{"/a": node}}})

			before := requestCount(RouteGetContent, "error", source)
			beforeSuccess := requestCount(RouteGetContent, "success", source)
			payload := `{"URI":"/a","env":{"dimensions":["test"]}}`

			if source == "webserver" {
				w := httptest.NewRecorder()
				NewHTTP(l, r).ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/contentserver/getContent", strings.NewReader(payload)))
				require.Equal(t, http.StatusInternalServerError, w.Code)
			} else {
				NewSocket(l, r).execute(RouteGetContent, []byte(payload))
			}

			require.Equal(t, before+1, requestCount(RouteGetContent, "error", source))
			require.Equal(t, beforeSuccess, requestCount(RouteGetContent, "success", source))
			require.Len(t, logs.FilterMessage("could not encode reply").FilterLevelExact(zap.ErrorLevel).All(), 1)
		})
	}
}

func TestGetRepoOutcomes(t *testing.T) {
	for _, source := range []string{"webserver", sourceSocketServer} {
		for _, failed := range []bool{false, true} {
			t.Run(source+"/failed="+strconv.FormatBool(failed), func(t *testing.T) {
				core, logs := observer.New(zap.DebugLevel)
				l := zap.New(core)
				history, err := repo.NewHistory(l, repo.HistoryWithHistoryDir(t.TempDir()))
				require.NoError(t, err)

				r := repo.New(l, "", history)
				status := "error"

				if !failed {
					r.SetJSONBuffer(bytes.NewBufferString(`{}`))

					status = "success"
				}

				before := requestCount(RouteGetRepo, status, source)
				if source == "webserver" {
					w := httptest.NewRecorder()
					NewHTTP(l, r).ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/contentserver/getRepo", strings.NewReader(`{}`)))

					if failed {
						require.Equal(t, http.StatusInternalServerError, w.Code)
					} else {
						require.JSONEq(t, `{"reply":{}}`, w.Body.String())
					}
				} else {
					reply := NewSocket(l, r).execute(RouteGetRepo, []byte(`{}`))
					if failed {
						require.Contains(t, string(reply), `"code":5`)
					} else {
						require.JSONEq(t, `{"reply":{}}`, string(reply))
					}
				}

				require.Equal(t, before+1, requestCount(RouteGetRepo, status, source))

				if failed {
					require.Len(t, logs.FilterMessage("failed to write repo bytes").FilterLevelExact(zap.ErrorLevel).All(), 1)
				}
			})
		}
	}
}

func TestFailedUpdate(t *testing.T) {
	for _, source := range []string{"webserver", sourceSocketServer} {
		t.Run(source, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			l := zap.New(core)

			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer upstream.Close()

			history, err := repo.NewHistory(l, repo.HistoryWithHistoryDir(t.TempDir()))
			require.NoError(t, err)

			r := repo.New(l, upstream.URL, history)
			ctx, cancel := context.WithCancel(t.Context())

			done := make(chan error, 1)
			go func() { done <- r.UpdateRoutine(ctx) }()

			defer func() {
				cancel()
				require.NoError(t, <-done)
			}()

			require.Eventually(t, func() bool {
				before := requestCount(RouteUpdate, "error", source)
				beforeSuccess := requestCount(RouteUpdate, "success", source)

				reply := executeTransport(t, source, l, r, RouteUpdate, `{}`)
				if source == sourceSocketServer {
					reply = reply[bytes.IndexByte(reply, '{'):]
				}

				var envelope struct {
					Reply responses.Update `json:"reply"`
				}
				require.NoError(t, json.Unmarshal(reply, &envelope))
				require.False(t, envelope.Reply.Success)
				require.Equal(t, before+1, requestCount(RouteUpdate, "error", source))
				require.Equal(t, beforeSuccess, requestCount(RouteUpdate, "success", source))

				return envelope.Reply.ErrorMessage != ""
			}, time.Second, time.Millisecond)
			require.Len(t, logs.FilterMessage("update failed").All(), 1)
			require.Empty(t, logs.FilterMessage("an API error occurred").All())
		})
	}
}

func TestMalformedTransport(t *testing.T) {
	t.Run("http", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			method string
			body   io.ReadCloser
			status int
		}{
			{"method", http.MethodGet, nil, http.StatusMethodNotAllowed},
			{"empty body", http.MethodPost, nil, http.StatusBadRequest},
			{"read failure", http.MethodPost, failingBody{}, http.StatusBadRequest},
		} {
			t.Run(tc.name, func(t *testing.T) {
				core, logs := observer.New(zap.DebugLevel)
				l := zap.New(core)
				request := httptest.NewRequestWithContext(t.Context(), tc.method, "/contentserver/getURIs", nil)
				request.Body = tc.body
				w := httptest.NewRecorder()
				before := requestCount(RouteGetURIs, "error", "webserver")

				NewHTTP(l, nil).ServeHTTP(w, request)
				require.Equal(t, tc.status, w.Code)
				require.Equal(t, http.StatusText(tc.status)+"\n", w.Body.String())
				require.Equal(t, before+1, requestCount(RouteGetURIs, "error", "webserver"))
				require.Empty(t, logs.FilterLevelExact(zap.ErrorLevel).All())
				require.Len(t, logs.FilterLevelExact(zap.WarnLevel).All(), 1)
			})
		}
	})
	t.Run("socket", func(t *testing.T) {
		for _, payload := range []string{"getURIs:nope{", "getURIs:0{", "getURIs:5{"} {
			t.Run(payload, func(t *testing.T) {
				core, logs := observer.New(zap.DebugLevel)
				before := requestCount(RouteGetURIs, "error", sourceSocketServer)

				NewSocket(zap.New(core), nil).Serve(&testConn{Reader: strings.NewReader(payload)})
				require.Equal(t, before+1, requestCount(RouteGetURIs, "error", sourceSocketServer))
				require.Empty(t, logs.FilterLevelExact(zap.ErrorLevel).All())
				require.Len(t, logs.FilterLevelExact(zap.WarnLevel).All(), 1)
			})
		}
	})
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("broken request body") }
func (failingBody) Close() error             { return nil }

func durationCount(t *testing.T, route Route, status, source string) uint64 {
	t.Helper()

	metric, ok := metrics.ServiceRequestDuration.WithLabelValues(string(route), status, source).(prometheus.Metric)
	require.True(t, ok)

	value := &dto.Metric{}
	require.NoError(t, metric.Write(value))

	return value.GetSummary().GetSampleCount()
}

func TestNullRequests(t *testing.T) {
	for _, source := range []string{"webserver", sourceSocketServer} {
		for _, route := range []Route{RouteGetURIs, RouteGetNodes, RouteGetContent, RouteUpdate} {
			t.Run(source+"/"+string(route), func(t *testing.T) {
				core, logs := observer.New(zap.DebugLevel)
				l := zap.New(core)
				r := repo.New(l, "", nil)
				before := requestCount(route, "error", source)

				var (
					reply []byte
					err   error
				)
				if source == "webserver" {
					reply, err = (&HTTP{l: l}).handleRequest(t.Context(), r, route, []byte("null"), source)
				} else {
					reply, err = (&Socket{l: l}).handleRequest(r, route, []byte("null"), source)
				}

				require.NoError(t, err)

				switch route {
				case RouteUpdate:
					require.Contains(t, string(reply), `"success":false`)
				case RouteGetContent:
					require.JSONEq(t, `{"reply":{"status":500,"code":3,"message":"internal error repo.GetContent invalid request: request must not be nil"}}`, string(reply))
				default:
					require.Contains(t, string(reply), `"status":500`)
				}

				require.Equal(t, before+1, requestCount(route, "error", source))
				require.Empty(t, logs.FilterLevelExact(zap.ErrorLevel).All())
			})
		}
	}
}

func TestMalformedHTTPWithRequestLogger(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	l := zap.New(core)
	w := httptest.NewRecorder()
	h := middleware.Logger()(l, "contentserver", NewHTTP(l, nil))
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/contentserver/getURIs", nil))

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	require.Empty(t, logs.FilterLevelExact(zap.ErrorLevel).All())
	warnings := logs.FilterLevelExact(zap.WarnLevel).All()
	require.Len(t, warnings, 1)
	require.Equal(t, "handled http request", warnings[0].Message)
	fields := warnings[0].ContextMap()
	require.EqualValues(t, http.StatusMethodNotAllowed, fields["http_response_status_code"])
	require.Contains(t, fields, "duration")
	require.Contains(t, fields, "error_type")
	require.Equal(t, "http server error: method not allowed", fields["error_message"])
}

func requestCount(route Route, status, source string) uint64 {
	return uint64(testutil.ToFloat64(metrics.ServiceRequestCounter.WithLabelValues(string(route), status, source)))
}

func executeTransport(t *testing.T, source string, l *zap.Logger, r *repo.Repo, route Route, payload string) []byte {
	t.Helper()

	if source == "webserver" {
		w := httptest.NewRecorder()
		NewHTTP(l, r).ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/contentserver/"+string(route), strings.NewReader(payload)))
		require.Equal(t, http.StatusOK, w.Code)

		return w.Body.Bytes()
	}

	c := &testConn{Reader: strings.NewReader(string(route) + ":" + strconv.Itoa(len(payload)) + payload)}
	NewSocket(l, r).Serve(c)

	return c.Bytes()
}

type testConn struct {
	*strings.Reader
	bytes.Buffer
}

func (c *testConn) Read(p []byte) (int, error)     { return c.Reader.Read(p) }
func (c *testConn) Write(p []byte) (int, error)    { return c.Buffer.Write(p) }
func (*testConn) Close() error                     { return nil }
func (*testConn) LocalAddr() net.Addr              { return &net.UnixAddr{Name: "local", Net: "unix"} }
func (*testConn) RemoteAddr() net.Addr             { return &net.UnixAddr{Name: "remote", Net: "unix"} }
func (*testConn) SetDeadline(time.Time) error      { return nil }
func (*testConn) SetReadDeadline(time.Time) error  { return nil }
func (*testConn) SetWriteDeadline(time.Time) error { return nil }
