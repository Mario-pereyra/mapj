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

	// ── Hierarchy ──────────────────────────────────────────────────────────
	// Ancestors are ordered from root to immediate parent.
	// We skip the first ancestor (space root) as it's redundant with space_key.
	if len(page.Ancestors) > 0 {
		// immediate parent = last ancestor
		immediate := page.Ancestors[len(page.Ancestors)-1]
		writeField("parent_id", immediate.ID)
		writeField("parent_title", immediate.Title)
		b.WriteString(fmt.Sprintf("depth: %d\n", len(page.Ancestors)))

		// Full ancestor chain as YAML list
		b.WriteString("ancestors:\n")
		for _, a := range page.Ancestors {
			b.WriteString(fmt.Sprintf("  - id: %q\n    title: %q\n", a.ID, a.Title))
		}

		// Human-readable breadcrumb line
		titles := page.AncestorTitles()
		titles = append(titles, page.Title)
		writeField("breadcrumb", strings.Join(titles, " > "))
	} else {
		b.WriteString("depth: 0\n")
	}
	// ───────────────────────────────────────────────────────────────────────

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
	PageID     string        `json:"page_id"`
	Title      string        `json:"title"`
	Slug       string        `json:"slug"`
	SourceURL  string        `json:"source_url"`
	SpaceKey   string        `json:"space_key"`
	SpaceName  string        `json:"space_name"`
	Labels     []string      `json:"labels,omitempty"`
	ExportPath string        `json:"export_path"`
	ExportedAt string        `json:"exported_at"`
	// Hierarchy fields
	ParentID   string        `json:"parent_id,omitempty"`
	ParentTitle string       `json:"parent_title,omitempty"`
	Depth      int           `json:"depth"`
	Ancestors  []AncestorRef `json:"ancestors,omitempty"`
	Breadcrumb string        `json:"breadcrumb,omitempty"`
}

// WriteSpaceIndex generates a README.md index and a hierarchy tree for a space.
func WriteSpaceIndex(outputPath, spaceKey, spaceName string, entries []*ManifestEntry) error {
	spaceDir := filepath.Join(outputPath, "spaces", sanitizeFilename(spaceKey))
	if err := os.MkdirAll(spaceDir, 0755); err != nil {
		return err
	}

	// ── README.md: flat index ordered by breadcrumb ────────────────────────
	indexPath := filepath.Join(spaceDir, "README.md")
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Space: %s (%s)\n\n", spaceName, spaceKey))
	b.WriteString(fmt.Sprintf("Exported: %d pages — see `tree.md` for the full hierarchy.\n\n", len(entries)))
	b.WriteString(fmt.Sprintf("## Pages (%d documents)\n\n", len(entries)))

	for _, e := range entries {
		relPath, _ := filepath.Rel(spaceDir, filepath.Join(outputPath, e.ExportPath))
		relPath = filepath.ToSlash(relPath)
		indent := strings.Repeat("  ", e.Depth)
		crumb := e.Breadcrumb
		if crumb == "" {
			crumb = e.Title
		}
		labelStr := ""
		if len(e.Labels) > 0 {
			labelStr = " — Labels: " + strings.Join(e.Labels, ", ")
		}
		b.WriteString(fmt.Sprintf("%s- [%s](%s)%s\n", indent, e.Title, relPath, labelStr))
		_ = crumb
	}
	if err := os.WriteFile(indexPath, []byte(b.String()), 0644); err != nil {
		return err
	}

	// ── tree.md: ASCII hierarchy tree ─────────────────────────────────────
	if err := WriteHierarchyTree(spaceDir, spaceName, spaceKey, entries); err != nil {
		return err
	}

	return nil
}

// WriteHierarchyTree generates tree.md (ASCII art) inside the space directory.
func WriteHierarchyTree(spaceDir, spaceName, spaceKey string, entries []*ManifestEntry) error {
	// Build parent→children map
	type node struct {
		Entry    *ManifestEntry
		Children []*ManifestEntry
	}
	byID := make(map[string]*ManifestEntry, len(entries))
	children := make(map[string][]*ManifestEntry)
	var roots []*ManifestEntry

	for _, e := range entries {
		byID[e.PageID] = e
	}
	for _, e := range entries {
		if e.ParentID == "" {
			roots = append(roots, e)
		} else if _, parentInSet := byID[e.ParentID]; parentInSet {
			children[e.ParentID] = append(children[e.ParentID], e)
		} else {
			// Parent not exported — treat as root
			roots = append(roots, e)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s (%s) — Page Hierarchy\n\n", spaceName, spaceKey))
	b.WriteString(fmt.Sprintf("Total: %d pages\n\n", len(entries)))
	b.WriteString("```\n")

	var writeNode func(e *ManifestEntry, prefix string, isLast bool)
	writeNode = func(e *ManifestEntry, prefix string, isLast bool) {
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}
		b.WriteString(fmt.Sprintf("%s%s%s [%s]\n", prefix, connector, e.Title, e.PageID))
		kids := children[e.PageID]
		for i, kid := range kids {
			writeNode(kid, childPrefix, i == len(kids)-1)
		}
	}

	for i, r := range roots {
		writeNode(r, "", i == len(roots)-1)
	}
	b.WriteString("```\n")

	return os.WriteFile(filepath.Join(spaceDir, "tree.md"), []byte(b.String()), 0644)
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
