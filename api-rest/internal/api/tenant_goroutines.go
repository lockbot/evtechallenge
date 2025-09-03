package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// TenantData contains all tenant-related information
type TenantData struct {
	Status     *TenantStatus
	Channel    chan struct{}
	Cancel     context.CancelFunc
	Collection *gocb.Collection
}

// TenantStatus represents a tenant's readiness status
type TenantStatus struct {
	TenantID    string    `json:"tenantId"`
	Ready       bool      `json:"ready"`
	WarmedAt    time.Time `json:"warmedAt"`
	LastRequest time.Time `json:"lastRequest"`
	Message     string    `json:"message"`
}

const (
	// System document keys
	TenantStatusKey = "_system/tenant_status"

	// Timeout settings
	TenantWarmupTimeout = 30 * time.Minute
)

// TenantGoroutineManager manages tenant goroutines with warm-up/cool-down cycles
type TenantGoroutineManager struct {
	mu sync.RWMutex
	// One map for all tenant data: tenantID -> TenantData
	tenants map[string]*TenantData
	// Store bucket reference for collection operations
	bucket *gocb.Bucket
}

// NewTenantGoroutineManager creates a new tenant goroutine manager
func NewTenantGoroutineManager(bucket *gocb.Bucket) *TenantGoroutineManager {
	return &TenantGoroutineManager{
		tenants: make(map[string]*TenantData),
		bucket:  bucket,
	}
}

// WarmUpTenant starts a goroutine for a tenant and initializes their collection
func (tgm *TenantGoroutineManager) WarmUpTenant(tenantID string) error {
	tgm.mu.Lock()

	// Check if tenant is already warm
	if tenantData, exists := tgm.tenants[tenantID]; exists && tenantData.Status.Ready {
		log.Info().Str("tenant", tenantID).Msg("Tenant already warm, updating last request time")
		tenantData.Status.LastRequest = time.Now().UTC()
		tgm.mu.Unlock()
		return nil
	}

	// Create or get existing channel
	var ch chan struct{}
	if tenantData, exists := tgm.tenants[tenantID]; exists {
		ch = tenantData.Channel
	} else {
		ch = make(chan struct{})
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize tenant data
	tenantData := &TenantData{
		Status: &TenantStatus{
			TenantID:    tenantID,
			Ready:       false,
			WarmedAt:    time.Now().UTC(),
			LastRequest: time.Now().UTC(),
			Message:     "Tenant warming up",
		},
		Channel: ch,
		Cancel:  cancel,
	}
	tgm.tenants[tenantID] = tenantData

	// Start tenant goroutine
	go tgm.tenantGoroutine(ctx, tenantID, ch)

	log.Info().Str("tenant", tenantID).Msg("Tenant goroutine started")

	// Release lock before waiting
	tgm.mu.Unlock()

	// Wait for collection to be ready (with increased timeout)
	ready := make(chan bool, 1)
	go func() {
		for i := 0; i < 60; i++ { // Wait up to 60 seconds (doubled)
			time.Sleep(1 * time.Second)
			if tgm.IsTenantWarm(tenantID) {
				ready <- true
				return
			}
		}
		ready <- false
	}()

	select {
	case isReady := <-ready:
		if isReady {
			log.Info().Str("tenant", tenantID).Msg("Tenant collection ready")
			return nil
		} else {
			return fmt.Errorf("tenant collection initialization timeout")
		}
	case <-time.After(60 * time.Second): // Increased to 60 seconds
		return fmt.Errorf("tenant collection initialization timeout")
	}
}

// tenantGoroutine manages a single tenant's lifecycle
func (tgm *TenantGoroutineManager) tenantGoroutine(ctx context.Context, tenantID string, ch chan struct{}) {
	defer func() {
		log.Info().Str("tenant", tenantID).Msg("Tenant goroutine stopped")
	}()

	// Initialize tenant collection
	err := tgm.initializeTenantCollection(tenantID)
	if err != nil {
		log.Error().Str("tenant", tenantID).Err(err).Msg("Failed to initialize tenant collection")
		return
	}

	// Mark tenant as ready
	tgm.mu.Lock()
	if tenantData, exists := tgm.tenants[tenantID]; exists {
		tenantData.Status.Ready = true
		tenantData.Status.Message = "Tenant ready"
	}
	tgm.mu.Unlock()

	log.Info().Str("tenant", tenantID).Msg("Tenant collection initialized and ready")

	// Main loop with timeout
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, exit gracefully
			return
		case <-ch:
			// Activity detected, reset timeout
			tgm.mu.Lock()
			if tenantData, exists := tgm.tenants[tenantID]; exists {
				tenantData.Status.LastRequest = time.Now().UTC()
			}
			tgm.mu.Unlock()
		case <-ticker.C:
			// Check if tenant should go cold
			if tgm.shouldGoCold(tenantID) {
				tgm.goCold(tenantID)
				return
			}
		}
	}
}

