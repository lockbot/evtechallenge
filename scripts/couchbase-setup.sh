#!/bin/sh
set -e

echo "Starting Couchbase setup..."

# Wait for Couchbase to be fully ready
echo "Waiting for Couchbase UI..."
attempts=0; max=13
until curl -sf http://evt-db:8091 > /dev/null; do
  attempts=$((attempts+1))
  if [ $attempts -ge $max ]; then echo "ERROR: Couchbase UI not ready"; exit 1; fi
  echo "Waiting for Couchbase... ($attempts/$max)"
  sleep 3
done

# Check if database is already initialized
if [ -f "/tmp/db-init-flag/initialized" ]; then
  echo "Database already initialized, skipping setup..."
  exit 0
fi

attempts=0; max=7
until curl -sf http://evt-db:8091/pools > /dev/null; do
  attempts=$((attempts+1))
  if [ $attempts -ge $max ]; then echo "ERROR: Couchbase UI not ready"; exit 1; fi
  echo "Waiting for Couchbase... ($attempts/$max)"
  sleep 3
done
echo "Couchbase is ready!"

# Resolve DB container IP for advertised hostname
DB_IP=$(getent hosts evt-db | head -n1 | cut -d" " -f1 || true)
if [ -z "$DB_IP" ]; then DB_IP=$(getent ahostsv4 evt-db | head -n1 | cut -d" " -f1 || true); fi
if [ -z "$DB_IP" ]; then echo "ERROR: Unable to resolve DB IP"; exit 1; fi
echo "Using node IP: $DB_IP"

echo "Running node-init..."
/opt/evt-db/bin/evt-db-cli node-init -c evt-db --node-init-hostname "$DB_IP" -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD"

echo "Initializing cluster (data,index,query)..."
/opt/evt-db/bin/evt-db-cli cluster-init -c evt-db \
  --cluster-username "$COUCHBASE_ADMINISTRATOR_USERNAME" --cluster-password "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --services data,index,query --cluster-ramsize 512 \
  --cluster-index-ramsize 256 --index-storage-setting default

echo "Creating bucket..."
/opt/evt-db/bin/evt-db-cli bucket-create -c evt-db -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --bucket "$COUCHBASE_BUCKET" --bucket-type evt-db --bucket-ramsize 256 --wait

echo "Creating application user..."
/opt/evt-db/bin/evt-db-cli user-manage -c evt-db -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --set --rbac-username "$COUCHBASE_USERNAME" --rbac-password "$COUCHBASE_PASSWORD" \
  --roles bucket_full_access["$COUCHBASE_BUCKET"] --auth-domain local

echo "Granting admin role to application user..."
# This gives the application user full permissions to create scopes and collections
/opt/evt-db/bin/evt-db-cli user-manage -c evt-db -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --set --rbac-username "$COUCHBASE_USERNAME" --rbac-password "$COUCHBASE_PASSWORD" \
  --roles "admin" --auth-domain local

# Wait for bucket to be ready for queries
echo "Waiting for bucket to be ready for queries..."
sleep 10

# Create secondary indexes for efficient querying
echo "Creating secondary indexes..."

# Index on resourceType for filtering by resource type
curl -X POST "http://evt-db:8093/query/service" \
  -u "$COUCHBASE_USERNAME:$COUCHBASE_PASSWORD" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "statement=CREATE INDEX idx_resourceType ON \`$COUCHBASE_BUCKET\`(resourceType)"

# Index on id for specific resource lookups
curl -X POST "http://evt-db:8093/query/service" \
  -u "$COUCHBASE_USERNAME:$COUCHBASE_PASSWORD" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "statement=CREATE INDEX idx_id ON \`$COUCHBASE_BUCKET\`(id)"

# Index on subjectPatientId for patient references
curl -X POST "http://evt-db:8093/query/service" \
  -u "$COUCHBASE_USERNAME:$COUCHBASE_PASSWORD" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "statement=CREATE INDEX idx_subjectPatientId ON \`$COUCHBASE_BUCKET\`(subjectPatientId)"

# Index on practitionerIds for practitioner references
curl -X POST "http://evt-db:8093/query/service" \
  -u "$COUCHBASE_USERNAME:$COUCHBASE_PASSWORD" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "statement=CREATE INDEX idx_practitionerIds ON \`$COUCHBASE_BUCKET\`(practitionerIds)"

# Composite index for resourceType + id for efficient lookups
curl -X POST "http://evt-db:8093/query/service" \
  -u "$COUCHBASE_USERNAME:$COUCHBASE_PASSWORD" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "statement=CREATE INDEX idx_resourceType_id ON \`$COUCHBASE_BUCKET\`(resourceType, id)"

echo "Waiting for indexes to be built..."
sleep 15

# Mark as initialized
mkdir -p /tmp/db-init-flag
touch /tmp/db-init-flag/initialized
echo "Couchbase setup complete with indexes and marked as initialized!"