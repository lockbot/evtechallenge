package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// FHIRIngestionTotal tracks total FHIR resources ingested
	FHIRIngestionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fhir_ingestion_total",
			Help: "Total number of FHIR resources ingested",
		},
		[]string{"resource_type", "status"}, // "success", "skipped"
	)

	// FHIRIngestionDuration tracks ingestion duration
	FHIRIngestionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fhir_ingestion_duration_seconds",
			Help:    "Duration of FHIR ingestion operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource_type"},
	)

	// FHIRAPICallsTotal tracks total API calls to FHIR
	FHIRAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fhir_api_calls_total",
			Help: "Total number of API calls to FHIR",
		},
		[]string{"resource_type", "status"}, // "success", "error"
	)

	// FHIRAPICallDuration tracks FHIR API call duration
	FHIRAPICallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fhir_api_call_duration_seconds",
			Help:    "Duration of FHIR API calls in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource_type", "operation"}, // "bundle", "individual"
	)

	// HTTPFetchTotal tracks total HTTP fetch operations
	HTTPFetchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_fetch_total",
			Help: "Total number of HTTP fetch operations",
		},
		[]string{"operation", "status"}, // "bundle_fetch", "resource_fetch", "success", "error"
	)

	// HTTPFetchDuration tracks HTTP fetch duration
	HTTPFetchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_fetch_duration_seconds",
			Help:    "Duration of HTTP fetch operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// CouchbaseOperationsTotal tracks Couchbase operations
	CouchbaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "couchbase_operations_total",
			Help: "Total number of Couchbase operations",
		},
		[]string{"operation", "status"}, // "upsert", "get", "query", "success", "error"
	)

	// CouchbaseOperationDuration tracks Couchbase operation duration
	CouchbaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "couchbase_operation_duration_seconds",
			Help:    "Duration of Couchbase operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	GoMemstatsAllocBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fhir_go_memstats_alloc_bytes",
			Help: "Number of bytes allocated and still in use in FHIR service",
		},
		[]string{"service"},
	)

	GoMemstatsSysBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fhir_go_memstats_sys_bytes",
			Help: "Number of bytes obtained from system in FHIR service",
		},
		[]string{"service"},
	)

	GoThreads = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fhir_go_threads",
			Help: "Number of OS threads created in FHIR service",
		},
		[]string{"service"},
	)
)

// RecordFHIRIngestion records metrics for FHIR resource ingestion
func RecordFHIRIngestion(resourceType string, ingested, skipped int) {
	FHIRIngestionTotal.WithLabelValues(resourceType, "success").Add(float64(ingested))
	FHIRIngestionTotal.WithLabelValues(resourceType, "skipped").Add(float64(skipped))
}

// RecordFHIRAPICall records metrics for FHIR API calls
func RecordFHIRAPICall(resourceType, status string) {
	FHIRAPICallsTotal.WithLabelValues(resourceType, status).Inc()
}

// RecordFHIRAPICallDuration records FHIR API call duration
func RecordFHIRAPICallDuration(resourceType, operation string, duration time.Duration) {
	FHIRAPICallDuration.WithLabelValues(resourceType, operation).Observe(duration.Seconds())
}

// RecordHTTPFetch records HTTP fetch operations
func RecordHTTPFetch(operation, status string) {
	HTTPFetchTotal.WithLabelValues(operation, status).Inc()
}

// RecordHTTPFetchDuration records HTTP fetch duration
func RecordHTTPFetchDuration(operation string, duration time.Duration) {
	HTTPFetchDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordCouchbaseOperation records metrics for Couchbase operations
func RecordCouchbaseOperation(operation, status string) {
	CouchbaseOperationsTotal.WithLabelValues(operation, status).Inc()
}

// RecordCouchbaseOperationDuration records Couchbase operation duration
func RecordCouchbaseOperationDuration(operation string, duration time.Duration) {
	CouchbaseOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
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
