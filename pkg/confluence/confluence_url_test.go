package confluence

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfluenceURL(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectBase   string
		expectPageID string
		expectError  bool
	}{
		{
			name:         "Full URL with space",
			input:        "https://tdninterno.totvs.com/wiki/spaces/PROT/pages/123456789/Title",
			expectBase:   "https://tdninterno.totvs.com/wiki",
			expectPageID: "123456789",
			expectError:  false,
		},
		{
			name:         "Full URL with tilde user",
			input:        "https://company.atlassian.net/wiki/spaces/~user123/pages/987654321/Title",
			expectBase:   "",
			expectPageID: "",
			expectError:  true,
		},
		{
			name:         "Relative path",
			input:        "/spaces/PROT/pages/123456789",
			expectBase:   "",
			expectPageID: "123456789",
			expectError:  false,
		},
		{
			name:         "Page ID only",
			input:        "123456789",
			expectBase:   "",
			expectPageID: "123456789",
			expectError:  false,
		},
		{
			name:         "Empty string",
			input:        "",
			expectBase:   "",
			expectPageID: "",
			expectError:  true,
		},
		{
			name:         "Invalid URL",
			input:        "not-a-url-at-all",
			expectBase:   "",
			expectPageID: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, pageID, err := ParseConfluenceURL(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectBase, baseURL)
				assert.Equal(t, tt.expectPageID, pageID)
			}
		})
	}
}
