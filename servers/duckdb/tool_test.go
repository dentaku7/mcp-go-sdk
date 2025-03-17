package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	// Test create table and query
	createTableQuery := map[string]interface{}{
		"command": "query",
		"query":   "CREATE TABLE test (id INTEGER, name TEXT)",
	}
	queryJSON, _ := json.Marshal(createTableQuery)
	result, err := tool.Execute(queryJSON)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Test insert data
	insertQuery := map[string]interface{}{
		"command": "query",
		"query":   "INSERT INTO test VALUES (1, 'test1'), (2, 'test2')",
	}
	queryJSON, _ = json.Marshal(insertQuery)
	result, err = tool.Execute(queryJSON)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Test select data
	selectQuery := map[string]interface{}{
		"command": "query",
		"query":   "SELECT * FROM test ORDER BY id",
	}
	queryJSON, _ = json.Marshal(selectQuery)
	result, err = tool.Execute(queryJSON)
	if err != nil {
		t.Fatalf("Failed to select data: %v", err)
	}

	// Verify result structure
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	content, ok := resultMap["content"].([]map[string]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("Result content is invalid")
	}

	textResult := content[0]
	if textResult["type"] != "text" {
		t.Error("Expected text type result")
	}

	text, ok := textResult["text"].(string)
	if !ok || text == "" {
		t.Error("Expected non-empty text result")
	}

	// Verify metadata
	metadata, ok := resultMap["metadata"].(map[string]interface{})
	if !ok {
		t.Error("Expected metadata in result")
	}

	if _, ok := metadata["duration"].(float64); !ok {
		t.Error("Expected duration in metadata")
	}

	if _, ok := metadata["rowCount"].(int); !ok {
		t.Error("Expected rowCount in metadata")
	}

	if status, ok := metadata["status"].(string); !ok || status != "success" {
		t.Error("Expected success status in metadata")
	}

	// Test explain
	explainQuery := map[string]interface{}{
		"command": "explain",
		"query":   "SELECT * FROM test",
	}
	queryJSON, _ = json.Marshal(explainQuery)
	result, err = tool.Execute(queryJSON)
	if err != nil {
		t.Fatalf("Failed to explain query: %v", err)
	}

	// Test status
	statusQuery := map[string]interface{}{
		"command": "status",
	}
	queryJSON, _ = json.Marshal(statusQuery)
	result, err = tool.Execute(queryJSON)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	// Test invalid command
	invalidQuery := map[string]interface{}{
		"command": "invalid",
	}
	queryJSON, _ = json.Marshal(invalidQuery)
	if _, err := tool.Execute(queryJSON); err == nil {
		t.Error("Expected error for invalid command")
	}
}
