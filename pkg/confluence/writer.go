package confluence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const maxPathLength = 220

// WriteExportedPage writes a page's markdown to disk with proper directory structure.
// Structure: {outputPath}/spaces/{spaceKey}/pages/{id}-{slug}.md
func WriteExportedPage(outputPath string, page *Page, markdown string) (string, error) {
	spaceKey := page.SpaceKey()
	if spaceKey == "" {
		spaceKey = "unknown"
	}

	category := "pages"
	if page.IsHomepage() {
		category = "homepages"
	}

	relativeParent := filepath.Join("spaces", sanitizeFilename(spaceKey), category)
	filename := buildStableFilename(page.ID, page.Title, ".md", outputPath, relativeParent)
	fullPath := filepath.Join(outputPath, relativeParent, filename)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(markdown), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filepath.Join(relativeParent, filename), nil
}

// WriteAttachment downloads and writes an attachment to disk.
func WriteAttachment(outputPath string, page *Page, data []byte, attachmentID, filename, extension string) (string, error) {
	spaceKey := page.SpaceKey()
	if spaceKey == "" {
		spaceKey = "unknown"
	}

	relativeParent := filepath.Join("spaces", sanitizeFilename(spaceKey), "attachments", page.ID)
	safeFilename := buildStableFilename(attachmentID, filename, extension, outputPath, relativeParent)
	fullPath := filepath.Join(outputPath, relativeParent, safeFilename)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write attachment: %w", err)
	}

	return filepath.Join(relativeParent, safeFilename), nil
}

// IsHomepage checks if this page is the space homepage.
func (p *Page) IsHomepage() bool {
	// A page is a homepage if it has no ancestors (beyond root)
	return len(p.Ancestors) <= 1
}

// GenerateFrontMatter produces YAML front matter for a page.
func GenerateFrontMatter(page *Page, baseURL, exportPath string) string {
	var b strings.Builder
	b.WriteString("---\n")

	writeField := func(key, value string) {
		if value != "" {
			b.WriteString(fmt.Sprintf("%s: %q\n", key, value))
		}
	}

	writeField("page_id", page.ID)
	writeField("title", page.Title)
	writeField("source_url", page.SourceURL(baseURL))
	writeField("space_key", page.SpaceKey())
	writeField("space_name", page.Space.Name)

	if labels := page.GetLabels(); len(labels) > 0 {
		b.WriteString("labels:\n")
		for _, l := range labels {
			b.WriteString(fmt.Sprintf("  - %q\n", l))
		}
	}

	if page.Version != nil {
		writeField("updated_at", page.Version.When)
		if page.Version.By != nil {
			writeField("author", page.Version.By.DisplayName)
		}
		b.WriteString(fmt.Sprintf("version: %d\n", page.Version.Number))
	}

	b.WriteString(fmt.Sprintf("exported_at: %q\n", time.Now().UTC().Format(time.RFC3339)))

	b.WriteString("---\n")
	return b.String()
}

// GenerateBreadcrumbs produces a breadcrumb line from ancestors.
func GenerateBreadcrumbs(page *Page) string {
	titles := page.AncestorTitles()
	titles = append(titles, page.Title)
	return strings.Join(titles, " > ") + "\n"
}

// WriteManifest writes or appends a manifest entry.
func WriteManifest(outputPath string, entry *ManifestEntry) error {
	manifestPath := filepath.Join(outputPath, "manifest.jsonl")

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest entry: %w", err)
	}

	f, err := os.OpenFile(manifestPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open manifest: %w", err)
	}
	defer f.Close()

	f.Write(data)
	f.Write([]byte("\n"))
	return nil
}

// ManifestEntry is a single entry in manifest.jsonl.
type ManifestEntry struct {
	PageID     string   `json:"page_id"`
	Title      string   `json:"title"`
	Slug       string   `json:"slug"`
	SourceURL  string   `json:"source_url"`
	SpaceKey   string   `json:"space_key"`
	SpaceName  string   `json:"space_name"`
	Labels     []string `json:"labels,omitempty"`
	ExportPath string   `json:"export_path"`
	ExportedAt string   `json:"exported_at"`
}

// WriteSpaceIndex generates a README.md index for a space.
func WriteSpaceIndex(outputPath, spaceKey, spaceName string, entries []*ManifestEntry) error {
	spaceDir := filepath.Join(outputPath, "spaces", sanitizeFilename(spaceKey))
	indexPath := filepath.Join(spaceDir, "README.md")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Space: %s (%s)\n\n", spaceName, spaceKey))
	b.WriteString(fmt.Sprintf("## Pages (%d documents)\n\n", len(entries)))

	for _, e := range entries {
		relPath, _ := filepath.Rel(spaceDir, filepath.Join(outputPath, e.ExportPath))
		relPath = filepath.ToSlash(relPath)
		labelStr := ""
		if len(e.Labels) > 0 {
			labelStr = " — Labels: " + strings.Join(e.Labels, ", ")
		}
		b.WriteString(fmt.Sprintf("- [%s](%s)%s\n", e.Title, relPath, labelStr))
	}

	if err := os.MkdirAll(spaceDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(indexPath, []byte(b.String()), 0644)
}

// ==================== PATH HELPERS ====================

func buildStableFilename(id, title, extension, outputPath, relativeParent string) string {
	slug := slugify(title, 80)
	candidate := fmt.Sprintf("%s-%s%s", id, slug, extension)
	fullCandidate := filepath.Join(outputPath, relativeParent, candidate)

	if len(fullCandidate) <= maxPathLength {
		return candidate
	}

	// Trim slug to fit
	fixedOverhead := len(filepath.Join(outputPath, relativeParent, fmt.Sprintf("%s-%s", id, extension)))
	remaining := maxPathLength - fixedOverhead
	if remaining > 8 {
		trimmedSlug := slugify(title, remaining)
		return fmt.Sprintf("%s-%s%s", id, trimmedSlug, extension)
	}

	return fmt.Sprintf("%s%s", id, extension)
}

func slugify(value string, maxLength int) string {
	// Normalize unicode and convert to ASCII
	normalized := norm.NFKD.String(value)
	var ascii strings.Builder
	for _, r := range normalized {
		if r < 128 {
			ascii.WriteRune(r)
		}
	}

	result := strings.ToLower(ascii.String())
	re := regexp.MustCompile(`[^a-z0-9]+`)
	result = re.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	if result == "" {
		result = "document"
	}

	if maxLength > 0 && len(result) > maxLength {
		result = result[:maxLength]
		result = strings.TrimRight(result, "-")
		if result == "" {
			result = "document"
		}
	}

	return result
}

func sanitizeFilename(name string) string {
	// Replace unsafe characters for filenames
	unsafe := regexp.MustCompile(`[<>:"/\\|?*\x00\[\]]`)
	result := unsafe.ReplaceAllString(name, "_")
	result = strings.TrimRight(result, " .")

	// Check reserved Windows names
	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
		"COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
		"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}
	upper := strings.ToUpper(strings.TrimSuffix(result, filepath.Ext(result)))
	if reserved[upper] {
		result = result + "_"
	}

	// Limit length
	if len(result) > 255 {
		result = result[:255]
	}

	return result
}

// isASCII checks if a rune is ASCII.
func isASCII(r rune) bool {
	return r < unicode.MaxASCII
}
