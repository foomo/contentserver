package status

import (
	"github.com/prometheus/client_golang/prometheus"
)

// M is the Metrics instance
var M = NewMetrics()

const (
	namespace = "contentserver"

	metricLabelHandler = "handler"
	metricLabelStatus  = "status"
	metricLabelSource  = "source"
	metricLabelRemote  = "remote"
	metricLabelError   = "error"
)

type Metrics struct {
	ServiceRequestCounter   *prometheus.CounterVec // count the number of requests for each service function
	ServiceRequestDuration  *prometheus.SummaryVec // observe the duration of requests for each service function
	UpdatesRejectedCounter  *prometheus.CounterVec // count the number of completed updates
	UpdatesCompletedCounter *prometheus.CounterVec // count the number of rejected updates
	UpdatesFailedCounter    *prometheus.CounterVec // count the number of updates that had an error
	UpdateDuration          *prometheus.SummaryVec // observe the duration of each repo.update() call
	ContentRequestCounter   *prometheus.CounterVec // count the total number of content requests
	NumSocketsGauge         *prometheus.GaugeVec   // keep track of the total number of open sockets
}

func NewMetrics() *Metrics {
	return &Metrics{
		ServiceRequestCounter:   serviceRequestCounter(),
		ServiceRequestDuration:  serviceRequestDuration(),
		UpdatesRejectedCounter:  updatesRejectedCounter(),
		UpdatesCompletedCounter: updatesCompletedCounter(),
		UpdatesFailedCounter:    updatesFailedCounter(),
		UpdateDuration:          updateDuration(),
		ContentRequestCounter:   contentRequestCounter(),
		NumSocketsGauge:         numSocketsGauge(),
	}
}

func serviceRequestCounter() *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "service_request_count",
			Help:      "Count of requests for each handler",
		}, []string{metricLabelHandler, metricLabelStatus, metricLabelSource})
	prometheus.MustRegister(vec)
	return vec
}

func serviceRequestDuration() *prometheus.SummaryVec {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: namespace,
		Name:      "service_request_duration_seconds",
		Help:      "Seconds to unmarshal requests, execute a service function and marshal its reponses",
	}, []string{metricLabelHandler, metricLabelStatus, metricLabelSource})
	prometheus.MustRegister(vec)
	return vec
}

func updatesRejectedCounter() *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "updates_rejected_count",
		Help:      "Number of updates that were rejected because the queue was full",
	}, []string{})
	prometheus.MustRegister(vec)
	return vec
}

func updatesCompletedCounter() *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "updates_completed_count",
		Help:      "Number of updates that were successfully completed",
	}, []string{})
	prometheus.MustRegister(vec)
	return vec
}

func updatesFailedCounter() *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "updates_failed_count",
		Help:      "Number of updates that failed due to an error",
	}, []string{metricLabelError})
	prometheus.MustRegister(vec)
	return vec
}

func updateDuration() *prometheus.SummaryVec {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: namespace,
		Name:      "update_duration_seconds",
		Help:      "Duration in seconds for each successful repo.update() call",
	}, []string{})
	prometheus.MustRegister(vec)
	return vec
}

func numSocketsGauge() *prometheus.GaugeVec {
	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "num_sockets_total",
		Help:      "Total number of currently open socket connections",
	}, []string{metricLabelRemote})
	prometheus.MustRegister(vec)
	return vec
}

func contentRequestCounter() *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "content_request_count",
		Help:      "Number of requests for content",
	}, []string{metricLabelSource})
	prometheus.MustRegister(vec)
	return vec
}
