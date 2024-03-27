package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "contentserver"

	metricLabelHandler = "handler"
	metricLabelStatus  = "status"
	metricLabelSource  = "source"
	metricLabelRemote  = "remote"
)

// Metrics is the structure that holds all prometheus metrics
var (
	// InvalidNodeTreeRequests counts the number of invalid tree node requests
	InvalidNodeTreeRequests = newCounterVec(
		"invalid_node_tree_request_count",
		"Counts the number of invalid tree nodes for a specific node",
	)
	// ServiceRequestCounter count the number of requests for each service function
	ServiceRequestCounter = newCounterVec(
		"service_request_count",
		"Count of requests for each handler",
		metricLabelHandler, metricLabelStatus, metricLabelSource,
	)
	// ServiceRequestDuration observe the duration of requests for each service function
	ServiceRequestDuration = newSummaryVec(
		"service_request_duration_seconds",
		"Seconds to unmarshal requests, execute a service function and marshal its reponses",
		metricLabelHandler, metricLabelStatus, metricLabelSource,
	)
	// UpdatesCompletedCounter count the number of rejected updates
	UpdatesCompletedCounter = newCounterVec(
		"updates_completed_count",
		"Number of updates that were successfully completed",
	)
	// UpdatesFailedCounter count the number of updates that had an error
	UpdatesFailedCounter = newCounterVec(
		"updates_failed_count",
		"Number of updates that failed due to an error",
	)
	// UpdateDuration observe the duration of each repo.update() call
	UpdateDuration = newSummaryVec(
		"update_duration_seconds",
		"Duration in seconds for each successful repo.update() call",
	)
	// ContentRequestCounter count the total number of content requests
	ContentRequestCounter = newCounterVec(
		"content_request_count",
		"Number of requests for content",
		metricLabelSource,
	)
	// NumSocketsGauge keep track of the total number of open sockets
	NumSocketsGauge = newGaugeVec(
		"num_sockets_total",
		"Total number of currently open socket connections",
		metricLabelRemote,
	)
	// HistoryPersistFailedCounter count the number of failed attempts to persist the content history
	HistoryPersistFailedCounter = newCounterVec(
		"history_persist_failed_count",
		"Number of failures to store the content history on the filesystem",
	)
)

func newSummaryVec(name, help string, labels ...string) *prometheus.SummaryVec {
	vec := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		}, labels)
	prometheus.MustRegister(vec)
	return vec
}

func newCounterVec(name, help string, labels ...string) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		}, labels)
	prometheus.MustRegister(vec)
	return vec
}

func newGaugeVec(name, help string, labels ...string) *prometheus.GaugeVec {
	vec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		}, labels)
	prometheus.MustRegister(vec)
	return vec
}
