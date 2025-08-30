package metrics

import (
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// FHIR metrics are now managed by the MetricsManager singleton
// These variables are kept for backward compatibility but will be nil if metrics are disabled
var (
	fhirIngestionDuration   *prometheus.HistogramVec
	fhirIngestionTotal      *prometheus.CounterVec
	fhirResourcesProcessed  *prometheus.CounterVec
	fhirResourcesStored     *prometheus.CounterVec
	fhirResourcesFailed     *prometheus.CounterVec
	fhirHTTPRequestsTotal   *prometheus.CounterVec
	fhirHTTPRequestDuration *prometheus.HistogramVec
)

// initializeFHIRMetrics initializes FHIR metrics if they haven't been initialized yet
func initializeFHIRMetrics() {
	if fhirIngestionDuration != nil {
		return // Already initialized
	}

	// FHIR ingestion metrics
	fhirIngestionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fhir_ingestion_duration_seconds",
			Help:    "Time spent ingesting FHIR resources",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "status"},
	)

	fhirIngestionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fhir_ingestion_total",
			Help: "Total number of FHIR ingestion operations",
		},
		[]string{"endpoint", "status"},
	)

	fhirResourcesProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fhir_resources_processed_total",
			Help: "Total number of FHIR resources processed",
		},
		[]string{"endpoint", "collection"},
	)

	fhirResourcesStored = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fhir_resources_stored_total",
			Help: "Total number of FHIR resources successfully stored",
		},
		[]string{"endpoint", "collection"},
	)

	fhirResourcesFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fhir_resources_failed_total",
			Help: "Total number of FHIR resources that failed to store",
		},
		[]string{"endpoint", "collection", "error_type"},
	)

	// HTTP client metrics
	fhirHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fhir_http_requests_total",
			Help: "Total number of HTTP requests to FHIR server",
		},
		[]string{"endpoint", "status_code"},
	)

	fhirHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fhir_http_request_duration_seconds",
			Help:    "Time spent making HTTP requests to FHIR server",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// Register with the singleton registry
	mm := GetInstance()
	mm.registry.MustRegister(
		fhirIngestionDuration,
		fhirIngestionTotal,
		fhirResourcesProcessed,
		fhirResourcesStored,
		fhirResourcesFailed,
		fhirHTTPRequestsTotal,
		fhirHTTPRequestDuration,
	)
}

// RecordIngestionMetrics records metrics for FHIR ingestion operations
func RecordIngestionMetrics(endpoint, collection string, startTime time.Time, status string, resourceCount int, storedCount int, failedCount int) {
	// Check if business metrics are enabled
	if os.Getenv("ENABLE_BUSINESS_METRICS") != "true" {
		return
	}

	// Initialize metrics if needed
	initializeFHIRMetrics()

	duration := time.Since(startTime).Seconds()

	// Record ingestion duration and total
	fhirIngestionDuration.WithLabelValues(endpoint, status).Observe(duration)
	fhirIngestionTotal.WithLabelValues(endpoint, status).Inc()

	// Record resource counts
	fhirResourcesProcessed.WithLabelValues(endpoint, collection).Add(float64(resourceCount))
	fhirResourcesStored.WithLabelValues(endpoint, collection).Add(float64(storedCount))
	if failedCount > 0 {
		fhirResourcesFailed.WithLabelValues(endpoint, collection, "storage_error").Add(float64(failedCount))
	}
}

// RecordHTTPMetrics records metrics for HTTP operations
func RecordHTTPMetrics(endpoint string, startTime time.Time, statusCode int) {
	// Check if business metrics are enabled
	if os.Getenv("ENABLE_BUSINESS_METRICS") != "true" {
		return
	}

	// Initialize metrics if needed
	initializeFHIRMetrics()

	duration := time.Since(startTime).Seconds()

	fhirHTTPRequestsTotal.WithLabelValues(endpoint, fmt.Sprintf("%d", statusCode)).Inc()
	fhirHTTPRequestDuration.WithLabelValues(endpoint).Observe(duration)
}
