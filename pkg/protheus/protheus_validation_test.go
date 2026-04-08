package protheus

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateReadOnly_Select(t *testing.T) {
	validQueries := []string{
		"SELECT * FROM table",
		"SELECT col1, col2 FROM table WHERE id = 1",
		"select count(*) from table",
		"SELECT TOP 10 * FROM SPED050",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"  SELECT * FROM table  ",
		"EXEC sp_help table",
		"exec sp_help table",
	}

	for _, query := range validQueries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.NoError(t, err, "Expected %q to be valid SELECT/WITH/EXEC", query)
		})
	}
}

func TestValidateReadOnly_PrefixRejection(t *testing.T) {
	invalidQueries := []string{
		"INSERT INTO table VALUES (1, 2)",
		"UPDATE table SET col = 1",
		"DELETE FROM table",
		"DROP TABLE users",
		"ALTER TABLE add column",
		"CREATE TABLE newtable",
		"TRUNCATE TABLE users",
		"MERGE INTO target",
		"GRANT SELECT ON table",
		"SHOW TABLES",
		"EXPLAIN SELECT * FROM table",
		"DESCRIBE table",
	}

	for _, query := range invalidQueries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.Error(t, err, "Expected %q to be rejected", query)
			assert.Contains(t, err.Error(), "query must start with SELECT, WITH, or EXEC")
		})
	}
}

func TestValidateReadOnly_SQLComments(t *testing.T) {
	queries := []string{
		"-- this is a comment\nSELECT * FROM table",
		"/* block comment */ SELECT * FROM table",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.NoError(t, err, "Comments should be stripped")
		})
	}

	malicious := "-- comment\nDROP TABLE users;"
	err := ValidateReadOnly(malicious)
	assert.Error(t, err)
	// Error can be about forbidden keyword (DROP), semicolon, or prefix check
	errMsg := err.Error()
	assert.True(t, strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "semicolon") || strings.Contains(errMsg, "query must start with SELECT, WITH, or EXEC"),
		"Error should mention forbidden keyword, semicolon, or prefix requirement, got: %s", errMsg)
}

func TestValidateReadOnly_SQLInjection(t *testing.T) {
	// These are SQL injection attempts that start with valid SELECT
	// but contain malicious statements after semicolon or inline
	injectionQueries := []string{
		"SELECT * FROM users; DROP TABLE users;",
		"SELECT * FROM users; DELETE FROM users;",
		"SELECT * FROM users; INSERT INTO users VALUES (1, 'hacker');",
		"SELECT * FROM users; UPDATE users SET admin = 1;",
		"SELECT * FROM users; TRUNCATE TABLE users;",
		"SELECT * FROM users; ALTER TABLE users ADD column hack;",
		"SELECT * FROM users; CREATE TABLE hack (id int);",
		"SELECT * FROM users; GRANT ALL ON users TO hacker;",
		"SELECT * FROM users; REVOKE ALL ON users FROM admin;",
		"SELECT * FROM users; EXEC sp_hack;",
		"SELECT * FROM users; EXECUTE sp_hack;",
		"SELECT * FROM users; MERGE INTO users USING hack;",
		"SELECT * FROM users; BACKUP DATABASE x;",
		"SELECT * FROM users; RESTORE DATABASE x;",
		"SELECT * FROM users; DENY SELECT ON users TO admin;",
		// Inline dangerous keywords
		"SELECT * FROM (DELETE FROM users) AS x",
		"SELECT * FROM users WHERE id = 1 OR DROP TABLE users",
	}

	for _, query := range injectionQueries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.Error(t, err, "Expected SQL injection %q to be rejected", query)
			// Error should mention forbidden keyword or semicolon
			errMsg := err.Error()
			assert.True(t, strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "semicolon"),
				"Error should mention forbidden keyword or semicolon, got: %s", errMsg)
		})
	}
}

func TestValidateReadOnly_SemicolonAsSeparator(t *testing.T) {
	// Semicolon as statement separator should be detected
	queries := []string{
		"SELECT 1; SELECT 2",
		"SELECT * FROM users; -- comment",
		"WITH cte AS (SELECT 1) SELECT * FROM cte; SELECT 2",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.Error(t, err, "Query with semicolon should be rejected")
			assert.Contains(t, err.Error(), "semicolon", "Error should mention semicolon")
		})
	}
}

func TestValidateReadOnly_LegitimateExec(t *testing.T) {
	// Valid EXEC calls should still be accepted (but not with semicolon)
	validExecs := []string{
		"EXEC sp_help 'table'",
		"EXEC sp_columns 'users'",
		"EXECUTE sp_helpfile",
		"exec dbo.sp_test",
	}

	for _, query := range validExecs {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.NoError(t, err, "Legitimate EXEC should be accepted")
		})
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("localhost", 1433, "mydb", "user", "pass")

	assert.Equal(t, "localhost", client.Server)
	assert.Equal(t, 1433, client.Port)
	assert.Equal(t, "mydb", client.Database)
	assert.Equal(t, "user", client.User)
	assert.Equal(t, "pass", client.Password)
}

func TestClient_ConnectionString(t *testing.T) {
	client := NewClient("192.168.1.100", 1433, "PROTHEUS", "admin", "secret")

	connStr := client.connectionString()

	assert.Contains(t, connStr, "server=192.168.1.100")
	assert.Contains(t, connStr, "port=1433")
	assert.Contains(t, connStr, "database=PROTHEUS")
	assert.Contains(t, connStr, "user id=admin")
	assert.Contains(t, connStr, "password=secret")
	assert.Contains(t, connStr, "encrypt=disable")
}
