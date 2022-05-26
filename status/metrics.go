package status

import (
	"github.com/prometheus/client_golang/prometheus"
)

// M is the Metrics instance
var M = newMetrics()

const (
	namespace = "contentserver"

	metricLabelHandler = "handler"
	metricLabelStatus  = "status"
	metricLabelSource  = "source"
	metricLabelRemote  = "remote"
)

// Metrics is the structure that holds all prometheus metrics
type Metrics struct {
	ServiceRequestCounter       *prometheus.CounterVec // count the number of requests for each service function
	ServiceRequestDuration      *prometheus.SummaryVec // observe the duration of requests for each service function
	UpdatesRejectedCounter      *prometheus.CounterVec // count the number of completed updates
	UpdatesCompletedCounter     *prometheus.CounterVec // count the number of rejected updates
	UpdatesFailedCounter        *prometheus.CounterVec // count the number of updates that had an error
	UpdateDuration              *prometheus.SummaryVec // observe the duration of each repo.update() call
	ContentRequestCounter       *prometheus.CounterVec // count the total number of content requests
	NumSocketsGauge             *prometheus.GaugeVec   // keep track of the total number of open sockets
	HistoryPersistFailedCounter *prometheus.CounterVec // count the number of failed attempts to persist the content history
	InvalidNodeTreeRequests     *prometheus.CounterVec // counts the number of invalid tree node requests
}

// newMetrics can be used to instantiate a metrics instance
// since this function will also register each metric and metrics should only be registered once
// it is private
// the package exposes the initialized Metrics instance as the variable M.
func newMetrics() *Metrics {
	return &Metrics{
		InvalidNodeTreeRequests: newCounterVec("invalid_node_tree_request_count",
			"Counts the number of invalid tree nodes for a specific node"),
		ServiceRequestCounter: newCounterVec(
			"service_request_count",
			"Count of requests for each handler",
			metricLabelHandler, metricLabelStatus, metricLabelSource,
		),
		ServiceRequestDuration: newSummaryVec(
			"service_request_duration_seconds",
			"Seconds to unmarshal requests, execute a service function and marshal its reponses",
			metricLabelHandler, metricLabelStatus, metricLabelSource,
		),
		UpdatesRejectedCounter: newCounterVec(
			"updates_rejected_count",
			"Number of updates that were rejected because the queue was full",
		),
		UpdatesCompletedCounter: newCounterVec(
			"updates_completed_count",
			"Number of updates that were successfully completed",
		),
		UpdatesFailedCounter: newCounterVec(
			"updates_failed_count",
			"Number of updates that failed due to an error",
		),
		UpdateDuration: newSummaryVec(
			"update_duration_seconds",
			"Duration in seconds for each successful repo.update() call",
		),
		ContentRequestCounter: newCounterVec(
			"content_request_count",
			"Number of requests for content",
			metricLabelSource,
		),
		NumSocketsGauge: newGaugeVec(
			"num_sockets_total",
			"Total number of currently open socket connections",
			metricLabelRemote,
		),
		HistoryPersistFailedCounter: newCounterVec(
			"history_persist_failed_count",
			"Number of failures to store the content history on the filesystem",
		),
	}
}

/*
 *	Metric constructors
 */

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
