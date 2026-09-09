package handler

import (
	"time"

	"github.com/foomo/contentserver/pkg/metrics"
)

// Route type
type Route string

const (
	// RouteGetURIs get uris, many at once, to keep it fast
	RouteGetURIs Route = "getURIs"
	// RouteGetContent get (site) content
	RouteGetContent Route = "getContent"
	// RouteGetNodes get nodes
	RouteGetNodes Route = "getNodes"
	// RouteUpdate update repo
	RouteUpdate Route = "update"
	// RouteGetRepo get the whole repo
	RouteGetRepo Route = "getRepo"
)

func observeRequest(route Route, source string, start time.Time, failed bool) {
	result := "success"
	if failed {
		result = "error"
	}

	metrics.ServiceRequestCounter.WithLabelValues(string(route), result, source).Inc()
	metrics.ServiceRequestDuration.WithLabelValues(string(route), result, source).Observe(time.Since(start).Seconds())
}
