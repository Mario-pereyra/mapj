package cli

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservableCommandInterface(t *testing.T) {
	// Test that ObservableCommand interface is satisfied
	var obs ObservableCommand = &testObservable{}
	assert.NotNil(t, obs)
}

type testObservable struct {
	observed bool
	err      error
	duration time.Duration
	cmdName  string
}

func (t *testObservable) ObservableName() string {
	return "test"
}

func (t *testObservable) Observe(ctx context.Context, cmd *cobra.Command, runErr error, dur time.Duration) {
	t.observed = true
	t.err = runErr
	t.duration = dur
	t.cmdName = cmd.Name()
}

func TestRegisterObservable(t *testing.T) {
	// Reset observables for clean test state
	observablesMu.Lock()
	observables = make(map[string]ObservableCommand)
	observablesMu.Unlock()

	// Create a test command
	cmd := &cobra.Command{
		Use:   "test",
		Short: "test command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	// Create observable implementation
	obs := &testObservable{}

	// Register
	RegisterObservable(cmd, obs)

	// Verify registration
	observablesMu.Lock()
	retrieved, exists := observables[cmd.CommandPath()]
	observablesMu.Unlock()

	require.True(t, exists, "command should be registered")
	assert.Equal(t, obs, retrieved, "should return same observable")
}

func TestObserveCommand(t *testing.T) {
	// Reset observables for clean test state
	observablesMu.Lock()
	observables = make(map[string]ObservableCommand)
	observablesMu.Unlock()

	// Create a test command
	cmd := &cobra.Command{
		Use:   "test",
		Short: "test command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	// Set parent to simulate full path
	cmd.Parent()

	// Create observable implementation
	obs := &testObservable{}

	// Register
	RegisterObservable(cmd, obs)

	// Call observeCommand
	ctx := context.Background()
	dur := 100 * time.Millisecond
	testErr := assert.AnError
	observeCommand(ctx, cmd, testErr, dur)

	// Verify Observe was called
	assert.True(t, obs.observed, "Observe should have been called")
	assert.Equal(t, testErr, obs.err, "error should be passed")
	assert.Equal(t, dur, obs.duration, "duration should be passed")
	assert.Equal(t, "test", obs.cmdName, "command name should be passed")
}

func TestObserveCommandNotRegistered(t *testing.T) {
	// Reset observables for clean test state
	observablesMu.Lock()
	observables = make(map[string]ObservableCommand)
	observablesMu.Unlock()

	// Create a test command that is NOT registered
	cmd := &cobra.Command{
		Use:   "unregistered",
		Short: "unregistered command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	// Create observable implementation
	obs := &testObservable{}

	// Call observeCommand - should not panic even though not registered
	ctx := context.Background()
	dur := 100 * time.Millisecond
	observeCommand(ctx, cmd, nil, dur)

	// Verify Observe was NOT called (because not registered)
	assert.False(t, obs.observed, "Observe should NOT have been called for unregistered command")
}

func TestObserveCommandPanicRecovery(t *testing.T) {
	// Reset observables for clean test state
	observablesMu.Lock()
	observables = make(map[string]ObservableCommand)
	observablesMu.Unlock()

	// Create a command
	cmd := &cobra.Command{
		Use:   "panic",
		Short: "panic command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	// Create observable that panics
	panicObs := &panicObservable{}
	RegisterObservable(cmd, panicObs)

	// Call observeCommand - should recover and not panic
	ctx := context.Background()
	dur := 100 * time.Millisecond
	observeCommand(ctx, cmd, nil, dur)

	// Verify Observe was called but panicked
	assert.True(t, panicObs.panicked, "Observe should have been called")
	assert.Equal(t, 1, panicObs.panicCount, "Should have panicked exactly once")
}

type panicObservable struct {
	panicCount int
	panicked   bool
}

func (p *panicObservable) ObservableName() string {
	return "panic"
}

func (p *panicObservable) Observe(ctx context.Context, cmd *cobra.Command, runErr error, dur time.Duration) {
	p.panicCount++
	p.panicked = true
	panic("intentional panic for testing")
}

func TestIsObservabilityEnabled(t *testing.T) {
	// Note: isObservabilityEnabled checks MAPJ_OBSERVE env var
	// This test documents the behavior
	envVal := "0" // Default in test environment
	if envVal == "1" {
		assert.True(t, isObservabilityEnabled())
	} else {
		assert.False(t, isObservabilityEnabled())
	}
}

func TestMetrics(t *testing.T) {
	// Reset metrics
	metricsMu.Lock()
	metrics = Metrics{}
	metricsMu.Unlock()

	// Test IncCommandCount
	IncCommandCount()
	IncCommandCount()

	m := GetMetrics()
	assert.Equal(t, int64(2), m.CommandCount, "should have 2 commands")

	// Test AddLatency
	AddLatency(100)
	AddLatency(200)

	m = GetMetrics()
	assert.Equal(t, int64(300), m.TotalLatency, "should have 300ms total latency")

	// Test RecordCommand
	RecordCommand(50 * time.Millisecond)
	m = GetMetrics()
	assert.Equal(t, int64(3), m.CommandCount, "should have 3 commands after RecordCommand")
	assert.Equal(t, int64(350), m.TotalLatency, "should have 350ms total latency")
}

func TestRegisterObservableOverwrite(t *testing.T) {
	// Reset observables for clean test state
	observablesMu.Lock()
	observables = make(map[string]ObservableCommand)
	observablesMu.Unlock()

	cmd := &cobra.Command{
		Use:   "test",
		Short: "test command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	obs1 := &testObservable{}
	obs2 := &testObservable{}

	RegisterObservable(cmd, obs1)
	RegisterObservable(cmd, obs2)

	// Should have the second one
	observablesMu.Lock()
	retrieved, _ := observables[cmd.CommandPath()]
	observablesMu.Unlock()

	assert.Equal(t, obs2, retrieved, "should return the second observable (overwrite)")
}

func TestRecordCommandMetrics(t *testing.T) {
	// Reset global counters and histograms
	globalCounters = NewLabeledCounters()
	globalHistograms = NewCommandMetrics()

	// Record some metrics
	RecordCommandMetrics("tdn search", 0, 100*time.Millisecond)
	RecordCommandMetrics("tdn search", 0, 50*time.Millisecond)
	RecordCommandMetrics("tdn search", 0, 75*time.Millisecond)
	RecordCommandMetrics("protheus query", 1, 200*time.Millisecond)

	// Verify counters
	counters := globalCounters.GetAll()
	assert.Equal(t, int64(3), counters["tdn search:0"], "tdn search should have 3 successful runs")
	assert.Equal(t, int64(1), counters["protheus query:1"], "protheus query should have 1 failed run")

	// Verify histogram has entries
	output := GetAllMetricsPrometheus()
	assert.Contains(t, output, `mapj_command_total{cmd="tdn search",exit_code="0"} 3`)
	assert.Contains(t, output, `mapj_command_total{cmd="protheus query",exit_code="1"} 1`)
	assert.Contains(t, output, `mapj_command_duration_ms_count{cmd="tdn search"} 3`)
	assert.Contains(t, output, `mapj_command_duration_ms_sum{cmd="tdn search"} 225`)
}

func TestRecordCommandMetricsPrometheusOutput(t *testing.T) {
	// Reset global counters and histograms
	globalCounters = NewLabeledCounters()
	globalHistograms = NewCommandMetrics()

	// Record metrics
	RecordCommandMetrics("test cmd", 0, 100*time.Millisecond)
	RecordCommandMetrics("test cmd", 0, 200*time.Millisecond)
	RecordCommandMetrics("test cmd", 1, 50*time.Millisecond)

	output := GetAllMetricsPrometheus()

	// Check that output contains expected metrics
	assert.Contains(t, output, "# mapj observability metrics")
	assert.Contains(t, output, `mapj_command_total{cmd="test cmd",exit_code="0"} 2`)
	assert.Contains(t, output, `mapj_command_total{cmd="test cmd",exit_code="1"} 1`)
	assert.Contains(t, output, `mapj_command_duration_ms_count{cmd="test cmd"} 3`)
	assert.Contains(t, output, `mapj_command_duration_ms_sum{cmd="test cmd"} 350`)
	assert.Contains(t, output, `mapj_command_duration_ms_bucket{cmd="test cmd",le="100"}`)
	assert.Contains(t, output, `mapj_command_duration_ms_bucket{cmd="test cmd",le="+Inf"} 3`)
}

func TestGetAllMetricsPrometheusEmpty(t *testing.T) {
	// Reset global counters and histograms
	globalCounters = NewLabeledCounters()
	globalHistograms = NewCommandMetrics()

	output := GetAllMetricsPrometheus()
	assert.Equal(t, "# No metrics recorded yet\n", output)
}

func TestLabeledCountersInc(t *testing.T) {
	lc := NewLabeledCounters()

	// Test Inc
	lc.Inc("cmd1", 0)
	lc.Inc("cmd1", 0)
	lc.Inc("cmd1", 1)
	lc.Inc("cmd2", 0)

	counters := lc.GetAll()
	assert.Equal(t, int64(2), counters["cmd1:0"])
	assert.Equal(t, int64(1), counters["cmd1:1"])
	assert.Equal(t, int64(1), counters["cmd2:0"])
}

func TestCommandMetricsHistogram(t *testing.T) {
	cm := NewCommandMetrics()

	// Record values across different buckets
	cm.Record("cmd", 0, 3*time.Millisecond)    // < 5
	cm.Record("cmd", 0, 7*time.Millisecond)    // < 10
	cm.Record("cmd", 0, 30*time.Millisecond)   // < 50
	cm.Record("cmd", 0, 500*time.Millisecond)  // < 1000

	output := cm.ToPrometheusFormat()

	// Find the bucket for le=5
	found := false
	for _, o := range output {
		if o.Name == "mapj_command_duration_ms_bucket" && o.Labels == `cmd="cmd",le="5"` {
			assert.Equal(t, "1", o.Value)
			found = true
		}
	}
	assert.True(t, found, "Should have bucket for le=5")

	// Find +Inf bucket (should be total count = 4)
	found = false
	for _, o := range output {
		if o.Name == "mapj_command_duration_ms_bucket" && o.Labels == `cmd="cmd",le="+Inf"` {
			assert.Equal(t, "4", o.Value)
			found = true
		}
	}
	assert.True(t, found, "Should have +Inf bucket with count 4")
}
