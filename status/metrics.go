package status

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricLabelHandler = "handler"
	MetricLabelStatus  = "status"
	MetricLabelSource  = "source"
	namespace          = "contentserver"
)

type Metrics struct {
	ServiceRequestCounter  *prometheus.CounterVec // count the number of requests for each service function
	ServiceRequestDuration *prometheus.SummaryVec // count the duration of requests for each service function
}

func NewMetrics() *Metrics {
	return &Metrics{
		ServiceRequestCounter:  serviceRequestCounter(),
		ServiceRequestDuration: serviceRequestDuration(),
	}
}

func serviceRequestCounter() *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "service_request_count",
			Help:      "count of requests per func",
		}, []string{MetricLabelHandler, MetricLabelStatus, MetricLabelSource})
	prometheus.MustRegister(vec)
	return vec
}

func serviceRequestDuration() *prometheus.SummaryVec {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: namespace,
		Name:      "service_request_duration_seconds",
		Help:      "seconds to unmarshal requests, execute a service function and marshal its reponses",
	}, []string{MetricLabelHandler, MetricLabelStatus, MetricLabelSource})
	prometheus.MustRegister(vec)
	return vec
}
