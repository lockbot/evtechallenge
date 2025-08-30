package couchbase

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// DatabaseLocker provides database locking functionality
type DatabaseLocker struct {
	bucket *gocb.Bucket
	locked bool
}

// NewDatabaseLocker creates a new database locker
func NewDatabaseLocker(bucket *gocb.Bucket) *DatabaseLocker {
	return &DatabaseLocker{
		bucket: bucket,
		locked: false,
	}
}

// Lock locks the database for exclusive access
func (l *DatabaseLocker) Lock() error {
	if l.locked {
		return fmt.Errorf("database is already locked")
	}

	// Store lock document
	lockDoc := map[string]interface{}{
		"locked":    true,
		"lockedAt":  time.Now().UTC(),
		"lockedBy":  "evtechallenge-ingest",
		"expiresAt": time.Now().UTC().Add(1 * time.Hour), // Lock expires in 1 hour
	}

	// Use default collection for lock document
	col := l.bucket.DefaultCollection()

	_, err := col.Upsert("db_lock", lockDoc, &gocb.UpsertOptions{})
	if err != nil {
		return fmt.Errorf("failed to create lock document: %w", err)
	}

	l.locked = true
	log.Info().Msg("Database locked successfully")
	return nil
}

// Unlock unlocks the database
func (l *DatabaseLocker) Unlock() error {
	if !l.locked {
		return fmt.Errorf("database is not locked")
	}

	// Use default collection for lock document
	col := l.bucket.DefaultCollection()

	_, err := col.Remove("db_lock", &gocb.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove lock document: %w", err)
	}

	l.locked = false
	log.Info().Msg("Database unlocked successfully")
	return nil
}

// IsLocked returns true if the database is currently locked
func (l *DatabaseLocker) IsLocked() bool {
	return l.locked
}

// CheckLockStatus checks the actual lock status in the database
func (l *DatabaseLocker) CheckLockStatus() (bool, error) {
	// Use default collection for lock document
	col := l.bucket.DefaultCollection()

	// Try to get lock document
	resultDoc, err := col.Get("db_lock", &gocb.GetOptions{})
	if err != nil {
		// Check if it's a document not found error
		if err == gocb.ErrDocumentNotFound {
			return false, nil // No lock document found
		}
		return false, fmt.Errorf("failed to check lock status: %w", err)
	}

	var lockDoc map[string]interface{}
	err = resultDoc.Content(&lockDoc)
	if err != nil {
		return false, fmt.Errorf("failed to parse lock document: %w", err)
	}

	// Check if lock has expired
	if expiresAt, ok := lockDoc["expiresAt"].(string); ok {
		expiresTime, err := time.Parse(time.RFC3339, expiresAt)
		if err == nil && time.Now().UTC().After(expiresTime) {
			// Lock has expired, remove it
			col.Remove("db_lock", &gocb.RemoveOptions{})
			l.locked = false
			return false, nil
		}
	}

	// Update local state
	l.locked = lockDoc["locked"].(bool)
	return l.locked, nil
}
