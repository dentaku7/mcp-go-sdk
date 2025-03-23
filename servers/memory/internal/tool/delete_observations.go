package tool

import (
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
)

// DeleteObservationsTool implements the Tool interface for deleting observations
type DeleteObservationsTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewDeleteObservationsTool creates a new DeleteObservationsTool instance
func NewDeleteObservationsTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &DeleteObservationsTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *DeleteObservationsTool) Name() string {
	return "delete_observations"
}

// Description returns the description of the tool
func (t *DeleteObservationsTool) Description() string {
	return "Deletes observations from the knowledge graph"
}

// Schema returns the JSON schema for the tool's parameters
func (t *DeleteObservationsTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"ids": {
				"type": "array",
				"items": {
					"type": "string"
				}
			}
		},
		"required": ["ids"]
	}`)
}

// Execute deletes observations from the knowledge graph
func (t *DeleteObservationsTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		IDs []string `json:"ids"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	if err := t.manager.DeleteObservations(input.IDs); err != nil {
		return formatError(fmt.Errorf("failed to delete observations: %w", err)), nil
	}

	return formatResponse(
		fmt.Sprintf("Successfully deleted %d observations", len(input.IDs)),
		map[string]interface{}{
			"deleted_count": len(input.IDs),
			"deleted_ids":   input.IDs,
		},
		nil,
	), nil
}