// initializeTenantCollection sets up the tenant's collection by copying from DefaultCollection
func (tgm *TenantGoroutineManager) initializeTenantCollection(tenantID string) error {
	log.Info().Str("tenant", tenantID).Msg("Initializing tenant collection...")

	// Use consistent collection name for FHIR data
	tenantCollectionName := "reviewable-fhir"

	// Get the default collection to copy from
	defaultCollection := tgm.bucket.DefaultCollection()

	// Check if tenant collection already exists, if not create it
	tenantCollection, err := tgm.ensureTenantCollectionExists(tenantID, tenantCollectionName)
	if err != nil {
		log.Error().Str("tenant", tenantID).Err(err).Msg("Failed to ensure tenant collection exists")
		// Fallback to default collection
		tenantCollection = defaultCollection
		log.Info().Str("tenant", tenantID).Msg("Falling back to default collection")
	}

	// Store the collection reference in the tenant data
	tgm.mu.Lock()
	if tenantData, exists := tgm.tenants[tenantID]; exists {
		tenantData.Collection = tenantCollection
	}
	tgm.mu.Unlock()

	log.Info().Str("tenant", tenantID).Msg("Tenant collection initialized successfully")
	return nil
}

// shouldGoCold checks if a tenant should go cold due to inactivity
func (tgm *TenantGoroutineManager) shouldGoCold(tenantID string) bool {
	tgm.mu.RLock()
	defer tgm.mu.RUnlock()

	tenantData, exists := tgm.tenants[tenantID]
	if !exists {
		return true
	}

	// Check if 30 minutes have passed since last request
	return time.Since(tenantData.Status.LastRequest) > TenantWarmupTimeout
}

// goCold gracefully stops a tenant's goroutine
func (tgm *TenantGoroutineManager) goCold(tenantID string) {
	tgm.mu.Lock()
	defer tgm.mu.Unlock()

	log.Info().Str("tenant", tenantID).Msg("Tenant going cold due to inactivity")

	// Cancel the goroutine
	if tenantData, exists := tgm.tenants[tenantID]; exists {
		if tenantData.Cancel != nil {
			tenantData.Cancel()
		}
		// Mark tenant as not ready
		tenantData.Status.Ready = false
		tenantData.Status.Message = "Tenant cold - needs warm-up"
	}
}

// IsTenantWarm checks if a tenant is currently warm and ready
func (tgm *TenantGoroutineManager) IsTenantWarm(tenantID string) bool {
	tgm.mu.RLock()
	defer tgm.mu.RUnlock()

	tenantData, exists := tgm.tenants[tenantID]
	return exists && tenantData.Status.Ready
}

