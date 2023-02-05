package services

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricRequeueServiceCount is the number of times a particular service has been requeued.
var MetricRequeueServiceCount = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "",
	Subsystem: "",
	Name:      "requeue_service_total",
	Help:      "A metric that captures the number of times a service is requeued after failing to sync"},
	[]string{
		"name",
	},
)

// MetricSyncServiceCount is the number of times a particular service has been synced.
var MetricSyncServiceCount = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "",
	Subsystem: "",
	Name:      "sync_service_total",
	Help:      "A metric that captures the number of times a service is synced"},
	[]string{
		"name",
	},
)

// MetricSyncServiceLatency is the time taken to sync a service with the OVN load balancers.
var MetricSyncServiceLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "",
	Subsystem: "",
	Name:      "sync_service_latency_seconds",
	Help:      "The latency of syncing a service",
	Buckets:   prometheus.ExponentialBuckets(.1, 2, 15)},
	[]string{"name"},
)
