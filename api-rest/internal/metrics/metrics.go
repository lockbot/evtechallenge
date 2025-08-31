package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request counter
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP request duration histogram
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	// Active HTTP connections gauge
	HTTPActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Business logic metrics
	AllGoodRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "allgood_requests_total",
			Help: "Total number of all-good requests",
		},
		[]string{"result"}, // "success", "validation_failed", "invalid_json"
	)
)

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)

	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
}

// RecordAllGoodRequest records business logic metrics
func RecordAllGoodRequest(result string) {
	AllGoodRequestsTotal.WithLabelValues(result).Inc()
}

// IncActiveConnections increments active connections
func IncActiveConnections() {
	HTTPActiveConnections.Inc()
}

// DecActiveConnections decrements active connections
func DecActiveConnections() {
	HTTPActiveConnections.Dec()
}
