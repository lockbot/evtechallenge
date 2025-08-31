package metrics

import (
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

// RecordCouchbaseOperation records metrics for Couchbase operations
func RecordCouchbaseOperation(operation, status string) {
	CouchbaseOperationsTotal.WithLabelValues(operation, status).Inc()
}
