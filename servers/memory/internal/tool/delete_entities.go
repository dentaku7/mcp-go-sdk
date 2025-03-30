package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
)

//go:embed schemas/delete_entities.json
var deleteEntitiesSchemaJSON []byte // Use []byte for json.RawMessage

// DeleteEntitiesTool implements the Tool interface for deleting entities
type DeleteEntitiesTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewDeleteEntitiesTool creates a new DeleteEntitiesTool instance
func NewDeleteEntitiesTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &DeleteEntitiesTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *DeleteEntitiesTool) Name() string {
	return "delete_entities"
}

// Description returns the description of the tool
func (t *DeleteEntitiesTool) Description() string {
	return "Deletes entities from the knowledge graph"
}

// Schema returns the JSON schema for the tool's parameters
func (t *DeleteEntitiesTool) Schema() json.RawMessage {
	return deleteEntitiesSchemaJSON
}

// Execute deletes entities from the knowledge graph
func (t *DeleteEntitiesTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		EntityNames []string `json:"entityNames"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	if err := t.manager.DeleteEntities(input.EntityNames); err != nil {
		return formatError(fmt.Errorf("failed to delete entities: %w", err)), nil
	}

	return formatResponse(
		fmt.Sprintf("Successfully deleted %d entities", len(input.EntityNames)),
		map[string]interface{}{
			"deleted_count": len(input.EntityNames),
			"deleted_ids":   input.EntityNames,
		},
		nil,
	), nil
}
