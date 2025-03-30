package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
)

//go:embed schemas/find_paths.json
var findPathsSchemaJSON []byte // Use []byte for json.RawMessage

// FindPathsTool implements the Tool interface for finding paths between nodes
type FindPathsTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewFindPathsTool creates a new FindPathsTool instance
func NewFindPathsTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &FindPathsTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *FindPathsTool) Name() string {
	return "find_paths"
}

// Description returns the description of the tool
func (t *FindPathsTool) Description() string {
	return "Finds all simple paths (no repeated nodes) between a start and end entity, up to a maximum length."
}

// Schema returns the JSON schema for the tool's parameters
func (t *FindPathsTool) Schema() json.RawMessage {
	return findPathsSchemaJSON
}

// Execute performs the path finding operation
func (t *FindPathsTool) Execute(params json.RawMessage) (interface{}, error) {
	// Define a temporary struct to unmarshal the full input including tool filters
	var input struct {
		StartNodeID string       `json:"start_node_id"`
		EndNodeID   string       `json:"end_node_id"`
		MaxLength   int          `json:"max_length"`
		Filters     *PathFilters `json:"filters"` // Use tool filter type
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	// Translate tool filters to internal graph filters
	var internalFilters *graph.PathFiltersInternal
	if input.Filters != nil {
		internalFilters = &graph.PathFiltersInternal{}
		if input.Filters.NodeFilter != nil {
			internalFilters.NodeFilter = &graph.NodeFilter{
				Conditions: translateConditions(input.Filters.NodeFilter.Conditions),
			}
		}
		if input.Filters.RelationFilter != nil {
			internalFilters.RelationFilter = &graph.RelationFilter{
				Conditions: translateConditions(input.Filters.RelationFilter.Conditions),
			}
		}
	}

	// Create parameters for the manager call
	graphParams := graph.FindPathsParams{
		StartNodeID: input.StartNodeID,
		EndNodeID:   input.EndNodeID,
		MaxLength:   input.MaxLength,
		Filters:     internalFilters,
	}

	// Execute the path finding using the manager
	result, err := t.manager.FindPaths(graphParams)
	if err != nil {
		return formatError(fmt.Errorf("path finding failed: %w", err)), nil
	}

	// Format the successful response
	pathCount := len(result.Paths)
	message := fmt.Sprintf("Found %d path(s) between %s and %s.", pathCount, graphParams.StartNodeID, graphParams.EndNodeID)
	if pathCount == 0 {
		message = fmt.Sprintf("No paths found between %s and %s within constraints.", graphParams.StartNodeID, graphParams.EndNodeID) // Updated message
	}

	resp := formatResponse(
		message,
		map[string]interface{}{ // Optional metadata
			"start_node":      graphParams.StartNodeID,
			"end_node":        graphParams.EndNodeID,
			"max_length":      graphParams.MaxLength,
			"paths_found":     pathCount,
			"filters_applied": input.Filters != nil, // Indicate if filters were present
		},
		result, // Include the full result data (list of paths)
	)
	return resp, nil
}
