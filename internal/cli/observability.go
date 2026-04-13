package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
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

// CommandMetricEntry represents metrics for a specific command with labels.
type CommandMetricEntry struct {
	Count   int64
	TotalMs int64
	MinMs   int64
	MaxMs   int64
	SumSqMs int64 // sum of squares for stddev calculation
	Buckets map[int64]int64
}

// CommandMetrics is a thread-safe map of command metrics keyed by command name.
type CommandMetrics struct {
	mu      sync.Mutex
	metrics map[string]*CommandMetricEntry
}

// NewCommandMetrics creates a new CommandMetrics collector.
func NewCommandMetrics() *CommandMetrics {
	return &CommandMetrics{
		metrics: make(map[string]*CommandMetricEntry),
	}
}

// HistogramBucket boundaries in milliseconds
var histogramBuckets = []int64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}

func (cm *CommandMetrics) getOrCreate(cmd string) *CommandMetricEntry {
	if entry, exists := cm.metrics[cmd]; exists {
		return entry
	}
	entry := &CommandMetricEntry{
		MinMs:   -1, // -1 indicates unset
		Buckets: make(map[int64]int64),
	}
	for _, bucket := range histogramBuckets {
		entry.Buckets[bucket] = 0
	}
	cm.metrics[cmd] = entry
	return entry
}

// Record records a command execution with exit code.
func (cm *CommandMetrics) Record(cmd string, exitCode int, duration time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ms := duration.Milliseconds()
	entry := cm.getOrCreate(cmd)

	entry.Count++
	entry.TotalMs += ms

	if entry.MinMs == -1 || ms < entry.MinMs {
		entry.MinMs = ms
	}
	if ms > entry.MaxMs {
		entry.MaxMs = ms
	}
	entry.SumSqMs += ms * ms

	// Update histogram buckets
	for _, bucket := range histogramBuckets {
		if ms <= bucket {
			entry.Buckets[bucket]++
		}
	}
}

// CounterEntry represents a counter with labels.
type CounterEntry struct {
	Count int64
}

// LabeledCounters is a thread-safe map of counters keyed by "cmd:exitCode".
type LabeledCounters struct {
	mu       sync.Mutex
	counters map[string]*CounterEntry
}

// NewLabeledCounters creates a new LabeledCounters collector.
func NewLabeledCounters() *LabeledCounters {
	return &LabeledCounters{
		counters: make(map[string]*CounterEntry),
	}
}

// key creates a unique key for cmd and exitCode combination.
func counterKey(cmd string, exitCode int) string {
	return fmt.Sprintf("%s:%d", cmd, exitCode)
}

// Inc increments the counter for the given cmd and exitCode.
func (lc *LabeledCounters) Inc(cmd string, exitCode int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	key := counterKey(cmd, exitCode)
	if entry, exists := lc.counters[key]; exists {
		entry.Count++
	} else {
		lc.counters[key] = &CounterEntry{Count: 1}
	}
}

// GetAll returns all counters as a map of key -> count.
func (lc *LabeledCounters) GetAll() map[string]int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	result := make(map[string]int64)
	for key, entry := range lc.counters {
		result[key] = entry.Count
	}
	return result
}

// globalMetrics holds the global metrics collectors
var (
	globalCounters   = NewLabeledCounters()
	globalHistograms = NewCommandMetrics()
)

// RecordCommandMetrics records metrics for a command execution.
// cmdName is the command path, exitCode is the process exit code, duration is execution time.
func RecordCommandMetrics(cmdName string, exitCode int, duration time.Duration) {
	globalCounters.Inc(cmdName, exitCode)
	globalHistograms.Record(cmdName, exitCode, duration)
}

// PromMetricsOutput represents the prometheus text format output for a single metric.
type PromMetricsOutput struct {
	Name       string
	Labels     string
	Value      string
	Help       string
	MetricType string
}

