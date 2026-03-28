package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AttachmentInfo holds metadata about a page attachment.
type AttachmentInfo struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Title      string               `json:"title"`
	MediaType  string               `json:"mediaType"`
	FileSize   int64                `json:"fileSize,omitempty"`
	Links      AttachmentLinks      `json:"_links"`
	Version    *Version             `json:"version,omitempty"`
	Extensions *AttachmentExtensions `json:"extensions,omitempty"`
}

// AttachmentLinks holds API links for an attachment.
type AttachmentLinks struct {
	Download string `json:"download,omitempty"`
	Self     string `json:"self,omitempty"`
}

// AttachmentExtensions holds extra attachment metadata.
type AttachmentExtensions struct {
	MediaType string `json:"mediaType,omitempty"`
	FileID    string `json:"fileId,omitempty"`
}

// GetAttachments retrieves all attachments for a page.
func (c *Client) GetAttachments(ctx context.Context, pageID string) ([]AttachmentInfo, error) {
	var allAttachments []AttachmentInfo
	start := 0
	limit := 100

	for {
		params := map[string]string{
			"limit":  fmt.Sprintf("%d", limit),
			"start":  fmt.Sprintf("%d", start),
			"expand": "version",
		}

		body, err := c.do(ctx, "GET", fmt.Sprintf("/rest/api/content/%s/child/attachment", pageID), params)
		if err != nil {
			return allAttachments, fmt.Errorf("failed to get attachments for page %s: %w", pageID, err)
		}

		var resp struct {
			Results []AttachmentInfo `json:"results"`
			Size    int             `json:"size"`
		}

		if err := json.Unmarshal(body, &resp); err != nil {
			return allAttachments, fmt.Errorf("failed to parse attachments: %w", err)
		}

		allAttachments = append(allAttachments, resp.Results...)

		if resp.Size < limit {
			break
		}
		start += resp.Size
	}

	return allAttachments, nil
}

// DownloadAttachment downloads an attachment's raw content.
func (c *Client) DownloadAttachment(ctx context.Context, downloadURL string) ([]byte, error) {
	fullURL := c.BaseURL + downloadURL

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	// Apply auth headers
	if c.basicUser != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	} else if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download body: %w", err)
	}

	return data, nil
}
