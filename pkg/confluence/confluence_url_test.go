package confluence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfluenceInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectBase   string
		expectPageID string
		expectSpace  string
		expectTitle  string
		expectError  bool
	}{
		{
			name:         "Full Cloud URL with space and page ID",
			input:        "https://tdninterno.totvs.com/wiki/spaces/PROT/pages/123456789/Title",
			expectBase:   "https://tdninterno.totvs.com",
			expectPageID: "123456789",
			expectSpace:  "PROT",
		},
		{
			name:         "Cloud URL with tilde user space",
			input:        "https://company.atlassian.net/wiki/spaces/~user123/pages/987654321/Title",
			expectBase:   "https://company.atlassian.net",
			expectPageID: "987654321",
			expectSpace:  "~user123",
		},
		{
			name:         "Relative path with page ID",
			input:        "/spaces/PROT/pages/123456789",
			expectPageID: "123456789",
			expectSpace:  "PROT",
		},
		{
			name:         "Page ID only",
			input:        "123456789",
			expectPageID: "123456789",
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
		},
		{
			name:         "ViewPage action URL",
			input:        "https://tdninterno.totvs.com/pages/viewpage.action?pageId=22479548",
			expectBase:   "https://tdninterno.totvs.com",
			expectPageID: "22479548",
		},
		{
			name:         "ReleaseView action URL",
			input:        "https://tdninterno.totvs.com/pages/releaseview.action?pageId=22479548",
			expectBase:   "https://tdninterno.totvs.com",
			expectPageID: "22479548",
		},
		{
			name:        "Server display format",
			input:       "https://tdn.totvs.com/display/tec/REST+API+Guide",
			expectBase:  "https://tdn.totvs.com",
			expectSpace: "tec",
			expectTitle: "REST API Guide",
		},
		{
			name:        "Invalid single word",
			input:       "not-a-url-at-all",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseConfluenceInput(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.expectBase != "" {
				assert.Equal(t, tt.expectBase, result.BaseURL, "BaseURL")
			}
			if tt.expectPageID != "" {
				assert.Equal(t, tt.expectPageID, result.PageID, "PageID")
			}
			if tt.expectSpace != "" {
				assert.Equal(t, tt.expectSpace, result.SpaceKey, "SpaceKey")
			}
			if tt.expectTitle != "" {
				assert.Equal(t, tt.expectTitle, result.Title, "Title")
			}
		})
	}
}
