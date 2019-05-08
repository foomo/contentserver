package metrics

import (
	"fmt"
	"github.com/foomo/contentserver/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const (
	DefaultPrometheusListener = ":9200"
)

func PrometheusHandler() http.Handler {
	h := http.NewServeMux()
	h.Handle("/metrics", promhttp.Handler())
	return h
}

func RunPrometheusHandler(listener string) {
	log.Notice(fmt.Sprintf("starting prometheus handler on address '%s'", DefaultPrometheusListener))
	log.Error(http.ListenAndServe(listener, PrometheusHandler()))
}

func RunPrometheusHandlerOnDefaultAddress() {
	RunPrometheusHandler(DefaultPrometheusListener)
}
