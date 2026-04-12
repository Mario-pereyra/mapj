package logging

import (
	"go.uber.org/zap"
)

// WithTraceID returns a logger with the traceId field added.
func WithTraceID() *zap.Logger {
	return zap.L().With(zap.String("traceId", GetTraceID()))
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
