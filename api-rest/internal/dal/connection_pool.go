package dal

import (
	"sync"
)

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
		// Pool is full, discard connection
	}
}

// GetConnectionWithRetry gets a connection with retry logic
func GetConnectionWithRetry() (*Connection, error) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		conn, err := GetConnOrGenConn()
		if err == nil {
			return conn, nil
		}
		if i == maxRetries-1 {
			return nil, err
		}
	}
	return nil, nil // This should never be reached
}
