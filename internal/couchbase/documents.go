package couchbase

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/gocb/v2"
)

// DocumentManager handles document CRUD operations
type DocumentManager struct {
	bucket *gocb.Bucket
	locker *DatabaseLocker
}

// NewDocumentManager creates a new document manager
func NewDocumentManager(bucket *gocb.Bucket, locker *DatabaseLocker) *DocumentManager {
	return &DocumentManager{
		bucket: bucket,
		locker: locker,
	}
}

// UpsertDocument stores or updates a document in Couchbase
func (dm *DocumentManager) UpsertDocument(collection, docID string, data interface{}) error {
	// Check if database is locked
	if dm.locker.IsLocked() {
		return fmt.Errorf("database is locked, cannot write document")
	}

	// Get collection - use default scope and collection for simplicity
	// In production, you might want to use specific scopes/collections
	col := dm.bucket.DefaultCollection()

	// Convert data to JSON if it's not already
	var jsonData interface{}
	switch v := data.(type) {
	case json.RawMessage:
		jsonData = v
	case []byte:
		jsonData = v
	default:
		jsonData = data
	}

	// Upsert document
	_, err := col.Upsert(docID, jsonData, &gocb.UpsertOptions{})
	if err != nil {
		return fmt.Errorf("failed to upsert document %s: %w", docID, err)
	}

	return nil
}

// GetDocument retrieves a document from Couchbase
func (dm *DocumentManager) GetDocument(collection, docID string, result interface{}) error {
	// Check if database is locked
	if dm.locker.IsLocked() {
		return fmt.Errorf("database is locked, cannot read document")
	}

	// Get collection - use default scope and collection for simplicity
	col := dm.bucket.DefaultCollection()

	// Get document
	resultDoc, err := col.Get(docID, &gocb.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get document %s: %w", docID, err)
	}

	// Parse result
	err = resultDoc.Content(result)
	if err != nil {
		return fmt.Errorf("failed to parse document content: %w", err)
	}

	return nil
}

// DeleteDocument removes a document from Couchbase
func (dm *DocumentManager) DeleteDocument(collection, docID string) error {
	// Check if database is locked
	if dm.locker.IsLocked() {
		return fmt.Errorf("database is locked, cannot delete document")
	}

	// Get collection - use default scope and collection for simplicity
	col := dm.bucket.DefaultCollection()

	// Delete document
	_, err := col.Remove(docID, &gocb.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete document %s: %w", docID, err)
	}

	return nil
}