// RecordActivity records activity for a tenant to keep it warm
func (tgm *TenantGoroutineManager) RecordActivity(tenantID string) {
	tgm.mu.RLock()
	tenantData, exists := tgm.tenants[tenantID]
	tgm.mu.RUnlock()

	if exists && tenantData.Channel != nil {
		// Non-blocking send to avoid blocking the request
		select {
		case tenantData.Channel <- struct{}{}:
			// Activity recorded
		default:
			// Channel buffer full, ignore
		}
	}
}

// GetTenantStatus returns the current status of a tenant
func (tgm *TenantGoroutineManager) GetTenantStatus(tenantID string) *TenantStatus {
	tgm.mu.RLock()
	defer tgm.mu.RUnlock()

	if tenantData, exists := tgm.tenants[tenantID]; exists {
		// Return a copy to avoid race conditions
		return &TenantStatus{
			TenantID:    tenantData.Status.TenantID,
			Ready:       tenantData.Status.Ready,
			WarmedAt:    tenantData.Status.WarmedAt,
			LastRequest: tenantData.Status.LastRequest,
			Message:     tenantData.Status.Message,
		}
	}
	return nil
}

// GetTenantCollection returns the collection for a specific tenant from stored status
func (tgm *TenantGoroutineManager) GetTenantCollection(tenantID string) (*gocb.Collection, error) {
	tgm.mu.RLock()
	defer tgm.mu.RUnlock()

	if tenantData, exists := tgm.tenants[tenantID]; exists && tenantData.Status.Ready && tenantData.Collection != nil {
		return tenantData.Collection, nil
	}
	return nil, fmt.Errorf("tenant %s not ready or collection not initialized", tenantID)
}

// copyCollectionData copies all documents from source to destination collection
func (tgm *TenantGoroutineManager) copyCollectionData(source, dest *gocb.Collection, tenantID string) error {
	log.Info().Str("tenant", tenantID).Msg("Starting collection data copy...")

	// Get cluster for N1QL queries
	cluster := GetCluster()
	if cluster == nil {
		return fmt.Errorf("cluster not initialized")
	}

	// Query all documents from source collection (default scope)
	query := "SELECT META(d).id AS id, d AS resource FROM `" + GetBucketName() + "`.`_default`.`_default` AS d"
	rows, err := cluster.Query(query, nil)
	if err != nil {
		return fmt.Errorf("failed to query source collection: %w", err)
	}
	defer rows.Close()

	// Copy each document to destination collection
	copiedCount := 0
	for rows.Next() {
		var row struct {
			ID       string                 `json:"id"`
			Resource map[string]interface{} `json:"resource"`
		}

		err := rows.Row(&row)
		if err != nil {
			log.Error().Str("tenant", tenantID).Err(err).Msg("Failed to read row, skipping")
			continue
		}

		// Add tenant-specific fields
		doc := row.Resource
		doc["reviewed"] = false
		doc["reviewTime"] = ""
		doc["tenantId"] = tenantID

		// Insert into destination collection
		_, err = dest.Insert(row.ID, doc, nil)
		if err != nil {
			log.Error().Str("tenant", tenantID).Err(err).Msg("Failed to copy document, skipping")
			continue
		}

		copiedCount++
	}

	log.Info().Str("tenant", tenantID).Int("copiedCount", copiedCount).Msg("Collection data copy completed")
	return nil
}

