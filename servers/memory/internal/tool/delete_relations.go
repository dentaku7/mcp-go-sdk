package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/delete_relations.json
var deleteRelationsSchemaJSON []byte // Use []byte for json.RawMessage

// DeleteRelationsTool implements the Tool interface for deleting relations
type DeleteRelationsTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewDeleteRelationsTool creates a new DeleteRelationsTool instance
func NewDeleteRelationsTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &DeleteRelationsTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *DeleteRelationsTool) Name() string {
	return "delete_relations"
}

// Description returns the description of the tool
func (t *DeleteRelationsTool) Description() string {
	return "Deletes relations from the knowledge graph"
}

// Schema returns the JSON schema for the tool's parameters
func (t *DeleteRelationsTool) Schema() json.RawMessage {
	return deleteRelationsSchemaJSON
}

// Execute deletes relations from the knowledge graph
func (t *DeleteRelationsTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		Relations []struct {
			From         string `json:"from"`
			To           string `json:"to"`
			RelationType string `json:"relationType"`
		} `json:"relations"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Get current graph state to find relation IDs
	graph := t.manager.ReadGraph()
	relationsToDelete := make([]types.Relation, 0, len(input.Relations))
	deletedCount := 0

	// For each relation to delete, try to find its ID in the graph
	for _, rel := range input.Relations {
		// Find all matching relations
		for id, existingRel := range graph.Relations {
			if existingRel.Source == rel.From &&
				existingRel.Target == rel.To &&
				existingRel.Type == rel.RelationType {
				// Found matching relation, use its ID
				relationsToDelete = append(relationsToDelete, types.Relation{
					ID:     id,
					Type:   rel.RelationType,
					Source: rel.From,
					Target: rel.To,
				})
				deletedCount++
			}
		}
	}

	if err := t.manager.DeleteRelations(relationsToDelete); err != nil {
		return nil, fmt.Errorf("failed to delete relations: %w", err)
	}

	return formatResponse(
		fmt.Sprintf("Successfully deleted %d relations", deletedCount),
		map[string]interface{}{
			"deleted_count": deletedCount,
			"deleted_items": relationsToDelete,
		},
		nil,
	), nil
}
