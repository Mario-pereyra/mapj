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
	}

	for _, query := range validQueries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.NoError(t, err, "Expected %q to be valid SELECT", query)
		})
	}
}

func TestValidateReadOnly_Insert(t *testing.T) {
	queries := []string{
		"INSERT INTO table VALUES (1, 2)",
		"INSERT INTO table (col) VALUES ('value')",
		"insert into table select * from other",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "INSERT")
		})
	}
}

func TestValidateReadOnly_Update(t *testing.T) {
	queries := []string{
		"UPDATE table SET col = 1",
		"UPDATE table SET col = 1 WHERE id = 2",
		"update table set col = 1",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "UPDATE")
		})
	}
}

func TestValidateReadOnly_Delete(t *testing.T) {
	queries := []string{
		"DELETE FROM table",
		"DELETE FROM table WHERE id = 1",
		"delete from table",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "DELETE")
		})
	}
}

func TestValidateReadOnly_DangerousKeywords(t *testing.T) {
	dangerousQueries := []struct {
		query   string
		keyword string
	}{
		{"DROP TABLE users", "DROP"},
		{"ALTER TABLE add column", "ALTER"},
		{"CREATE TABLE newtable", "CREATE"},
		{"TRUNCATE TABLE users", "TRUNCATE"},
		{"EXEC sp_name", "EXEC"},
		{"EXECUTE sp_name", "EXECUTE"},
		{"MERGE INTO target", "MERGE"},
		{"GRANT SELECT ON table", "GRANT"},
		{"REVOKE SELECT ON table", "REVOKE"},
	}

	for _, q := range dangerousQueries {
		t.Run(q.query, func(t *testing.T) {
			err := ValidateReadOnly(q.query)
			assert.Error(t, err, "Expected %q to be rejected", q.query)
			assert.Contains(t, err.Error(), q.keyword)
		})
	}
}

func TestValidateReadOnly_SQLComments(t *testing.T) {
	queries := []string{
		"SELECT * FROM table -- this is a comment",
		"SELECT * FROM table /* block comment */",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.NoError(t, err, "Comments should be stripped")
		})
	}

	malicious := "SELECT * FROM table; DROP TABLE users; -- comment"
	err := ValidateReadOnly(malicious)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DROP")
}

func TestValidateReadOnly_NotSelect(t *testing.T) {
	invalidQueries := []string{
		"SHOW TABLES",
		"EXPLAIN SELECT * FROM table",
		"DESCRIBE table",
	}

	for _, query := range invalidQueries {
		t.Run(query, func(t *testing.T) {
			err := ValidateReadOnly(query)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "must be a SELECT")
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
