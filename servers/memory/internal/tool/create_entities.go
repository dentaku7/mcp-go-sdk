package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/create_entities.json
var createEntitiesSchemaJSON []byte // Use []byte for json.RawMessage

// CreateEntitiesTool implements the Tool interface for creating entities
type CreateEntitiesTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewCreateEntitiesTool creates a new CreateEntitiesTool instance
func NewCreateEntitiesTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &CreateEntitiesTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *CreateEntitiesTool) Name() string {
	return "create_entities"
}

// Description returns the description of the tool
func (t *CreateEntitiesTool) Description() string {
	return `Creates new entities in the knowledge graph.`
}

// Schema returns the JSON schema for the tool's parameters
func (t *CreateEntitiesTool) Schema() json.RawMessage {
	return createEntitiesSchemaJSON
}

// Execute creates new entities in the knowledge graph
func (t *CreateEntitiesTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		Entities []types.Entity `json:"entities"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	createdEntities, err := t.manager.CreateEntities(input.Entities)
	if err != nil {
		return formatError(fmt.Errorf("failed to create entities: %w", err)), nil
	}

	return formatResponse(
		fmt.Sprintf("Successfully created %d entities", len(createdEntities)),
		map[string]interface{}{
			"input_count":   len(input.Entities),
			"created_count": len(createdEntities),
			"created_items": createdEntities,
		},
		createdEntities,
	), nil
}
