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

// getEnv retrieves environment variable with fallback default
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// isConnectionAlive tests if a connection is still usable
func isConnectionAlive(conn *Connection) bool {
	if conn == nil || conn.cluster == nil {
		return false
	}

	// Try a simple operation to test the connection
	_, err := conn.cluster.Ping(&gocb.PingOptions{})
	return err == nil
}

// createNewConnection creates a fresh Couchbase connection
func createNewConnection() (*Connection, error) {
	return getConnOrGenConn()
}

// IsClusterClosedError checks if an error indicates the cluster is closed
func IsClusterClosedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check if error ends with "cluster closed" (last 14 characters)
	if len(errStr) >= 14 && errStr[len(errStr)-14:] == "cluster closed" {
		return true
	}
	return false
}

// getConnOrGenConn creates a new Couchbase connection
func getConnOrGenConn() (*Connection, error) {
	cbURL := getEnv("COUCHBASE_URL", "couchbase://evt-db")
	user := getEnv("COUCHBASE_USERNAME", "evtechallenge_user")
	pass := getEnv("COUCHBASE_PASSWORD", "password")
	bucketName := getEnv("COUCHBASE_BUCKET", "EvTeChallenge")

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
	err = bucket.WaitUntilReady(15*time.Second, &gocb.WaitUntilReadyOptions{
		Context:      context.Background(),
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery},
	})
	if err != nil {
		log.Error().Err(err).Msg("e ")
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

// GetScope returns a specific scope from the bucket
func (c *Connection) GetScope(scopeName string) *gocb.Scope {
	return c.bucket.Scope(scopeName)
}

// GetDefaultScope returns the default scope
func (c *Connection) GetDefaultScope() *gocb.Scope {
	return c.bucket.Scope("_default")
}

// GetCollection returns a specific collection from a scope
func (c *Connection) GetCollection(scopeName, collectionName string) *gocb.Collection {
	return c.bucket.Scope(scopeName).Collection(collectionName)
}

// GetDefaultCollection returns the default collection from the default scope
func (c *Connection) GetDefaultCollection() *gocb.Collection {
	return c.bucket.DefaultCollection()
}
