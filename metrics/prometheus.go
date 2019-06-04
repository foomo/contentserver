package metrics

import (
	"net/http"

	. "github.com/foomo/contentserver/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const metricsRoute = "/metrics"

func RunPrometheusHandler(listener string) {
	Log.Info("starting prometheus handler on",
		zap.String("address", listener),
		zap.String("route", metricsRoute),
	)
	Log.Error("prometheus listener failed",
		zap.Error(http.ListenAndServe(listener, promhttp.Handler())),
	)
}