// ToPrometheusFormat converts command metrics to Prometheus text format.
func (cm *CommandMetrics) ToPrometheusFormat() []PromMetricsOutput {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var results []PromMetricsOutput
	helpText := "Command duration in milliseconds"

	// Command count (counter)
	for cmd, entry := range cm.metrics {
		results = append(results, PromMetricsOutput{
			Name:       "mapj_command_duration_ms_count",
			Labels:     fmt.Sprintf(`cmd="%s"`, cmd),
			Value:      fmt.Sprintf("%d", entry.Count),
			Help:       helpText,
			MetricType: "counter",
		})
	}

	// Total latency sum (counter)
	for cmd, entry := range cm.metrics {
		results = append(results, PromMetricsOutput{
			Name:       "mapj_command_duration_ms_sum",
			Labels:     fmt.Sprintf(`cmd="%s"`, cmd),
			Value:      fmt.Sprintf("%d", entry.TotalMs),
			Help:       helpText,
			MetricType: "counter",
		})
	}

	// Histogram buckets
	for cmd, entry := range cm.metrics {
		for bucket, count := range entry.Buckets {
			results = append(results, PromMetricsOutput{
				Name:       "mapj_command_duration_ms_bucket",
				Labels:     fmt.Sprintf(`cmd="%s",le="%d"`, cmd, bucket),
				Value:      fmt.Sprintf("%d", count),
				Help:       helpText,
				MetricType: "histogram",
			})
		}
		// +Inf bucket equals total count
		results = append(results, PromMetricsOutput{
			Name:       "mapj_command_duration_ms_bucket",
			Labels:     fmt.Sprintf(`cmd="%s",le="+Inf"`, cmd),
			Value:      fmt.Sprintf("%d", entry.Count),
			Help:       helpText,
			MetricType: "histogram",
		})
	}

	// Min/Max (gauge)
	for cmd, entry := range cm.metrics {
		if entry.MinMs >= 0 {
			results = append(results, PromMetricsOutput{
				Name:       "mapj_command_duration_ms_min",
				Labels:     fmt.Sprintf(`cmd="%s"`, cmd),
				Value:      fmt.Sprintf("%d", entry.MinMs),
				Help:       "Minimum command duration in milliseconds",
				MetricType: "gauge",
			})
		}
		results = append(results, PromMetricsOutput{
			Name:       "mapj_command_duration_ms_max",
			Labels:     fmt.Sprintf(`cmd="%s"`, cmd),
			Value:      fmt.Sprintf("%d", entry.MaxMs),
			Help:       "Maximum command duration in milliseconds",
			MetricType: "gauge",
		})
	}

	return results
}

// ToPrometheusFormat converts labeled counters to Prometheus text format.
func (lc *LabeledCounters) ToPrometheusFormat() []PromMetricsOutput {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	var results []PromMetricsOutput

	for key, entry := range lc.counters {
		// Parse key back to cmd and exitCode
		// Key format is "cmdName:exitCode"
		var cmd string
		var exitCode int
		for i := len(key) - 1; i >= 0; i-- {
			if key[i] == ':' {
				cmd = key[:i]
				fmt.Sscanf(key[i+1:], "%d", &exitCode)
				break
			}
		}

		results = append(results, PromMetricsOutput{
			Name:       "mapj_command_total",
			Labels:     fmt.Sprintf(`cmd="%s",exit_code="%d"`, cmd, exitCode),
			Value:      fmt.Sprintf("%d", entry.Count),
			Help:       "Total number of commands executed",
			MetricType: "counter",
		})
	}

	return results
}

// GetAllMetrics returns all metrics in Prometheus text format.
func GetAllMetricsPrometheus() string {
	var lines []string

	// Header comment
	lines = append(lines, "# mapj observability metrics")
	lines = append(lines, "# Generated by mapj observability")

	// Get counters
	counterOutputs := globalCounters.ToPrometheusFormat()
	for _, output := range counterOutputs {
		if output.Help != "" {
			lines = append(lines, fmt.Sprintf("# HELP %s %s", output.Name, output.Help))
		}
		if output.MetricType != "" {
			lines = append(lines, fmt.Sprintf("# TYPE %s %s", output.Name, output.MetricType))
		}
		lines = append(lines, fmt.Sprintf("%s{%s} %s", output.Name, output.Labels, output.Value))
	}

	// Get histograms
	histogramOutputs := globalHistograms.ToPrometheusFormat()
	// Sort for consistent output
	sort.Slice(histogramOutputs, func(i, j int) bool {
		if histogramOutputs[i].Name != histogramOutputs[j].Name {
			return histogramOutputs[i].Name < histogramOutputs[j].Name
		}
		return histogramOutputs[i].Labels < histogramOutputs[j].Labels
	})

	for _, output := range histogramOutputs {
		if output.Help != "" {
			lines = append(lines, fmt.Sprintf("# HELP %s %s", output.Name, output.Help))
		}
		if output.MetricType != "" {
			lines = append(lines, fmt.Sprintf("# TYPE %s %s", output.Name, output.MetricType))
		}
		lines = append(lines, fmt.Sprintf("%s{%s} %s", output.Name, output.Labels, output.Value))
	}

	if len(lines) == 2 { // Only header, no metrics
		return "# No metrics recorded yet\n"
	}

	return fmt.Sprintf("%s\n", joinLines(lines))
}

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}
