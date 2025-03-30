package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/update_entities.json
var updateEntitiesSchemaJSON []byte // Use []byte for json.RawMessage

// UpdateEntitiesTool implements the Tool interface for partially updating entities in batch
type UpdateEntitiesTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewUpdateEntitiesTool creates a new UpdateEntitiesTool instance
func NewUpdateEntitiesTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &UpdateEntitiesTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *UpdateEntitiesTool) Name() string {
	return "update_entities"
}

// Description returns the description of the tool
func (t *UpdateEntitiesTool) Description() string {
	return "Performs partial updates on existing entities in the knowledge graph."

}

// Schema returns the JSON schema for the tool's parameters
func (t *UpdateEntitiesTool) Schema() json.RawMessage {
	return updateEntitiesSchemaJSON
}

// Execute performs partial updates on entities in batch
func (t *UpdateEntitiesTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		Entities []types.Entity `json:"entities"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	if len(input.Entities) == 0 {
		return formatError(fmt.Errorf("entities array cannot be empty")), nil
	}

	// Basic validation: ensure all entities have an ID before calling manager
	for i, entity := range input.Entities {
		if entity.ID == "" {
			return formatError(fmt.Errorf("entity at index %d is missing the required 'id' field", i)), nil
		}
	}

	updatedEntities, err := t.manager.UpdateEntities(input.Entities)
	if err != nil {
		// Error already includes specifics from the manager
		return formatError(fmt.Errorf("failed to update entities: %w", err)), nil
	}

	return formatResponse(
		fmt.Sprintf("Successfully updated %d entities", len(updatedEntities)),
		map[string]interface{}{
			"input_count":   len(input.Entities),
			"updated_count": len(updatedEntities),
			"updated_items": updatedEntities,
		},
		updatedEntities,
	), nil
}
