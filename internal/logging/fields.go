package logging

import (
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// WithTraceID returns a logger with the traceId field added.
func WithTraceID() *zap.Logger {
	return zap.L().With(zap.String("traceId", GetTraceID()))
}

// WithCommandMetadata returns a logger with command metadata fields added.
func WithCommandMetadata(cmdPath string) *zap.Logger {
	return WithTraceID().With(
		zap.String("command", cmdPath),
	)
}

// Debug logs a message at Debug level.
func Debug(msg string, fields ...zap.Field) {
	WithTraceID().Debug(msg, fields...)
}

// Info logs a message at Info level.
func Info(msg string, fields ...zap.Field) {
	WithTraceID().Info(msg, fields...)
}

// Warn logs a message at Warn level.
func Warn(msg string, fields ...zap.Field) {
	WithTraceID().Warn(msg, fields...)
}

// Error logs a message at Error level.
func Error(msg string, fields ...zap.Field) {
	WithTraceID().Error(msg, fields...)
}

// Fatal logs a message at Fatal level and then calls os.Exit(1).
func Fatal(msg string, fields ...zap.Field) {
	WithTraceID().Fatal(msg, fields...)
}

// Sync flushes any buffered log entries.
func Sync() {
	zap.L().Sync()
}

// GetCommandPath returns the full command path (e.g., "tdn search" from "mapj tdn search").
func GetCommandPath(cmd *cobra.Command) string {
	path := cmd.Name()
	for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Name() != "mapj" {
			path = parent.Name() + " " + path
		}
	}
	return path
}

// LogCommandStart logs the start of a command execution.
func LogCommandStart(cmd *cobra.Command) {
	cmdPath := GetCommandPath(cmd)
	WithCommandMetadata(cmdPath).Info("command started",
		zap.String("status", "started"),
	)
}

// LogCommandComplete logs the completion of a command execution.
func LogCommandComplete(cmd *cobra.Command, duration time.Duration, success bool) {
	cmdPath := GetCommandPath(cmd)
	status := "success"
	if !success {
		status = "error"
	}
	WithCommandMetadata(cmdPath).Info("command completed",
		zap.String("status", status),
		zap.Int64("latencyMs", duration.Milliseconds()),
	)
}
