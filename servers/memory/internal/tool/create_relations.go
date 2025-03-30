package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/create_relations.json
var createRelationsSchemaJSON []byte // Use []byte for json.RawMessage

// CreateRelationsTool implements the Tool interface for creating relations
type CreateRelationsTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewCreateRelationsTool creates a new CreateRelationsTool instance
func NewCreateRelationsTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &CreateRelationsTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *CreateRelationsTool) Name() string {
	return "create_relations"
}

// Description returns the description of the tool
func (t *CreateRelationsTool) Description() string {
	return "Creates new relations between entities in the knowledge graph"
}

// Schema returns the JSON schema for the tool's parameters
func (t *CreateRelationsTool) Schema() json.RawMessage {
	return createRelationsSchemaJSON
}

// Execute creates new relations in the knowledge graph
func (t *CreateRelationsTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		Relations []types.Relation `json:"relations"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	createdRelations, err := t.manager.CreateRelations(input.Relations)
	if err != nil {
		return formatError(fmt.Errorf("failed to create relations: %w", err)), nil
	}

	return formatResponse(
		fmt.Sprintf("Successfully created %d relations", len(createdRelations)),
		map[string]interface{}{
			"input_count":   len(input.Relations),
			"created_count": len(createdRelations),
			"created_items": createdRelations,
		},
		createdRelations,
	), nil
}
