package dal

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// Connection represents the Couchbase connection
type Connection struct {
	cluster    *gocb.Cluster
	bucket     *gocb.Bucket
	bucketName string
}

// NewConnection creates a new Couchbase connection
func NewConnection() (*Connection, error) {
	var err error

	// Get configuration from environment
	couchbaseURL := getEnvOrDefault("COUCHBASE_URL", "couchbase://evtechallenge-db")
	username := getEnvOrDefault("COUCHBASE_USERNAME", "evtechallenge_user")
	password := getEnvOrDefault("COUCHBASE_PASSWORD", "password")
	bucketName := getEnvOrDefault("COUCHBASE_BUCKET", "evtechallenge")

	cluster, err := gocb.Connect(couchbaseURL, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
		TimeoutsConfig: gocb.TimeoutsConfig{
			ConnectTimeout:    60 * time.Second,
			KVTimeout:         5 * time.Second,
			QueryTimeout:      30 * time.Second,
			ManagementTimeout: 30 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Couchbase: %w", err)
	}

	bucket := cluster.Bucket(bucketName)

	// Wait for the bucket to be ready for KV operations only
	err = bucket.WaitUntilReady(90*time.Second, &gocb.WaitUntilReadyOptions{
		Context:      context.Background(),
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue},
	})
	if err != nil {
		return nil, fmt.Errorf("couchbase bucket not ready: %w", err)
	}

	log.Info().
		Str("couchbase_url", couchbaseURL).
		Str("bucket", bucketName).
		Msg("Couchbase connection initialized successfully")

	return &Connection{
		cluster:    cluster,
		bucket:     bucket,
		bucketName: bucketName,
	}, nil
}

// Close closes the Couchbase connection
func (c *Connection) Close() error {
	if c.cluster != nil {
		return c.cluster.Close(nil)
	}
	return nil
}

// GetBucket returns the Couchbase bucket
func (c *Connection) GetBucket() *gocb.Bucket {
	return c.bucket
}

// GetCluster returns the Couchbase cluster
func (c *Connection) GetCluster() *gocb.Cluster {
	return c.cluster
}

// GetBucketName returns the Couchbase bucket name
func (c *Connection) GetBucketName() string {
	return c.bucketName
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
