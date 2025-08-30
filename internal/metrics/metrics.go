package metrics

import (
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTP metrics are now managed by the MetricsManager singleton
// These variables are kept for backward compatibility but will be nil if metrics are disabled
var (
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	HTTPActiveConnections prometheus.Gauge
	AllGoodRequestsTotal  *prometheus.CounterVec
)

// initializeHTTPMetrics initializes HTTP metrics if they haven't been initialized yet
func initializeHTTPMetrics() {
	if HTTPRequestsTotal != nil {
		return // Already initialized
	}

	// HTTP request counter
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP request duration histogram
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	// Active HTTP connections gauge
	HTTPActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Business logic metrics
	AllGoodRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "allgood_requests_total",
			Help: "Total number of all-good requests",
		},
		[]string{"result"}, // "success", "validation_failed", "invalid_json"
	)

	// Register with the singleton registry
	mm := GetInstance()
	mm.registry.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		HTTPActiveConnections,
		AllGoodRequestsTotal,
	)
}

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	// Check if business metrics are enabled
	if os.Getenv("ENABLE_BUSINESS_METRICS") != "true" {
		return
	}

	// Initialize metrics if needed
	initializeHTTPMetrics()

	status := strconv.Itoa(statusCode)

	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
}

// RecordAllGoodRequest records business logic metrics
func RecordAllGoodRequest(result string) {
	// Check if business metrics are enabled
	if os.Getenv("ENABLE_BUSINESS_METRICS") != "true" {
		return
	}

	// Initialize metrics if needed
	initializeHTTPMetrics()

	AllGoodRequestsTotal.WithLabelValues(result).Inc()
}

// IncActiveConnections increments active connections
func IncActiveConnections() {
	// Check if business metrics are enabled
	if os.Getenv("ENABLE_BUSINESS_METRICS") != "true" {
		return
	}

	// Initialize metrics if needed
	initializeHTTPMetrics()

	HTTPActiveConnections.Inc()
}

// DecActiveConnections decrements active connections
func DecActiveConnections() {
	// Check if business metrics are enabled
	if os.Getenv("ENABLE_BUSINESS_METRICS") != "true" {
		return
	}

	// Initialize metrics if needed
	initializeHTTPMetrics()

	HTTPActiveConnections.Dec()
}
