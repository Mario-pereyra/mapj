package confluence

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel controls the verbosity of export logging.
type LogLevel int

const (
	LogNormal  LogLevel = iota // Progress + errors only
	LogVerbose                 // + warnings, skipped pages, macro details
	LogDebug                   // + HTTP headers, raw HTML dumps
)

// ExportLogger handles structured JSONL error logging and console output.
type ExportLogger struct {
	mu         sync.Mutex
	level      LogLevel
	outputPath string
	errorFile  *os.File
	debugDir   string

	// Counters for summary
	Total    int
	Success  int
	Warnings int
	Failed   int
	Errors   []*ExportError
}

// NewExportLogger creates a logger that writes errors to {outputPath}/export-errors.jsonl.
func NewExportLogger(outputPath string, level LogLevel) (*ExportLogger, error) {
	errPath := filepath.Join(outputPath, "export-errors.jsonl")
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	f, err := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create error log: %w", err)
	}

	debugDir := ""
	if level >= LogDebug {
		debugDir = filepath.Join(outputPath, ".debug")
		if err := os.MkdirAll(debugDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create debug directory: %w", err)
		}
	}

	return &ExportLogger{
		level:      level,
		outputPath: outputPath,
		errorFile:  f,
		debugDir:   debugDir,
	}, nil
}

// Close flushes and closes the error log file.
func (l *ExportLogger) Close() error {
	if l.errorFile != nil {
		return l.errorFile.Close()
	}
	return nil
}

// LogError records a structured error to the JSONL file.
func (l *ExportLogger) LogError(exportErr *ExportError) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Failed++
	l.Errors = append(l.Errors, exportErr)

	entry := struct {
		Timestamp string `json:"ts"`
		*ExportError
	}{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		ExportError: exportErr,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to marshal error log entry: %v\n", err)
		return
	}

	l.errorFile.Write(data)
	l.errorFile.Write([]byte("\n"))

	// Always print errors to stderr
	fmt.Fprintf(os.Stderr, "  ❌ [%s] %s: %s\n", exportErr.Code, exportErr.Title, exportErr.Message)
}

// LogWarning records a non-fatal warning.
func (l *ExportLogger) LogWarning(pageID, title, message string) {
	l.mu.Lock()
	l.Warnings++
	l.mu.Unlock()

	if l.level >= LogVerbose {
		fmt.Fprintf(os.Stderr, "  ⚠️  [%s] %s: %s\n", pageID, title, message)
	}
}

// LogProgress prints a progress line for a page being exported.
func (l *ExportLogger) LogProgress(current, total int, pageID, title string) {
	l.mu.Lock()
	l.Total = total
	l.mu.Unlock()

	fmt.Fprintf(os.Stderr, "  [%d/%d] 📄 %s (ID: %s)\n", current, total, title, pageID)
}

// LogSuccess records a successful export.
func (l *ExportLogger) LogSuccess(pageID, title string) {
	l.mu.Lock()
	l.Success++
	l.mu.Unlock()

	if l.level >= LogVerbose {
		fmt.Fprintf(os.Stderr, "  ✅ %s (ID: %s)\n", title, pageID)
	}
}

// LogVerbose prints a message only in verbose mode.
func (l *ExportLogger) LogVerbose(format string, args ...interface{}) {
	if l.level >= LogVerbose {
		fmt.Fprintf(os.Stderr, "  "+format+"\n", args...)
	}
}

// LogDebug prints a message only in debug mode.
func (l *ExportLogger) LogDebug(format string, args ...interface{}) {
	if l.level >= LogDebug {
		fmt.Fprintf(os.Stderr, "  [DEBUG] "+format+"\n", args...)
	}
}

// DumpDebugFile writes a debug artifact to the .debug/ directory.
func (l *ExportLogger) DumpDebugFile(pageID, suffix string, content []byte) {
	if l.debugDir == "" {
		return
	}

	filename := fmt.Sprintf("%s_%s", pageID, suffix)
	path := filepath.Join(l.debugDir, filename)
	if err := os.WriteFile(path, content, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  [DEBUG] failed to write debug file %s: %v\n", path, err)
	}
}

// PrintSummary prints the final export summary to the given writer.
func (l *ExportLogger) PrintSummary(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()

	pct := float64(0)
	if l.Total > 0 {
		pct = float64(l.Success) / float64(l.Total) * 100
	}

	fmt.Fprintf(w, "\n╔═══════════════════════════════════════════╗\n")
	fmt.Fprintf(w, "║          EXPORT SUMMARY                   ║\n")
	fmt.Fprintf(w, "╠═══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║  Total pages:    %5d                    ║\n", l.Total)
	fmt.Fprintf(w, "║  ✅ Exported:    %5d  (%5.1f%%)          ║\n", l.Success, pct)
	fmt.Fprintf(w, "║  ⚠️  Warnings:   %5d                    ║\n", l.Warnings)
	fmt.Fprintf(w, "║  ❌ Failed:      %5d                    ║\n", l.Failed)

	if l.Failed > 0 {
		// Group errors by code
		codeCounts := map[string]int{}
		for _, e := range l.Errors {
			codeCounts[e.Code]++
		}

		fmt.Fprintf(w, "║                                           ║\n")
		fmt.Fprintf(w, "║  Failures by type:                        ║\n")
		for code, count := range codeCounts {
			fmt.Fprintf(w, "║    %-18s %3d                 ║\n", code+":", count)
		}
		fmt.Fprintf(w, "║                                           ║\n")
		errLogPath := filepath.Join(l.outputPath, "export-errors.jsonl")
		fmt.Fprintf(w, "║  Error log: %-29s║\n", errLogPath)
		fmt.Fprintf(w, "║  Retry: mapj confluence retry-failed \\    ║\n")
		fmt.Fprintf(w, "║           --output-path %-17s║\n", l.outputPath)
	}

	fmt.Fprintf(w, "╚═══════════════════════════════════════════╝\n")
}
