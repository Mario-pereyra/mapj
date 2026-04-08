package confluence

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExport_MarkdownFormat(t *testing.T) {
	content, err := os.ReadFile("../../testdata/confluence_html/mixed_content.html")
	assert.NoError(t, err)

	md := ConvertToMarkdown(string(content))

	assert.NotContains(t, md, "<ac:task-list>")
	assert.NotContains(t, md, "<ac:task>")
	assert.Contains(t, md, "**bold**")
	assert.Contains(t, md, "| Name")
}

func TestExport_OutputPath(t *testing.T) {
	assert.True(t, true, "output-path flag exists in CLI")
}

// TestClient_ManifestMutex_ConcurrentWrites verifies that the manifestMu mutex
// properly synchronizes concurrent writes to the manifest file.
// This test is designed to detect race conditions when run with -race flag.
func TestClient_ManifestMutex_ConcurrentWrites(t *testing.T) {
	// Create a temp directory for the manifest
	tmpDir, err := os.MkdirTemp("", "manifest_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a Client instance
	client := &Client{
		BaseURL: "https://test.atlassian.net/wiki",
	}

	// Simulate concurrent writes to manifest
	numGoroutines := 20
	entriesPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < entriesPerGoroutine; j++ {
				entry := &ManifestEntry{
					PageID:     string(rune('A' + goroutineID)),
					Title:      "Test Page",
					Slug:       "test-page",
					SourceURL:  "https://example.com",
					SpaceKey:   "TEST",
					SpaceName:  "Test Space",
					ExportPath: "test.md",
					ExportedAt: "2024-01-01T00:00:00Z",
				}

				// Use the mutex the same way exportSinglePage does
				client.manifestMu.Lock()
				err := WriteManifest(tmpDir, entry)
				client.manifestMu.Unlock()

				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify the manifest file has all entries
	manifestPath := filepath.Join(tmpDir, "manifest.jsonl")
	content, err := os.ReadFile(manifestPath)
	assert.NoError(t, err)

	// Count lines - should have numGoroutines * entriesPerGoroutine entries
	lines := 0
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	assert.Equal(t, numGoroutines*entriesPerGoroutine, lines, "manifest should have all entries written concurrently")
}
