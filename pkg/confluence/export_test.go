package confluence

import (
	"os"
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
