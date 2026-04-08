package protheus

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

type Client struct {
	Server   string
	Port     int
	Database string
	User     string
	Password string
}

type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`
}

func NewClient(server string, port int, database, user, password string) *Client {
	return &Client{
		Server:   server,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
	}
}

func (c *Client) connectionString() string {
	return fmt.Sprintf(
		"server=%s;port=%d;database=%s;user id=%s;password=%s;encrypt=disable",
		c.Server, c.Port, c.Database, c.User, c.Password,
	)
}

// forbiddenKeywords are SQL keywords that indicate write operations
// These must be detected anywhere in the query to prevent SQL injection
var forbiddenKeywords = []string{
	"INSERT",
	"UPDATE",
	"DELETE",
	"DROP",
	"ALTER",
	"CREATE",
	"TRUNCATE",
	"MERGE",
	"GRANT",
	"REVOKE",
	"DENY",
	"BACKUP",
	"RESTORE",
}

func ValidateReadOnly(sqlQuery string) error {
	normalized := strings.ToUpper(strings.TrimSpace(sqlQuery))

	// Remove leading multi-line comments for prefix checking
	for strings.HasPrefix(normalized, "/*") {
		endIdx := strings.Index(normalized, "*/")
		if endIdx == -1 {
			return fmt.Errorf("query contains unclosed multi-line comment")
		}
		normalized = strings.TrimSpace(normalized[endIdx+2:])
	}

	// Remove leading single-line comments for prefix checking
	for strings.HasPrefix(normalized, "--") {
		endIdx := strings.Index(normalized, "\n")
		if endIdx == -1 {
			return fmt.Errorf("query contains only comments")
		}
		normalized = strings.TrimSpace(normalized[endIdx+1:])
	}

	// Check for semicolon as statement separator (SQL injection pattern)
	// This prevents multiple statements like: SELECT * FROM users; DROP TABLE users;
	if strings.Contains(normalized, ";") {
		return fmt.Errorf("query contains semicolon: multiple statements not allowed")
	}

	// Check prefix for valid read-only keywords
	if !strings.HasPrefix(normalized, "SELECT") && !strings.HasPrefix(normalized, "WITH") && !strings.HasPrefix(normalized, "EXEC") && !strings.HasPrefix(normalized, "EXECUTE") {
		return fmt.Errorf("query must start with SELECT, WITH, or EXEC")
	}

	// Check for forbidden keywords anywhere in the query (SQL injection detection)
	// This catches patterns like: SELECT * FROM users WHERE id = 1 OR DROP TABLE users
	for _, keyword := range forbiddenKeywords {
		// Use word boundary matching to avoid false positives
		// e.g., "SELECT" should not match "SELECTED"
		if containsWord(normalized, keyword) {
			return fmt.Errorf("query contains forbidden keyword: %s", keyword)
		}
	}

	return nil
}

// containsWord checks if a string contains a keyword as a complete word
// This prevents false positives like "SELECTED" matching "SELECT"
func containsWord(s, word string) bool {
	// Find all occurrences of the word
	idx := strings.Index(s, word)
	for idx != -1 {
		// Check character before the match
		validBefore := idx == 0 || !isAlphaNum(rune(s[idx-1]))
		// Check character after the match
		validAfter := idx+len(word) >= len(s) || !isAlphaNum(rune(s[idx+len(word)]))

		if validBefore && validAfter {
			return true
		}
		// Look for next occurrence
		idx = strings.Index(s[idx+1:], word)
		if idx != -1 {
			idx = idx + len(word) // Adjust index relative to original string
		}
	}
	return false
}

// isAlphaNum checks if a character is alphanumeric
func isAlphaNum(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func (c *Client) Query(ctx context.Context, sqlQuery string, maxRows int) (*QueryResult, error) {
	if err := ValidateReadOnly(sqlQuery); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	db, err := sql.Open("mssql", c.connectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results [][]interface{}
	count := 0

	for rows.Next() {
		if maxRows > 0 && count >= maxRows {
			// Break the loop and let defer rows.Close() cleanly abort the cursor on the server
			break
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make([]interface{}, len(columns))
		for i, v := range values {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			default:
				row[i] = val
			}
		}
		results = append(results, row)
		count++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    results,
		Count:   count,
	}, nil
}

// Ping tests the database connection and returns the round-trip latency.
func (c *Client) Ping(ctx context.Context) (latencyMs int64, err error) {
	start := time.Now()

	db, err := sql.Open("mssql", c.connectionString())
	if err != nil {
		return 0, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return 0, fmt.Errorf("ping failed: %w", err)
	}

	return time.Since(start).Milliseconds(), nil
}
