package metrics

import (
	"fmt"
	"net/http"

	"github.com/foomo/contentserver/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func PrometheusHandler() http.Handler {
	h := http.NewServeMux()
	h.Handle("/metrics", promhttp.Handler())
	return h
}

func RunPrometheusHandler(listener string) {
	log.Notice(fmt.Sprintf("starting prometheus handler on address '%s'", listener))
	log.Error(http.ListenAndServe(listener, PrometheusHandler()))
}
