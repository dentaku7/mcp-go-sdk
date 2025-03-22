package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
)

// DuckDBTool implements the MCP tool interface for DuckDB
type DuckDBTool struct {
	db     *sql.DB
	dbPath string
	mu     sync.RWMutex
}

// NewDuckDBTool creates a new DuckDB tool instance
func NewDuckDBTool(dbPath string) *DuckDBTool {
	return &DuckDBTool{
		dbPath: dbPath,
	}
}

// Name returns the tool name
func (t *DuckDBTool) Name() string {
	return "duckdb"
}

// Description returns the tool description
func (t *DuckDBTool) Description() string {
	return "Execute SQL queries against a DuckDB database"
}

// Schema returns the JSON schema for the tool's parameters
func (t *DuckDBTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The command to execute (query, explain, status)"
			},
			"query": {
				"type": "string",
				"description": "The SQL query to execute (required for query and explain commands)"
			}
		},
		"required": ["command"]
	}`)
}

// Execute handles the tool execution
func (t *DuckDBTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		Command string `json:"command"`
		Query   string `json:"query,omitempty"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	// Ensure we have a connection
	if err := t.ensureConnection(); err != nil {
		return nil, err
	}

	switch input.Command {
	case "query":
		return t.handleQuery(input.Query)
	case "explain":
		return t.handleExplain(input.Query)
	case "status":
		return t.handleStatus()
	default:
		return nil, fmt.Errorf("unknown command: %s", input.Command)
	}
}

// ensureConnection ensures we have a valid database connection
func (t *DuckDBTool) ensureConnection() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.db != nil {
		return nil
	}

	db, err := sql.Open("duckdb", t.dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %v", err)
	}

	t.db = db
	log.Printf("Connected to DuckDB database: %s", t.dbPath)
	return nil
}

// handleQuery executes a SQL query
func (t *DuckDBTool) handleQuery(query string) (interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if query == "" {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Error: Query cannot be empty",
				},
			},
			"metadata": map[string]interface{}{
				"status": "error",
				"error":  "Query cannot be empty",
			},
		}, nil
	}

	start := time.Now()
	rows, err := t.db.Query(query)
	if err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Error executing query:\n%s\n\nDuckDB Error:\n%v", query, err),
				},
			},
			"metadata": map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
				"query":  query,
			},
		}, nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Error getting columns:\n%v", err),
				},
			},
			"metadata": map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			},
		}, nil
	}

	var resultRows [][]interface{}
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("Error scanning row:\n%v", err),
					},
				},
				"metadata": map[string]interface{}{
					"status": "error",
					"error":  err.Error(),
				},
			}, nil
		}

		row := make([]interface{}, len(columns))
		for i, v := range values {
			row[i] = v
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Error during row iteration:\n%v", err),
				},
			},
			"metadata": map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			},
		}, nil
	}

	duration := time.Since(start)

	// Format the table as text
	var formattedText string

	// Add headers
	for i, col := range columns {
		if i > 0 {
			formattedText += " | "
		}
		formattedText += fmt.Sprintf("%-15s", col)
	}
	formattedText += "\n"

	// Add separator
	for i := range columns {
		if i > 0 {
			formattedText += "-|-"
		}
		formattedText += "---------------"
	}
	formattedText += "\n"

	// Add rows
	for _, row := range resultRows {
		for i, val := range row {
			if i > 0 {
				formattedText += " | "
			}
			formattedText += fmt.Sprintf("%-15v", val)
		}
		formattedText += "\n"
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": formattedText,
			},
		},
		"metadata": map[string]interface{}{
			"duration": duration.Seconds(),
			"rowCount": len(resultRows),
			"status":   "success",
			"query":    query,
		},
	}, nil
}

// handleExplain shows the query execution plan
func (t *DuckDBTool) handleExplain(query string) (interface{}, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	return t.handleQuery(fmt.Sprintf("EXPLAIN %s", query))
}

// handleStatus returns the current connection status
func (t *DuckDBTool) handleStatus() (interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status := "Not connected"
	if t.db != nil {
		status = fmt.Sprintf("Connected to: %s", t.dbPath)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": status,
			},
		},
	}, nil
}

// Close closes the database connection
func (t *DuckDBTool) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.db != nil {
		if err := t.db.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %v", err)
		}
		t.db = nil
	}
	return nil
}

// GetVersion returns the DuckDB version
func (t *DuckDBTool) GetVersion() (string, error) {
	if err := t.ensureConnection(); err != nil {
		return "", err
	}

	var version string
	err := t.db.QueryRow("SELECT version();").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("failed to get DuckDB version: %v", err)
	}
	return version, nil
}
