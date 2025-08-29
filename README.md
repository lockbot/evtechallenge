# evtechallenge
Stealth Company Technical Challenge

This project includes a Go application with comprehensive monitoring and observability using Prometheus for metrics and Elasticsearch for structured logging, all visualized through automatically provisioned Grafana dashboards.

## Quick Start

### 1. Start the Infrastructure
```bash
# Start all services (Elasticsearch, Prometheus, Grafana)
docker-compose up
```

Wait about 30 seconds for Elasticsearch to fully start up.


Your application will now send:
- **Structured logs** to Elasticsearch (ECS JSON format)
- **Metrics** to Prometheus via `/metrics` endpoint
- **Console logs** (pretty formatted)

### 2. Access Grafana
1. Open your browser to **http://localhost:3000**
2. Login with:
   - **Username**: `admin`
   - **Password**: `admin`
3. Skip password change (click "Skip")

## Pre-configured Dashboards

### 3 Go Application Metrics Dashboard
**Automatically available** - No setup required!

This dashboard shows real-time metrics from your Go application:
- **HTTP Request Rate** - Requests per second over time
- **Request Duration** - 95th and 50th percentile response times
- **Active Connections** - Current HTTP connections
- **Business Logic Metrics** - AllGood endpoint request rates
- **Status Code Distribution** - HTTP response codes breakdown
- **Endpoint Performance** - Request rates by endpoint

### 4 Application Logs Dashboard
**Automatically available** - No setup required!

This dashboard displays your application logs:
- **Live Log Stream** - Real-time log entries
- **Log Volume** - Log count over time
- **Log Levels** - Distribution of log levels (INFO, WARN, ERROR)

## Data Sources

Both data sources are **automatically configured**:

- **Prometheus** - Metrics collection from your Go app
- **Elasticsearch** - Structured logging with ECS format

## Testing the System

### 5 Generate Metrics
```bash
# Test the hello endpoint
curl http://localhost:8080/hello

# Test the all-good endpoint (success case)
curl -X POST http://localhost:8080/all-good \
  -H "Content-Type: application/json" \
  -d '{"yes": true}'

# Test the all-good endpoint (validation failure)
curl -X POST http://localhost:8080/all-good \
  -H "Content-Type: application/json" \
  -d '{"yes": false}'
```

### 6 View Raw Data
- **Prometheus metrics**: http://localhost:9090
- **Elasticsearch logs**: `curl "http://localhost:9200/logs*/_search?pretty"`

## Troubleshooting

### No dashboards appearing?
1. Check if Grafana is running: `docker-compose ps`
2. Verify data sources are working in Grafana → Configuration → Data Sources
3. Ensure your app is generating logs and metrics

### No metrics data?
1. Verify your app is running and accessible at `http://localhost:8080`
2. Check Prometheus targets: http://localhost:9090/targets
3. Ensure the `/metrics` endpoint is accessible: `curl http://localhost:8080/metrics`

### No logs appearing?
1. Check if your app is running and producing logs
2. Verify Elasticsearch is receiving data:
   ```bash
   curl "http://localhost:9200/logs*/_search?pretty"
   ```
3. Check the time range in Grafana (try "Last 1 hour")

### Starting fresh?
To completely reset everything:
```bash
docker-compose down --volumes
docker-compose up -d
```

## URLs Reference
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Elasticsearch**: http://localhost:9200
- **Your App**: http://localhost:8080

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Go App       │    │   Prometheus    │    │   Grafana       │
│   :8080        │───▶│   :9090         │───▶│   :3000         │
│   (metrics)    │    │   (scraping)    │    │   (visualize)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Console       │    │   Metrics       │    │   Dashboards    │
│   (logs)        │    │   (storage)     │    │   (auto-provisioned)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │
         ▼
┌─────────────────┐
│   Elasticsearch │
│   :9200         │
│   (logs)        │
└─────────────────┘
```

## Metrics Collected

### HTTP Metrics
- `http_requests_total` - Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds` - Request duration histogram
- `http_active_connections` - Current active connections

### Business Logic Metrics
- `allgood_requests_total` - AllGood endpoint requests by result (success, validation_failed, invalid_json, method_not_allowed)

### Go Runtime Metrics (Automatic)
- Memory usage, goroutines, GC stats
- Process metrics (CPU, file descriptors)
- Go-specific metrics (heap, stack, etc.)