package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-memory/internal/types"
)

//go:embed schemas/open_nodes.json
var openNodesSchemaJSON []byte // Use []byte for json.RawMessage

// GraphManager defines the interface for graph operations
type GraphManager interface {
	OpenNodes(nodeIDs []string) (*types.KnowledgeGraphResult, error)
}

// OpenNodesTool implements the Tool interface for opening nodes
type OpenNodesTool struct {
	manager GraphManager
}

// NewOpenNodesTool creates a new OpenNodesTool instance
func NewOpenNodesTool(manager GraphManager) *OpenNodesTool {
	return &OpenNodesTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *OpenNodesTool) Name() string {
	return "open_nodes"
}

// Description returns the description of the tool
func (t *OpenNodesTool) Description() string {
	return "Returns all entities connected to the given entity IDs"
}

// Schema returns the JSON schema for the tool's parameters
func (t *OpenNodesTool) Schema() json.RawMessage {
	return openNodesSchemaJSON
}

type OpenNodesInput struct {
	NodeIDs []string `json:"node_ids"`
}

// Execute returns all entities connected to the given entity IDs
func (t *OpenNodesTool) Execute(input json.RawMessage) (interface{}, error) {
	var params OpenNodesInput
	if err := json.Unmarshal(input, &params); err != nil {
		return formatError(fmt.Errorf("failed to unmarshal input: %w", err)), nil
	}

	result, err := t.manager.OpenNodes(params.NodeIDs)
	if err != nil {
		return formatError(fmt.Errorf("failed to open nodes: %w", err)), nil
	}

	return formatResponse(
		fmt.Sprintf("Found %d entities and %d relations", len(result.Entities), len(result.Relations)),
		map[string]interface{}{
			"entity_count":   len(result.Entities),
			"relation_count": len(result.Relations),
			"input_ids":      params.NodeIDs,
		},
		result,
	), nil
}
