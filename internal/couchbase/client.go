package couchbase

// Client represents a Couchbase client that orchestrates all operations
type Client struct {
	connManager *ConnectionManager
	docManager  *DocumentManager
	locker      *DatabaseLocker
}

// NewClient creates a new Couchbase client
func NewClient(url, username, password string) (*Client, error) {
	// Initialize connection manager
	connManager, err := NewConnectionManager(url, username, password)
	if err != nil {
		return nil, err
	}

	// Initialize database locker
	locker := NewDatabaseLocker(connManager.GetBucket())

	// Initialize document manager
	docManager := NewDocumentManager(connManager.GetBucket(), locker)

	client := &Client{
		connManager: connManager,
		docManager:  docManager,
		locker:      locker,
	}

	return client, nil
}

// Close closes the Couchbase connection
func (c *Client) Close() error {
	return c.connManager.Close()
}

// GetLocker returns the database locker
func (c *Client) GetLocker() *DatabaseLocker {
	return c.locker
}

// UpsertDocument stores or updates a document in Couchbase
func (c *Client) UpsertDocument(collection, docID string, data interface{}) error {
	return c.docManager.UpsertDocument(collection, docID, data)
}

// GetDocument retrieves a document from Couchbase
func (c *Client) GetDocument(collection, docID string, result interface{}) error {
	return c.docManager.GetDocument(collection, docID, result)
}

// DeleteDocument removes a document from Couchbase
func (c *Client) DeleteDocument(collection, docID string) error {
	return c.docManager.DeleteDocument(collection, docID)
}