// ensureTenantCollectionExists checks if a tenant collection exists and creates it if needed
func (tgm *TenantGoroutineManager) ensureTenantCollectionExists(tenantID, collectionName string) (*gocb.Collection, error) {
	log.Info().Str("tenant", tenantID).Str("collection", collectionName).Msg("Checking if tenant collection exists...")

	// First check if scope already exists
	collections, err := tgm.bucket.Collections().GetAllScopes(&gocb.GetAllScopesOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes: %w", err)
	}

	// Look for existing tenant scope
	for _, scope := range collections {
		if scope.Name == tenantID {
			// Scope exists, return the collection within it
			log.Info().Str("tenant", tenantID).Msg("Tenant scope already exists")
			return tgm.bucket.Scope(tenantID).Collection(collectionName), nil
		}
	}

	// Scope doesn't exist, create it via HTTP management API
	log.Info().Str("tenant", tenantID).Msg("Creating tenant scope...")

	// Get cluster for management operations
	cluster := GetCluster()
	if cluster == nil {
		return nil, fmt.Errorf("cluster not initialized")
	}

	// Create scope via HTTP management API
	// Note: This requires admin privileges
	scopeName := tenantID
	bucketName := GetBucketName()

	// Debug logging
	log.Info().Str("tenant", tenantID).Str("scopeName", scopeName).Str("bucketName", bucketName).Msg("Attempting to create scope")

	// Create scope
	err = createScopeViaHTTP(bucketName, scopeName)
	if err != nil {
		log.Error().Err(err).Str("tenant", tenantID).Msg("Failed to create scope, falling back to default collection")
		return tgm.bucket.DefaultCollection(), nil
	}

	// Create collection within the scope
	err = createCollectionViaHTTP(bucketName, scopeName, collectionName)
	if err != nil {
		log.Error().Err(err).Str("tenant", tenantID).Msg("Failed to create collection, falling back to default collection")
		return tgm.bucket.DefaultCollection(), nil
	}

	log.Info().Str("tenant", tenantID).Str("scope", scopeName).Str("collection", collectionName).Msg("Tenant scope and collection created successfully")

	// Return the newly created collection
	return tgm.bucket.Scope(scopeName).Collection(collectionName), nil
}

// Global instance and accessor
var globalTenantManager *TenantGoroutineManager
var globalTenantManagerOnce sync.Once

// GetTenantGoroutineManager returns the global tenant goroutine manager instance
func GetTenantGoroutineManager() *TenantGoroutineManager {
	globalTenantManagerOnce.Do(func() {
		// Get bucket from the global bucket instance
		bucket := GetBucket()
		if bucket == nil {
			log.Error().Msg("Bucket not initialized, cannot create tenant manager")
			return
		}
		globalTenantManager = NewTenantGoroutineManager(bucket)
	})
	return globalTenantManager
}

// createScopeViaHTTP creates a scope using Couchbase's HTTP management API
func createScopeViaHTTP(bucketName, scopeName string) error {
	// Get management endpoint from environment or use default
	mgmtHost := os.Getenv("COUCHBASE_MANAGEMENT_HOST")
	if mgmtHost == "" {
		mgmtHost = "localhost:8091" // Default management port
	}

	username := os.Getenv("COUCHBASE_USERNAME")
	password := os.Getenv("COUCHBASE_PASSWORD")

	url := fmt.Sprintf("http://%s/pools/default/buckets/%s/scopes", mgmtHost, bucketName)

	payload := map[string]string{
		"name": scopeName,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal scope creation payload: %w", err)
	}

	// Debug logging
	log.Info().Str("url", url).Str("payload", string(jsonData)).Msg("Creating scope with payload")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create scope creation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute scope creation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scope creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// createCollectionViaHTTP creates a collection within a scope using Couchbase's HTTP management API
func createCollectionViaHTTP(bucketName, scopeName, collectionName string) error {
	// Get management endpoint from environment or use default
	mgmtHost := os.Getenv("COUCHBASE_MANAGEMENT_HOST")
	if mgmtHost == "" {
		mgmtHost = "localhost:8091" // Default management port
	}

	username := os.Getenv("COUCHBASE_USERNAME")
	password := os.Getenv("COUCHBASE_PASSWORD")

	url := fmt.Sprintf("http://%s/pools/default/buckets/%s/scopes/%s/collections", mgmtHost, bucketName, scopeName)

	payload := map[string]interface{}{
		"name":   collectionName,
		"maxTTL": 0, // No TTL
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal collection creation payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create collection creation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute collection creation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("collection creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
