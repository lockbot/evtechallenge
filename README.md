# evtechallenge
Stealth Company Technical Challenge

# Viewing Elasticsearch Logs in Grafana

This guide shows how to run the project and view your application logs in Grafana.

## Quick Start

### 1. Start the Infrastructure
```bash
# Start Elasticsearch and Grafana
docker-compose up -d

# Verify services are running
docker-compose ps
```

Wait about 30 seconds for Elasticsearch to fully start up.

### 2. Run Your Application
```bash
# Run your Go application (this will send logs to Elasticsearch)
go run main.go
```

Your application will now send structured logs to both:
- **Console** (pretty formatted)
- **Elasticsearch** (ECS JSON format)

### 3. Access Grafana
1. Open your browser to **http://localhost:3000**
2. Login with:
   - **Username**: `admin`
   - **Password**: `admin`
3. Skip password change (click "Skip")

### 4. Configure Elasticsearch Data Source

#### Add Data Source:
1. Click the **⚙️ gear icon** (Configuration) in the left sidebar
2. Click **Data Sources**
3. Click **Add data source**
4. Select **Elasticsearch**

#### Configure Settings:
- **URL**: `http://elasticsearch:9200`
- **Index name**: `logs*`
- **Time field name**: `@timestamp`
- Leave other settings as default

#### Save:
1. Click **Save & test**
2. You should see a green "Data source is working" message

### 5. View Your Logs

#### Create a Dashboard:
1. Click the **+** icon in the left sidebar
2. Click **Dashboard**
3. Click **Add visualization**
4. Select your **Elasticsearch** data source

#### Configure the Panel:
1. In the **Query** section:
   - Leave query as `*` (shows all logs)
   - Set **Time field** to `@timestamp`
2. In the **Panel options**:
   - Change **Visualization type** to **Logs**
3. Set time range to **Last 15 minutes** (top right corner)

#### View Results:
Your application logs should now appear in the panel! You can:
- **Search logs**: Use the query box to filter (e.g., `message:hello`)
- **View JSON**: Click on any log entry to see the full ECS structured data
- **Time filter**: Adjust the time range as needed

## Troubleshooting

### No logs appearing?
1. Check if your app is running and producing logs
2. Verify Elasticsearch is receiving data:
   ```bash
   curl "http://localhost:9200/logs*/_search?pretty"
   ```
3. Check the time range in Grafana (try "Last 1 hour")

### Elasticsearch connection failed?
1. Make sure Elasticsearch is running: `docker-compose ps`
2. Wait a bit longer - Elasticsearch takes time to start
3. Use `http://elasticsearch:9200` as the URL (not localhost)

### Starting fresh?
To completely reset everything:
```bash
docker-compose down --volumes
docker-compose up -d
```

## URLs Reference
- **Grafana**: http://localhost:3000 (admin/admin)
- **Elasticsearch**: http://localhost:9200
- **Check logs directly**: `curl "http://localhost:9200/logs*/_search?pretty"`