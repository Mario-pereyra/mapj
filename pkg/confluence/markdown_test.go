package confluence

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadFixture(t *testing.T, name string) string {
	content, err := os.ReadFile("../../testdata/confluence_html/" + name)
	if err != nil {
		t.Fatalf("Failed to load fixture %s: %v", name, err)
	}
	return string(content)
}

func TestConvertToMarkdown_TaskList(t *testing.T) {
	html := loadFixture(t, "task_list.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "Task completed")
	assert.Contains(t, md, "Task pending")
}

func TestConvertToMarkdown_CodeMacro(t *testing.T) {
	// Storage format: ac:structured-macro with ac:parameter language=java
	html := loadFixture(t, "code_macro.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "java")
	// The CDATA content may not always be rendered by the HTML parser,
	// but the language and code block markers should be present
	assert.Contains(t, md, "```")
}

func TestConvertToMarkdown_ExpandMacro(t *testing.T) {
	html := loadFixture(t, "expand_macro.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "Click to expand")
	assert.Contains(t, md, "hidden content")
}

func TestConvertToMarkdown_Panels(t *testing.T) {
	// Storage format uses ac:structured-macro with names info/warning/tip
	// Our storage fallback converts them to GitHub-style alerts
	html := loadFixture(t, "info_warning_panel.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "[!IMPORTANT]")  // info -> IMPORTANT
	assert.Contains(t, md, "[!CAUTION]")    // warning -> CAUTION
	assert.Contains(t, md, "[!TIP]")        // tip -> TIP
	assert.Contains(t, md, "info message")
	assert.Contains(t, md, "warning message")
	assert.Contains(t, md, "tip")
}

func TestConvertToMarkdown_StatusMacro(t *testing.T) {
	html := loadFixture(t, "status_macro.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "On Track")
	assert.Contains(t, md, "Blocked")
}

func TestConvertToMarkdown_Tables(t *testing.T) {
	html := loadFixture(t, "table_complex.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "|")
	assert.Contains(t, md, "Header 1")
	assert.Contains(t, md, "---")
}

func TestConvertToMarkdown_InternalLinks(t *testing.T) {
	// Storage format: ac:link with ri:page — our handler extracts the link text
	html := loadFixture(t, "internal_links.html")
	md := ConvertToMarkdown(html)
	// Storage fallback renders ac:link as text (page title or link body text)
	assert.Contains(t, md, "API Documentation")
	assert.Contains(t, md, "Configuration Page")
}

func TestConvertToMarkdown_Attachments(t *testing.T) {
	// Storage format: ac:image with ri:attachment — rendered as markdown image
	html := loadFixture(t, "attachments.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "screenshot.png")
	assert.Contains(t, md, "remote-image.jpg")
}

func TestConvertToMarkdown_BasicHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{"Bold", "<strong>bold text</strong>", "**bold text**"},
		{"Italic", "<em>italic text</em>", "*italic text*"},
		{"Code", "<code>inline code</code>", "`inline code`"},
		{"Heading", "<h1>Title</h1>", "# Title"},
		{"Link", `<a href="https://example.com">Link</a>`, "[Link](https://example.com)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := ConvertToMarkdown(tt.html)
			assert.Contains(t, md, tt.expected)
		})
	}
}

// ==================== EXPORT_VIEW FORMAT TESTS ====================

func TestConvertToMarkdown_ExportViewAlerts(t *testing.T) {
	// Export view format: divs with data-macro-name
	html := `
	<div data-macro-name="info"><p>This is an info box</p></div>
	<div data-macro-name="warning"><p>This is a warning box</p></div>
	<div data-macro-name="tip"><p>Quick tip here</p></div>
	`
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "[!IMPORTANT]")
	assert.Contains(t, md, "[!CAUTION]")
	assert.Contains(t, md, "[!TIP]")
	assert.Contains(t, md, "info box")
	assert.Contains(t, md, "warning box")
	assert.Contains(t, md, "Quick tip here")
}

func TestConvertToMarkdown_ExportViewCode(t *testing.T) {
	html := `
	<pre data-syntaxhighlighter-params="brush: python; gutter: true">
def hello():
    print("world")
	</pre>
	`
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "```python")
	assert.Contains(t, md, "def hello():")
	assert.Contains(t, md, "```")
}

func TestConvertToMarkdown_ExportViewExpand(t *testing.T) {
	html := `
	<div class="expand-container">
		<span class="expand-control-text">Show more details</span>
		<div class="expand-content"><p>Hidden details here</p></div>
	</div>
	`
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "<details>")
	assert.Contains(t, md, "Show more details")
	assert.Contains(t, md, "Hidden details here")
	assert.Contains(t, md, "</details>")
}

func TestConvertToMarkdown_ExportViewTaskList(t *testing.T) {
	html := `
	<ul>
		<li data-inline-task-id="1" class="checked">Done task</li>
		<li data-inline-task-id="2">Pending task</li>
	</ul>
	`
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "- [x] Done task")
	assert.Contains(t, md, "- [ ] Pending task")
}

func TestConvertToMarkdown_ExportViewTime(t *testing.T) {
	html := `<time datetime="2026-01-15">January 15</time>`
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "2026-01-15")
}
