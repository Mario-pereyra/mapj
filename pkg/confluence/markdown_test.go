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
	html := loadFixture(t, "code_macro.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "java")
	assert.Contains(t, md, "Example.java")
}

func TestConvertToMarkdown_ExpandMacro(t *testing.T) {
	html := loadFixture(t, "expand_macro.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "Click to expand")
	assert.Contains(t, md, "hidden content")
}

func TestConvertToMarkdown_Panels(t *testing.T) {
	html := loadFixture(t, "info_warning_panel.html")
	md := ConvertToMarkdown(html)
	assert.Contains(t, md, "Information")
	assert.Contains(t, md, "Warning")
	assert.Contains(t, md, "Pro Tip")
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
	assert.Contains(t, md, "| Header 1 |")
	assert.Contains(t, md, "| Merged Header |")
	assert.Contains(t, md, "---")
}

func TestConvertToMarkdown_InternalLinks(t *testing.T) {
	html := loadFixture(t, "internal_links.html")
	md := ConvertToMarkdown(html)
	assert.Empty(t, md)
}

func TestConvertToMarkdown_Attachments(t *testing.T) {
	html := loadFixture(t, "attachments.html")
	md := ConvertToMarkdown(html)
	assert.Empty(t, md)
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
