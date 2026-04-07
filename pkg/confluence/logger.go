package confluence

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// LogLevel controls the verbosity of export logging.
type LogLevel int

const (
	LogNormal  LogLevel = iota // Progress + errors only
	LogVerbose                 // + warnings, skipped pages, macro details
)

// ExportLogger handles progress and error logging to stderr.
type ExportLogger struct {
	mu         sync.Mutex
	level      LogLevel
	outputPath string

	// Counters for summary
	Total    int
	Success  int
	Warnings int
	Failed   int
	Errors   []*ExportError
}

// NewExportLogger creates a logger for export operations.
func NewExportLogger(outputPath string, level LogLevel) (*ExportLogger, error) {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &ExportLogger{
		level:      level,
		outputPath: outputPath,
	}, nil
}

// Close is a no-op now that we don't write to error files.
func (l *ExportLogger) Close() error {
	return nil
}

// LogError records an error.
func (l *ExportLogger) LogError(exportErr *ExportError) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Failed++
	l.Errors = append(l.Errors, exportErr)

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
	}

	fmt.Fprintf(w, "╚═══════════════════════════════════════════╝\n")
}
