package couchbase

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

// ConnectionManager handles Couchbase cluster and bucket connections
type ConnectionManager struct {
	cluster *gocb.Cluster
	bucket  *gocb.Bucket
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(url, username, password string) (*ConnectionManager, error) {
	// Ensure proper connection string format
	connectionString := url
	if len(url) > 7 && url[:7] == "http://" {
		// Convert http:// to couchbases:// for local development
		// In production, you should use couchbases:// directly
		connectionString = "couchbases://" + url[7:]
	} else if len(url) > 8 && url[:8] != "couchbases://" {
		// If no protocol specified, assume couchbases://
		connectionString = "couchbases://" + url
	}

	// Connect to cluster
	cluster, err := gocb.Connect(connectionString, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Wait for cluster to be ready
	err = cluster.WaitUntilReady(30*time.Second, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for cluster: %w", err)
	}

	// Open bucket (assume it exists - don't try to create it)
	bucket := cluster.Bucket("evtechallenge")

	// Wait for bucket to be ready
	err = bucket.WaitUntilReady(10*time.Second, nil)
	if err != nil {
		return nil, fmt.Errorf("bucket 'evtechallenge' is not accessible: %w", err)
	}

	return &ConnectionManager{
		cluster: cluster,
		bucket:  bucket,
	}, nil
}

// Close closes the Couchbase connection
func (cm *ConnectionManager) Close() error {
	return cm.cluster.Close(nil)
}

// GetBucket returns the bucket instance
func (cm *ConnectionManager) GetBucket() *gocb.Bucket {
	return cm.bucket
}

// GetCluster returns the cluster instance
func (cm *ConnectionManager) GetCluster() *gocb.Cluster {
	return cm.cluster
}
