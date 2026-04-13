package cli

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// ObservableCommand is an interface for commands that want custom observability behavior.
// Commands implementing this interface can register themselves for opt-in observability.
//
// Boilerplate for new commands (≤5 lines):
//
//   // 1. Implement the interface
//   type myCmd struct{}
//
//   func (c *myCmd) ObservableName() string { return "myCommand" }
//   func (c *myCmd) Observe(ctx context.Context, cmd *cobra.Command, runErr error, dur time.Duration) { /* custom logic */ }
//
//   // 2. Register in init()
//   func init() {
//       cmd := &cobra.Command{...}
//       RegisterObservable(cmd, &myCmd{})
//   }
type ObservableCommand interface {
	// ObservableName returns the name of the observable for logging.
	ObservableName() string
	// Observe is called after command execution with context, the command, any error, and duration.
	Observe(ctx context.Context, cmd *cobra.Command, runErr error, dur time.Duration)
}

var (
	// observablesMu protects the observables map
	observablesMu sync.Mutex
	// observables maps command paths to their ObservableCommand implementations
	observables = make(map[string]ObservableCommand)
	// observeEnabled indicates if observability is enabled via flag or env var
	observeEnabled bool
)

// RegisterObservable registers a command with its ObservableCommand implementation.
// This enables custom observability behavior for the command.
//
// The cmd parameter is the cobra.Command to register.
// The obs parameter is the ObservableCommand implementation.
func RegisterObservable(cmd *cobra.Command, obs ObservableCommand) {
	observablesMu.Lock()
	defer observablesMu.Unlock()
	observables[cmd.CommandPath()] = obs
}

// isObservabilityEnabled returns true if observability is enabled via --observe flag
// or MAPJ_OBSERVE=1 environment variable.
func isObservabilityEnabled() bool {
	// Check env var first (can be cached at package init)
	if os.Getenv("MAPJ_OBSERVE") == "1" {
		return true
	}
	return observeEnabled
}

// observeCommand finds and calls the Observe method for the given command if registered.
func observeCommand(ctx context.Context, cmd *cobra.Command, runErr error, dur time.Duration) {
	observablesMu.Lock()
	obs, exists := observables[cmd.CommandPath()]
	observablesMu.Unlock()

	if !exists {
		return
	}

	// Recover from any panic in Observe() to never crash the command
	defer func() {
		if r := recover(); r != nil {
			zap.L().With(zap.String("observable", obs.ObservableName())).
				Error("Observe() panicked",
					zap.Any("panic", r),
					zap.String("command", cmd.CommandPath()),
				)
		}
	}()

	obs.Observe(ctx, cmd, runErr, dur)
}

// Metrics holds basic command metrics.
type Metrics struct {
	CommandCount int64
	TotalLatency int64 // milliseconds
}

// metricsMu protects metrics
var metricsMu sync.Mutex
var metrics = Metrics{}

// IncCommandCount increments the command counter.
func IncCommandCount() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	metrics.CommandCount++
}

// AddLatency adds latency to the total.
func AddLatency(ms int64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	metrics.TotalLatency += ms
}

// GetMetrics returns the current metrics.
func GetMetrics() Metrics {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	return metrics
}

// RecordCommand records a command execution for metrics.
func RecordCommand(duration time.Duration) {
	IncCommandCount()
	AddLatency(duration.Milliseconds())
}
