package protheus

import (
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
	assert.Contains(t, err.Error(), "query must start with SELECT, WITH, or EXEC")
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
