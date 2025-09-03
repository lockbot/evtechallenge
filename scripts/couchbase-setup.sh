#!/bin/sh

set -e

echo "Starting Couchbase setup..."

# Wait for Couchbase to be fully ready
echo "Waiting for Couchbase UI..."
attempts=0; max=13
until curl -sf http://evtechallenge-db:8091 > /dev/null; do
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
until curl -sf http://evtechallenge-db:8091/pools > /dev/null; do
  attempts=$((attempts+1))
  if [ $attempts -ge $max ]; then echo "ERROR: Couchbase UI not ready"; exit 1; fi
  echo "Waiting for Couchbase... ($attempts/$max)"
  sleep 3
done
echo "Couchbase is ready!"

# Resolve DB container IP for advertised hostname
DB_IP=$(getent hosts evtechallenge-db | head -n1 | cut -d" " -f1 || true)
if [ -z "$DB_IP" ]; then DB_IP=$(getent ahostsv4 evtechallenge-db | head -n1 | cut -d" " -f1 || true); fi
if [ -z "$DB_IP" ]; then echo "ERROR: Unable to resolve DB IP"; exit 1; fi
echo "Using node IP: $DB_IP"

echo "Running node-init..."
/opt/couchbase/bin/couchbase-cli node-init -c evtechallenge-db --node-init-hostname "$DB_IP" -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD"

echo "Initializing cluster (data,index,query)..."
/opt/couchbase/bin/couchbase-cli cluster-init -c evtechallenge-db \
  --cluster-username "$COUCHBASE_ADMINISTRATOR_USERNAME" --cluster-password "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --services data,index,query --cluster-ramsize 512 \
  --cluster-index-ramsize 256 --index-storage-setting default

echo "Creating bucket..."
/opt/couchbase/bin/couchbase-cli bucket-create -c evtechallenge-db -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --bucket "$COUCHBASE_BUCKET" --bucket-type couchbase --bucket-ramsize 256 --wait

echo "Creating application user..."
/opt/couchbase/bin/couchbase-cli user-manage -c evtechallenge-db -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --set --rbac-username "$COUCHBASE_USERNAME" --rbac-password "$COUCHBASE_PASSWORD" \
  --roles bucket_full_access["$COUCHBASE_BUCKET"] --auth-domain local

echo "Granting admin role to application user..."
# This gives the application user full permissions to create scopes and collections
/opt/couchbase/bin/couchbase-cli user-manage -c evtechallenge-db -u "$COUCHBASE_ADMINISTRATOR_USERNAME" -p "$COUCHBASE_ADMINISTRATOR_PASSWORD" \
  --set --rbac-username "$COUCHBASE_USERNAME" --rbac-password "$COUCHBASE_PASSWORD" \
  --roles "admin" --auth-domain local

# Mark as initialized
mkdir -p /tmp/db-init-flag
touch /tmp/db-init-flag/initialized
echo "Couchbase setup complete and marked as initialized!"
