package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"
	"strings"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	// "mcp-memory/internal/types" // Removed as it's unused in this specific tool file
)

//go:embed schemas/update_entity_metadata.json
var updateEntityMetadataSchemaJSON []byte // Use []byte for json.RawMessage

// UpdateEntityMetadataToolInput represents the input structure for the update_entity_metadata tool.
type UpdateEntityMetadataToolInput struct {
	EntityID  string                 `json:"entity_id"` // Now required by schema
	Updates   map[string]interface{} `json:"updates"`
	Operation string                 `json:"operation,omitempty"` // "merge", "replace", "delete". Defaults to "merge".
}

// UpdateEntityMetadataTool is a tool for updating a single entity's metadata.
type UpdateEntityMetadataTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewUpdateEntityMetadataTool creates a new instance of UpdateEntityMetadataTool
func NewUpdateEntityMetadataTool(manager *graph.KnowledgeGraphManager) *UpdateEntityMetadataTool {
	return &UpdateEntityMetadataTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *UpdateEntityMetadataTool) Name() string {
	return "update_entity_metadata"
}

// Description returns the description of the tool
func (t *UpdateEntityMetadataTool) Description() string {
	return "Updates metadata for a single entity specified by 'entity_id'. Supports 'merge' (default), 'replace', 'delete' operations and nested paths."
}

// Schema returns the JSON schema for the tool's parameters
func (t *UpdateEntityMetadataTool) Schema() json.RawMessage {
	return updateEntityMetadataSchemaJSON
}

// Execute performs the metadata update operation for a single entity
func (t *UpdateEntityMetadataTool) Execute(params json.RawMessage) (interface{}, error) {
	var input UpdateEntityMetadataToolInput
	if err := json.Unmarshal(params, &input); err != nil {
		// Basic validation happens via schema now, but keep unmarshal check
		return formatError(fmt.Errorf("failed to parse input parameters: %w", err)), nil
	}

	// Schema ensures EntityID and Updates are present.
	// We still need to validate the operation string.
	op := strings.ToLower(input.Operation)
	if op == "" {
		op = "merge" // Default operation
	} else if op != "merge" && op != "replace" && op != "delete" {
		return formatError(fmt.Errorf("invalid operation '%s'. Must be 'merge', 'replace', or 'delete'", input.Operation)), nil
	}

	// Execute single update via the graph manager
	updatedEntity, err := t.manager.UpdateMetadataForEntity(input.EntityID, input.Updates, op)
	if err != nil {
		return formatError(fmt.Errorf("failed to update metadata for entity '%s': %w", input.EntityID, err)), nil
	}

	// Format the successful response
	return formatResponse(
		fmt.Sprintf("Successfully updated metadata for entity '%s'", input.EntityID),
		map[string]interface{}{
			"entity_id": input.EntityID,
			"operation": op,
		},
		updatedEntity, // Include the updated entity as data
	), nil
}

// Ensure UpdateEntityMetadataTool implements the Tool interface
var _ mcp.Tool = (*UpdateEntityMetadataTool)(nil)
