package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	// No need for types here, graph package provides params/results
)

//go:embed schemas/traverse_graph.json
var traverseGraphSchemaJSON []byte // Use []byte for json.RawMessage

// TraverseGraphTool implements the Tool interface for traversing the graph
type TraverseGraphTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewTraverseGraphTool creates a new TraverseGraphTool instance
func NewTraverseGraphTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &TraverseGraphTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *TraverseGraphTool) Name() string {
	return "traverse_graph"
}

// Description returns the description of the tool
func (t *TraverseGraphTool) Description() string {
	return "Performs graph traversal (BFS or DFS) starting from given nodes, returning visited nodes and depths."
}

// Schema returns the JSON schema for the tool's parameters
func (t *TraverseGraphTool) Schema() json.RawMessage {
	// Return the embedded schema content directly
	return traverseGraphSchemaJSON
}

// Execute performs the graph traversal
func (t *TraverseGraphTool) Execute(params json.RawMessage) (interface{}, error) {
	// Define a temporary struct to unmarshal the full input including tool filters
	var input struct {
		StartNodeIDs []string                 `json:"start_node_ids"`
		Algorithm    graph.TraversalAlgorithm `json:"algorithm"`
		MaxDepth     int                      `json:"max_depth"`
		Filters      *TraversalFilters        `json:"filters"` // Use tool filter type
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	// Translate tool filters to internal graph filters
	var internalFilters *graph.TraversalFiltersInternal
	if input.Filters != nil {
		internalFilters = &graph.TraversalFiltersInternal{}
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
	graphParams := graph.TraverseParams{
		StartNodeIDs: input.StartNodeIDs,
		Algorithm:    input.Algorithm,
		MaxDepth:     input.MaxDepth,
		Filters:      internalFilters,
	}

	// Default algorithm if empty or not provided
	if graphParams.Algorithm == "" {
		graphParams.Algorithm = graph.BFSAlgorithm
	}

	// Execute the traversal using the manager stored in the struct
	result, err := t.manager.Traverse(graphParams)
	if err != nil {
		// Assuming manager errors are suitable for direct return
		return formatError(fmt.Errorf("graph traversal failed: %w", err)), nil
	}

	// Format the successful response
	return formatResponse(
		fmt.Sprintf("Traversal complete. Visited %d unique entities.", len(result.VisitedEntities)),
		map[string]interface{}{ // Optional metadata
			"start_nodes_count": len(graphParams.StartNodeIDs),
			"algorithm_used":    graphParams.Algorithm,
			"max_depth_limit":   graphParams.MaxDepth,
			"unique_visits":     len(result.VisitedEntities),
			"filters_applied":   input.Filters != nil, // Indicate if filters were present
		},
		result, // Include the full result data
	), nil
}
