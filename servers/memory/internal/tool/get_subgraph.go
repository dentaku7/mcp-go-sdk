package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
)

//go:embed schemas/get_subgraph.json
var getSubgraphSchemaJSON []byte // Use []byte for json.RawMessage

// GetSubgraphTool implements the Tool interface for extracting subgraphs
type GetSubgraphTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewGetSubgraphTool creates a new GetSubgraphTool instance
func NewGetSubgraphTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &GetSubgraphTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *GetSubgraphTool) Name() string {
	return "get_subgraph"
}

// Description returns the description of the tool
func (t *GetSubgraphTool) Description() string {
	return "Extracts a subgraph containing all entities within a specified radius (hops) of the start nodes, including the relations connecting them."
}

// Schema returns the JSON schema for the tool's parameters
func (t *GetSubgraphTool) Schema() json.RawMessage {
	return getSubgraphSchemaJSON
}

// Execute performs the subgraph extraction
func (t *GetSubgraphTool) Execute(params json.RawMessage) (interface{}, error) {
	// Define a temporary struct to unmarshal the full input including tool filters
	var input struct {
		StartNodeIDs []string         `json:"start_node_ids"`
		Radius       int              `json:"radius"`
		Filters      *SubgraphFilters `json:"filters"` // Use tool filter type
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	if input.Radius < 0 {
		return formatError(fmt.Errorf("radius cannot be negative")), nil
	}

	// Translate tool filters to internal graph filters
	var internalFilters *graph.SubgraphFiltersInternal
	if input.Filters != nil {
		internalFilters = &graph.SubgraphFiltersInternal{}
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
	graphParams := graph.GetSubgraphParams{
		StartNodeIDs: input.StartNodeIDs,
		Radius:       input.Radius,
		Filters:      internalFilters,
	}

	// Execute the extraction using the manager
	result, err := t.manager.GetSubgraph(graphParams)
	if err != nil {
		return formatError(fmt.Errorf("subgraph extraction failed: %w", err)), nil
	}

	// Format the successful response
	resp := formatResponse(
		fmt.Sprintf("Subgraph extracted with %d entities and %d relations.", len(result.Entities), len(result.Relations)),
		map[string]interface{}{ // Optional metadata
			"start_nodes_count": len(graphParams.StartNodeIDs),
			"radius_limit":      graphParams.Radius,
			"entities_found":    len(result.Entities),
			"relations_found":   len(result.Relations),
			"filters_applied":   input.Filters != nil, // Indicate if filters were present
		},
		result, // Include the full result data (maps of entities and relations)
	)
	return resp, nil
}
