package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// LogLevel is the current log level.
	LogLevel = "info"
)

// Config holds the logging configuration.
type Config struct {
	Level  string
	TraceID string
}

// Init initializes the global logger with the given configuration.
// Must be called before using the logger.
func Init(cfg Config) {
	level := parseLevel(cfg.Level)
	
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stderr),
		level,
	)
	
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(logger)
	
	// Update LogLevel to reflect actual level
	LogLevel = cfg.Level
}

// parseLevel converts a string level to zapcore.Level.
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// GetLevel returns the current log level as a string.
func GetLevel() string {
	return LogLevel
}

// SetLevel sets the current log level.
func SetLevel(level string) {
	LogLevel = level
}
