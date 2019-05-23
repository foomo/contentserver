package metrics

import (
	"net/http"

	. "github.com/foomo/contentserver/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func PrometheusHandler() http.Handler {
	h := http.NewServeMux()
	h.Handle("/metrics", promhttp.Handler())
	return h
}

func RunPrometheusHandler(listener string) {
	Log.Info("starting prometheus handler on",
		zap.String("address", listener),
	)
	Log.Error("server failed: ",
		zap.Error(http.ListenAndServe(listener, PrometheusHandler())),
	)
}
