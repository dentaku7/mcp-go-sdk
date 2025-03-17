package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDuckDBTool(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "duckdb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Let DuckDB create the database file
	dbPath := filepath.Join(tmpDir, "test.duckdb")

	// Create tool instance
	tool := NewDuckDBTool(dbPath)
	defer tool.Close()

	// Test tool name and description
	if name := tool.Name(); name != "duckdb" {
		t.Errorf("Expected tool name 'duckdb', got '%s'", name)
	}

	if desc := tool.Description(); desc == "" {
		t.Error("Tool description should not be empty")
	}

	// Test schema
	schema := tool.Schema()
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Test error cases
	testCases := []struct {
		name    string
		command string
		query   string
		wantErr bool
		errText string
		checkFn func(t *testing.T, result map[string]interface{})
	}{
		{
			name:    "Empty query",
			command: "query",
			query:   "",
			wantErr: true,
			errText: "Query cannot be empty",
			checkFn: func(t *testing.T, result map[string]interface{}) {
				metadata := result["metadata"].(map[string]interface{})
				if metadata["status"] != "error" {
					t.Error("Expected error status")
				}
			},
		},
		{
			name:    "Invalid SQL syntax",
			command: "query",
			query:   "SELECT * FROMM nonexistent",
			wantErr: true,
			errText: "syntax error",
			checkFn: func(t *testing.T, result map[string]interface{}) {
				metadata := result["metadata"].(map[string]interface{})
				if metadata["status"] != "error" {
					t.Error("Expected error status")
				}
				if !strings.Contains(metadata["error"].(string), "syntax error") {
					t.Error("Expected syntax error in error message")
				}
			},
		},
		{
			name:    "Query nonexistent table",
			command: "query",
			query:   "SELECT * FROM nonexistent_table",
			wantErr: true,
			errText: "Table",
			checkFn: func(t *testing.T, result map[string]interface{}) {
				metadata := result["metadata"].(map[string]interface{})
				if metadata["status"] != "error" {
					t.Error("Expected error status")
				}
				if !strings.Contains(metadata["error"].(string), "Table") {
					t.Error("Expected table not found error")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]interface{}{
				"command": tc.command,
				"query":   tc.query,
			}
			queryJSON, _ := json.Marshal(input)
			result, err := tool.Execute(queryJSON)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatal("Result is not a map")
			}

			content, ok := resultMap["content"].([]map[string]interface{})
			if !ok || len(content) == 0 {
				t.Fatal("Result content is invalid")
			}

			text := content[0]["text"].(string)
			if tc.wantErr && !strings.Contains(text, tc.errText) {
				t.Errorf("Expected error text containing %q, got %q", tc.errText, text)
			}

			if tc.checkFn != nil {
				tc.checkFn(t, resultMap)
			}
		})
	}

	// Test successful cases
	t.Run("Successful operations", func(t *testing.T) {
		// Create table
		createTableQuery := map[string]interface{}{
			"command": "query",
			"query":   "CREATE TABLE test (id INTEGER, name TEXT)",
		}
		queryJSON, _ := json.Marshal(createTableQuery)
		result, err := tool.Execute(queryJSON)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Insert data
		insertQuery := map[string]interface{}{
			"command": "query",
			"query":   "INSERT INTO test VALUES (1, 'test1'), (2, 'test2')",
		}
		queryJSON, _ = json.Marshal(insertQuery)
		result, err = tool.Execute(queryJSON)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}

		// Select data
		selectQuery := map[string]interface{}{
			"command": "query",
			"query":   "SELECT * FROM test ORDER BY id",
		}
		queryJSON, _ = json.Marshal(selectQuery)
		result, err = tool.Execute(queryJSON)
		if err != nil {
			t.Fatalf("Failed to select data: %v", err)
		}

		resultMap := result.(map[string]interface{})
		metadata := resultMap["metadata"].(map[string]interface{})

		if metadata["status"] != "success" {
			t.Error("Expected success status")
		}

		if rowCount, ok := metadata["rowCount"].(int); !ok || rowCount != 2 {
			t.Error("Expected 2 rows in result")
		}

		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)
		if !strings.Contains(text, "test1") || !strings.Contains(text, "test2") {
			t.Error("Expected result to contain test data")
		}
	})

	// Test explain
	t.Run("Explain query", func(t *testing.T) {
		explainQuery := map[string]interface{}{
			"command": "explain",
			"query":   "SELECT * FROM test",
		}
		queryJSON, _ := json.Marshal(explainQuery)
		result, err := tool.Execute(queryJSON)
		if err != nil {
			t.Fatalf("Failed to explain query: %v", err)
		}

		resultMap := result.(map[string]interface{})
		metadata := resultMap["metadata"].(map[string]interface{})
		if metadata["status"] != "success" {
			t.Error("Expected success status for explain")
		}
	})

	// Test status
	t.Run("Status check", func(t *testing.T) {
		statusQuery := map[string]interface{}{
			"command": "status",
		}
		queryJSON, _ := json.Marshal(statusQuery)
		result, err := tool.Execute(queryJSON)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)
		if !strings.Contains(text, "Connected to:") {
			t.Error("Expected status to show connection info")
		}
	})

	// Test invalid command
	t.Run("Invalid command", func(t *testing.T) {
		invalidQuery := map[string]interface{}{
			"command": "invalid",
		}
		queryJSON, _ := json.Marshal(invalidQuery)
		_, err := tool.Execute(queryJSON)
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Error("Expected error for invalid command")
		}
	})
}
