package orchestrator

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

// SignalHandler manages OS signals for graceful shutdown
type SignalHandler struct {
	sigChan chan os.Signal
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler() *SignalHandler {
	sh := &SignalHandler{
		sigChan: make(chan os.Signal, 1),
	}

	// Register for interrupt and terminate signals
	signal.Notify(sh.sigChan, syscall.SIGINT, syscall.SIGTERM)

	return sh
}

// HandleSignals waits for shutdown signals and cancels the context
func (sh *SignalHandler) HandleSignals(ctx context.Context, cancel context.CancelFunc) {
	go func() {
		sig := <-sh.sigChan
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()
}
