package dal

import (
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

// getEnv retrieves environment variable with fallback default
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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

// GetConnectionWithRetry gets a connection and retries if cluster is closed
func GetConnectionWithRetry() (*Connection, error) {
	conn, err := GetConnOrGenConn()
	if err != nil {
		return nil, err
	}

	// Test the connection
	if !isConnectionAlive(conn) {
		// Connection is dead, try to get a fresh one
		return createNewConnection()
	}

	return conn, nil
}

// getConnOrGenConn creates a new Couchbase connection
func getConnOrGenConn() (*Connection, error) {
	cbURL := getEnv("COUCHBASE_URL", "couchbase://evt-db")
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
