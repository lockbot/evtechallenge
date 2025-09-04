package dal

import (
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

// getEnv retrieves environment variable with fallback default
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// NewConnection creates a new Couchbase connection
func NewConnection() (*Connection, error) {
	cbURL := getEnv("COUCHBASE_URL", "couchbase://evtechallenge-db")
	user := getEnv("COUCHBASE_USERNAME", "evtechallenge_user")
	pass := getEnv("COUCHBASE_PASSWORD", "password")
	bucketName := getEnv("COUCHBASE_BUCKET", "evtechallenge")

	log.Info().
		Str("url", cbURL).
		Str("bucket", bucketName).
		Msg("Creating Couchbase connection")

	cluster, err := gocb.Connect(cbURL, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{Username: user, Password: pass},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Couchbase cluster")
		return nil, fmt.Errorf("connect cluster: %w", err)
	}

	bucket := cluster.Bucket(bucketName)
	err = bucket.WaitUntilReady(30*time.Second, &gocb.WaitUntilReadyOptions{
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery},
	})
	if err != nil {
		log.Error().Err(err).Msg("Couchbase bucket not ready")
		return nil, fmt.Errorf("bucket not ready: %w", err)
	}

	log.Info().Msg("Couchbase connection created successfully")
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
