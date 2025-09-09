package dal

import (
	"context"
	"fmt"
	"os"
	"sync"
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

// ConnectionPool manages a pool of Couchbase connections
type ConnectionPool struct {
	connections chan *Connection
	maxSize     int
}

var (
	pool     *ConnectionPool
	poolOnce sync.Once
)

// GetConnOrGenConn gets a connection from the pool or creates a new one
func GetConnOrGenConn() (*Connection, error) {
	poolOnce.Do(func() {
		pool = &ConnectionPool{
			connections: make(chan *Connection, 5), // Pool of 5 connections
			maxSize:     5,
		}
	})

	// Try to get connection from pool
	select {
	case conn := <-pool.connections:
		// Test if connection is still alive
		if isConnectionAlive(conn) {
			return conn, nil
		}
		// Connection is dead, create a new one
		return createNewConnection()
	default:
		// Pool is empty, create new connection
		return createNewConnection()
	}
}

// ReturnConnection returns a connection to the pool
func ReturnConnection(conn *Connection) {
	if conn == nil {
		return
	}

	// Test if connection is still alive
	if !isConnectionAlive(conn) {
		// Connection is dead, don't return it to pool
		return
	}

	// Try to return to pool
	select {
	case pool.connections <- conn:
		// Successfully returned to pool
	default:
		// Pool is full, close the connection
		conn.Close()
	}
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

// CloseAllConnections closes all connections in the pool (for graceful shutdown)
func CloseAllConnections() {
	if pool == nil {
		return
	}

	log.Info().Msg("Closing all connections in pool...")

	// Close all connections in the pool
	for {
		select {
		case conn := <-pool.connections:
			if conn != nil {
				conn.Close()
			}
		default:
			// Pool is empty
			log.Info().Msg("All connections closed")
			return
		}
	}
}

// getConnOrGenConn creates a new Couchbase connection
func getConnOrGenConn() (*Connection, error) {
	var err error

	// Get configuration from environment
	couchbaseURL := getEnvOrDefault("COUCHBASE_URL", "couchbase://evt-db")
	username := getEnvOrDefault("COUCHBASE_USERNAME", "evtechallenge_user")
	password := getEnvOrDefault("COUCHBASE_PASSWORD", "password")
	bucketName := getEnvOrDefault("COUCHBASE_BUCKET", "EvTeChallenge")

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

	// Wait for the bucket to be ready for KV and Query operations
	err = bucket.WaitUntilReady(15*time.Second, &gocb.WaitUntilReadyOptions{
		Context:      context.Background(),
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery},
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
