package tool

import (
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
)

// SearchNodesTool implements the Tool interface for searching nodes
type SearchNodesTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewSearchNodesTool creates a new SearchNodesTool instance
func NewSearchNodesTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &SearchNodesTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *SearchNodesTool) Name() string {
	return "search_nodes"
}

// Description returns the description of the tool
func (t *SearchNodesTool) Description() string {
	return "Searches for entities based on type and metadata"
}

// Schema returns the JSON schema for the tool's parameters
func (t *SearchNodesTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"type": {
				"type": "string"
			},
			"metadata": {
				"type": "object",
				"additionalProperties": true
			}
		}
	}`)
}

// Execute searches for entities based on type and metadata
func (t *SearchNodesTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		Type     string                 `json:"type"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	results := t.manager.SearchNodes(input.Type, input.Metadata)

	return formatResponse(
		fmt.Sprintf("Found %d matching entities", len(results)),
		map[string]interface{}{
			"result_count": len(results),
			"search_type":  input.Type,
		},
		results,
	), nil
}
