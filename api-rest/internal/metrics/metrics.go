package metrics

import (
	"runtime"
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

	GoMemstatsAllocBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_go_memstats_alloc_bytes",
			Help: "Number of bytes allocated and still in use in API service",
		},
		[]string{"service"},
	)

	GoMemstatsSysBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_go_memstats_sys_bytes",
			Help: "Number of bytes obtained from system in API service",
		},
		[]string{"service"},
	)

	GoThreads = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_go_threads",
			Help: "Number of OS threads created in API service",
		},
		[]string{"service"},
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

// UpdateSystemMetrics updates Go runtime metrics with service label
func UpdateSystemMetrics(serviceName string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	GoMemstatsAllocBytes.WithLabelValues(serviceName).Set(float64(m.Alloc))
	GoMemstatsSysBytes.WithLabelValues(serviceName).Set(float64(m.Sys))
	GoThreads.WithLabelValues(serviceName).Set(float64(runtime.GOMAXPROCS(0)))
}

// StartSystemMetricsCollection starts a goroutine to collect system metrics
func StartSystemMetricsCollection(serviceName string) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			UpdateSystemMetrics(serviceName)
		}
	}()
}
