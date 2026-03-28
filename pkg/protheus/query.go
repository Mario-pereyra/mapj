package protheus

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
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

func ValidateReadOnly(sqlQuery string) error {
	normalized := strings.ToUpper(strings.TrimSpace(sqlQuery))

	normalized = regexp.MustCompile(`--.*$`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(normalized, "")

	dangerous := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
		"TRUNCATE", "EXEC", "EXECUTE", "MERGE", "INTO", "REPLACE",
		"GRANT", "REVOKE", "DENY", "BACKUP", "RESTORE",
	}

	for _, keyword := range dangerous {
		pattern := fmt.Sprintf(`\b%s\b`, keyword)
		if matched, _ := regexp.MatchString(pattern, normalized); matched {
			return fmt.Errorf("query contains forbidden keyword: %s", keyword)
		}
	}

	selectPattern := `^\s*(SELECT|WITH\s+)`
	if matched, _ := regexp.MatchString(selectPattern, normalized); !matched {
		return fmt.Errorf("query must be a SELECT statement")
	}

	return nil
}

func (c *Client) Query(ctx context.Context, sqlQuery string) (*QueryResult, error) {
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

	for rows.Next() {
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
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    results,
		Count:   len(results),
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
