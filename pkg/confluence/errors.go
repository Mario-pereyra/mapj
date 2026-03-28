package confluence

import "fmt"

// Error codes for structured error logging.
const (
	ErrHTTP403        = "HTTP_403"
	ErrHTTP404        = "HTTP_404"
	ErrHTTP429        = "HTTP_429"
	ErrHTTPTimeout    = "HTTP_TIMEOUT"
	ErrAuthFailed     = "AUTH_FAILED"
	ErrPathTooLong    = "PATH_TOO_LONG"
	ErrWritePermission = "WRITE_PERMISSION"
	ErrUnknownMacro   = "UNKNOWN_MACRO"
	ErrConvertPanic   = "CONVERT_PANIC"
	ErrAttachmentFail = "ATTACHMENT_FAIL"
	ErrPageNotFound   = "PAGE_NOT_FOUND"
	ErrParseFailed    = "PARSE_FAILED"
)

// Error phases identify where in the pipeline an error occurred.
const (
	PhaseAPIFetch   = "api_fetch"
	PhaseConvert    = "convert"
	PhaseWrite      = "write"
	PhaseAttachment = "attachment"
)

// ExportError is a structured error for the export pipeline.
type ExportError struct {
	PageID    string `json:"page_id"`
	Title     string `json:"title,omitempty"`
	Phase     string `json:"phase"`
	Code      string `json:"error_code"`
	Message   string `json:"message"`
	SourceURL string `json:"source_url,omitempty"`
	RetryCmd  string `json:"retry_cmd,omitempty"`

	// Optional diagnostic fields
	HTTPStatus    int    `json:"http_status,omitempty"`
	GeneratedPath string `json:"generated_path,omitempty"`
	MacroName     string `json:"macro_name,omitempty"`
	HTMLSnippet   string `json:"html_snippet,omitempty"`
}

func (e *ExportError) Error() string {
	return fmt.Sprintf("[%s] %s (page=%s, phase=%s): %s", e.Code, e.Title, e.PageID, e.Phase, e.Message)
}

// NewExportError creates a structured export error with a retry command.
func NewExportError(pageID, title, phase, code, message, outputPath string) *ExportError {
	return &ExportError{
		PageID:   pageID,
		Title:    title,
		Phase:    phase,
		Code:     code,
		Message:  message,
		RetryCmd: fmt.Sprintf("mapj confluence export %s --output-path %s", pageID, outputPath),
	}
}

// HTTPError classifies an HTTP status code into a structured error code.
func HTTPErrorCode(status int) string {
	switch {
	case status == 401 || status == 403:
		return ErrHTTP403
	case status == 404:
		return ErrHTTP404
	case status == 429:
		return ErrHTTP429
	default:
		return fmt.Sprintf("HTTP_%d", status)
	}
}
