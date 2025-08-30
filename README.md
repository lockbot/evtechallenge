# evtechallenge
Stealth Company Technical Challenge

This project includes a comprehensive healthcare data platform with FHIR data ingestion, document storage in Couchbase, and REST API services, all monitored through Prometheus and Elasticsearch with automatically provisioned Grafana dashboards.

## 🏗️ **Architecture Overview**

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Orchestrator  │    │   Couchbase     │    │   Elasticsearch │
│   (evtechallenge│    │   Database      │    │   (Logs)        │
│   -orch)        │───▶│   (FHIR Data)   │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Ingest        │    │   API Service   │    │   Prometheus    │
│   Service       │    │   (REST)        │    │   (Metrics)     │
│   (FHIR Data)   │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 **Quick Start**

### 1. **Start the Infrastructure**
```bash
# Start all services (Couchbase, Elasticsearch, Prometheus, Grafana)
# This will automatically set up Couchbase with the required bucket and user
docker-compose up -d

# Verify services are running
docker-compose ps
```

**What happens automatically:**
- Couchbase starts up
- Setup service creates the `evtechallenge` bucket
- Setup service creates the `evtechallenge_user` with proper permissions
- All services wait for setup to complete before starting

### 2. **Run Your Application**
```bash
# Run the orchestrator (this will launch ingest and API services)
go run main.go
```

The orchestrator will:
1. **Start the ingest service** - Fetches FHIR data from HAPI FHIR server
2. **Populate Couchbase** - Stores practitioners, patients, and encounters
3. **Start the API service** - Provides REST endpoints for the data
4. **Lock/unlock database** - Prevents API access during ingestion

## 📊 **Pre-configured Dashboards**

### 🚀 **Go Application Metrics Dashboard**
**Automatically available** - No setup required!

This dashboard shows real-time metrics from your Go services:
- **HTTP Request Rate** - Requests per second over time
- **Request Duration** - 95th and 50th percentile response times
- **Active Connections** - Current HTTP connections
- **Business Logic Metrics** - API endpoint performance
- **Status Code Distribution** - HTTP response codes breakdown
- **Endpoint Performance** - Request rates by endpoint

### 📊 **Application Logs Dashboard**
**Automatically available** - No setup required!

This dashboard displays your application logs:
- **Live Log Stream** - Real-time log entries from all services
- **Log Volume** - Log count over time
- **Service-specific logs** - Ingest, API, and orchestrator logs

## 🗄️ **Data Sources**

All data sources are **automatically configured**:

- **Prometheus** - Metrics collection from Go services and Couchbase
- **Elasticsearch** - Structured logging with ECS format
- **Couchbase** - Document database for FHIR resources

## 🔒 **Database Locking**

The system implements a **database locking mechanism**:

- **During ingestion**: Database is locked, API cannot read/write
- **After ingestion**: Database is unlocked, API can serve data
- **Lock expiration**: Automatic unlock after 1 hour (safety mechanism)
- **Lock status**: Stored in Couchbase as a document

## 📋 **FHIR Data Ingested**

The ingest service fetches data from [HAPI FHIR](https://hapi.fhir.org/baseR4):

1. **Practitioners** (502 records) - Healthcare providers
2. **Patients** (500 records) - Patient demographics and information  
3. **Encounters** (500 records) - Healthcare visits and interactions

## 🧪 **Testing the System**

### **Generate Metrics**
```bash
# Test the API endpoints (after ingestion completes)
curl http://localhost:8080/hello
curl http://localhost:8080/all-good -X POST -H "Content-Type: application/json" -d '{"yes": true}'
```

### **View Raw Data**
- **Prometheus metrics**: http://localhost:9090
- **Elasticsearch logs**: `curl "http://localhost:9200/logs*/_search?pretty"`
- **Couchbase UI**: http://localhost:8091 (Administrator/password)

## 🔍 **Troubleshooting**

### **No dashboards appearing?**
1. Check if Grafana is running: `docker-compose ps`
2. Verify data sources are working in Grafana → Configuration → Data Sources
3. Ensure your app is generating logs and metrics

### **Couchbase connection failed?**
1. **Check setup service logs**: `docker-compose logs evtechallenge-db-setup`
2. Wait longer - Couchbase takes time to start (check logs: `docker-compose logs evtechallenge-db`)
3. Verify Couchbase is accessible: `curl http://localhost:8091`
4. **Important**: The setup service must complete successfully before running the application

### **Ingest service not working?**
1. Verify FHIR server is accessible: `curl "https://hapi.fhir.org/baseR4/Patient?_count=1"`
2. Check Couchbase connection in ingest logs
3. Verify database lock status
4. Ensure Couchbase setup service completed successfully

### **Starting fresh?**
To completely reset everything:
```bash
docker-compose down --volumes
docker-compose up -d
# The setup service will automatically run and create everything needed
```

## 🌐 **URLs Reference**
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Elasticsearch**: http://localhost:9200
- **Couchbase UI**: http://localhost:8091 (Administrator/password)
- **Your API**: http://localhost:8080

## 📈 **Metrics Collected**

### **HTTP Metrics**
- `http_requests_total` - Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds` - Request duration histogram
- `http_active_connections` - Current active connections

### **Business Logic Metrics**
- `allgood_requests_total` - API endpoint requests by result
- FHIR data ingestion metrics (coming soon)

### **Infrastructure Metrics**
- Couchbase performance metrics
- Go runtime metrics (memory, goroutines, GC stats)

## 🔮 **Future Enhancements**

- **Valkey/Redis** - Caching layer for frequently accessed data
- **Advanced indexing** - Couchbase secondary indexes for complex queries
- **FHIR validation** - Enhanced data validation and consistency checks
- **Real-time updates** - Webhook support for FHIR data updates