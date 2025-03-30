package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"
	"strings"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/bulk_update_metadata.json
var bulkUpdateMetadataSchemaJSON []byte // Use []byte for json.RawMessage

// BulkUpdateMetadataToolInput represents the input structure for the bulk_update_metadata tool.
type BulkUpdateMetadataToolInput struct {
	Filter    types.EntityFilterCriteria `json:"filter"` // Now required by schema, not a pointer
	Updates   map[string]interface{}     `json:"updates"`
	Operation string                     `json:"operation,omitempty"` // "merge", "replace", "delete". Defaults to "merge".
}

// BulkUpdateMetadataTool is a tool for updating metadata for multiple entities matching a filter.
type BulkUpdateMetadataTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewBulkUpdateMetadataTool creates a new instance of BulkUpdateMetadataTool
func NewBulkUpdateMetadataTool(manager *graph.KnowledgeGraphManager) *BulkUpdateMetadataTool {
	return &BulkUpdateMetadataTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *BulkUpdateMetadataTool) Name() string {
	return "bulk_update_metadata"
}

// Description returns the description of the tool
func (t *BulkUpdateMetadataTool) Description() string {
	return "Updates metadata for multiple entities matching filter criteria (type, name_contains, etc.). Supports 'merge' (default), 'replace', 'delete' operations and nested paths."
}

// Schema returns the JSON schema for the tool's parameters
func (t *BulkUpdateMetadataTool) Schema() json.RawMessage {
	return bulkUpdateMetadataSchemaJSON
}

// Execute performs the metadata update operation for multiple entities matching a filter
func (t *BulkUpdateMetadataTool) Execute(params json.RawMessage) (interface{}, error) {
	var input BulkUpdateMetadataToolInput
	if err := json.Unmarshal(params, &input); err != nil {
		// Basic validation happens via schema now, but keep unmarshal check
		return formatError(fmt.Errorf("failed to parse input parameters: %w", err)), nil
	}

	// Schema ensures Filter and Updates are present.
	// Runtime validation for filter content (at least one criterion)
	if input.Filter.Type == "" && input.Filter.NameContains == "" && input.Filter.DescriptionContains == "" {
		return formatError(fmt.Errorf("bulk update filter must contain at least one criterion (type, name_contains, description_contains)")), nil
	}

	// Validate the operation string.
	op := strings.ToLower(input.Operation)
	if op == "" {
		op = "merge" // Default operation
	} else if op != "merge" && op != "replace" && op != "delete" {
		return formatError(fmt.Errorf("invalid operation '%s'. Must be 'merge', 'replace', or 'delete'", input.Operation)), nil
	}

	// Execute the bulk update via the graph manager
	updatedEntities, err := t.manager.BulkUpdateMetadata(input.Filter, input.Updates, op)
	if err != nil {
		return formatError(fmt.Errorf("failed during bulk metadata update: %w", err)), nil
	}

	// Format the successful response for bulk updates
	message := fmt.Sprintf("Successfully applied '%s' operation to %d entities matching filter.", op, len(updatedEntities))
	metadata := map[string]interface{}{
		"filter":                 input.Filter,
		"operation":              op,
		"updated_entities_count": len(updatedEntities),
	}
	// Optionally include updated entity IDs or the entities themselves if not too large
	// For now, just return the count and message.
	return formatResponse(message, metadata, nil), nil // Data is nil for bulk summary
}

// Ensure BulkUpdateMetadataTool implements the Tool interface
var _ mcp.Tool = (*BulkUpdateMetadataTool)(nil)
