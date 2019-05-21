package status

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricLabelHandler = "handler"
	MetricLabelStatus  = "status"
)

type Metrics struct {
	ServiceRequestCounter  *prometheus.CounterVec // count the number of requests for each service function
	ServiceRequestDuration *prometheus.SummaryVec // count the duration of requests for each service function
}

func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		ServiceRequestCounter:  serviceRequestCounter("api", namespace),
		ServiceRequestDuration: serviceRequestDuration("api", namespace),
	}
}

func serviceRequestCounter(subsystem, namespace string) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "count_service_requests",
			Help:      "count of requests per func",
		}, []string{MetricLabelHandler, MetricLabelStatus})
	prometheus.MustRegister(vec)
	return vec
}

func serviceRequestDuration(subsystem, namespace string) *prometheus.SummaryVec {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "time_nanoseconds",
		Help:      "nanoseconds to unmarshal requests, execute a service function and marshal its reponses",
	}, []string{MetricLabelHandler, MetricLabelStatus})
	prometheus.MustRegister(vec)
	return vec
}
