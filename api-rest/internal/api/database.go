package api

import (
	"fmt"
	"os"
	"time"

	"github.com/couchbase/gocb/v2"
)

var (
	cbCluster    *gocb.Cluster
	cbBucket     *gocb.Bucket
	cbBucketName string
)

// getEnv retrieves environment variable with fallback default
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// InitCouchbase initializes the Couchbase connection
func InitCouchbase() error {
	cbURL := getEnv("COUCHBASE_URL", "couchbase://evtechallenge-db")
	user := getEnv("COUCHBASE_USERNAME", "evtechallenge_user")
	pass := getEnv("COUCHBASE_PASSWORD", "password")
	cbBucketName = getEnv("COUCHBASE_BUCKET", "evtechallenge")

	cluster, err := gocb.Connect(cbURL, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{Username: user, Password: pass},
	})
	if err != nil {
		return fmt.Errorf("connect cluster: %w", err)
	}
	bucket := cluster.Bucket(cbBucketName)
	if err := bucket.WaitUntilReady(30*time.Second, &gocb.WaitUntilReadyOptions{ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery}}); err != nil {
		return fmt.Errorf("bucket not ready: %w", err)
	}
	cbCluster = cluster
	cbBucket = bucket
	return nil
}

// GetCluster returns the Couchbase cluster instance
func GetCluster() *gocb.Cluster {
	return cbCluster
}

// GetBucket returns the Couchbase bucket instance
func GetBucket() *gocb.Bucket {
	return cbBucket
}

// GetBucketName returns the Couchbase bucket name
func GetBucketName() string {
	return cbBucketName
}
