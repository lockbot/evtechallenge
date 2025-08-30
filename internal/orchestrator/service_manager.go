package orchestrator

import (
	"context"
	"os/exec"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// ServiceManager manages the lifecycle of ingest and API services
type ServiceManager struct {
	ingestCmd *exec.Cmd
	apiCmd    *exec.Cmd
}

// NewServiceManager creates a new service manager
func NewServiceManager() *ServiceManager {
	return &ServiceManager{}
}

// StartIngestService starts the ingest service
func (sm *ServiceManager) StartIngestService(ctx context.Context, binExt string) error {
	log.Info().Msg("Starting ingest service...")

	sm.ingestCmd = exec.CommandContext(ctx, "./ingest"+binExt)
	sm.ingestCmd.Stdout = log.Logger
	sm.ingestCmd.Stderr = log.Logger

	if err := sm.ingestCmd.Start(); err != nil {
		return err
	}

	// Wait a bit for ingest to start
	time.Sleep(2 * time.Second)
	return nil
}

// StartAPIService starts the API service
func (sm *ServiceManager) StartAPIService(ctx context.Context, binExt string) error {
	log.Info().Msg("Starting API service...")

	sm.apiCmd = exec.CommandContext(ctx, "./api"+binExt)
	sm.apiCmd.Stdout = log.Logger
	sm.apiCmd.Stderr = log.Logger

	if err := sm.apiCmd.Start(); err != nil {
		return err
	}

	return nil
}

// WaitForServices waits for both services to complete or context to be cancelled
func (sm *ServiceManager) WaitForServices(ctx context.Context) {
	log.Info().Msg("Both services started, waiting for completion...")

	// Wait for ingest to complete
	ingestDone := make(chan error, 1)
	go func() {
		ingestDone <- sm.ingestCmd.Wait()
	}()

	// Wait for API to complete
	apiDone := make(chan error, 1)
	go func() {
		apiDone <- sm.apiCmd.Wait()
	}()

	// Wait for either service to complete or context to be cancelled
	select {
	case err := <-ingestDone:
		if err != nil {
			log.Error().Err(err).Msg("Ingest service exited with error")
		} else {
			log.Info().Msg("Ingest service completed successfully")
		}
	case err := <-apiDone:
		if err != nil {
			log.Error().Err(err).Msg("API service exited with error")
		} else {
			log.Info().Msg("API service exited")
		}
	case <-ctx.Done():
		log.Info().Msg("Shutting down services...")
		sm.shutdownServices()
	}
}

// shutdownServices gracefully shuts down all services
func (sm *ServiceManager) shutdownServices() {
	// Terminate ingest service
	if sm.ingestCmd.Process != nil {
		sm.ingestCmd.Process.Signal(syscall.SIGTERM)
	}

	// Terminate API service
	if sm.apiCmd.Process != nil {
		sm.apiCmd.Process.Signal(syscall.SIGTERM)
	}

	// Wait for graceful shutdown
	time.Sleep(5 * time.Second)

	// Force kill if still running
	if sm.ingestCmd.Process != nil {
		sm.ingestCmd.Process.Kill()
	}
	if sm.apiCmd.Process != nil {
		sm.apiCmd.Process.Kill()
	}
}
