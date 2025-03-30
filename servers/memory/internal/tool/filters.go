package tool

import (
	"mcp-memory/internal/graph"
)

// FilterCondition defines a basic filter condition for equality.
// Property can use dot notation for simple nested metadata access (e.g., "metadata.key").
type FilterCondition struct {
	Property string      `json:"property"` // e.g., "type", "name", "metadata.department"
	Value    interface{} `json:"value"`    // The value to match
	// TODO: Add Operator field (e.g., "eq", "ne", "in", "contains") for more complex filters
}

// NodeFilter defines filters to apply to entities. Assumes AND logic between conditions.
type NodeFilter struct {
	Conditions []FilterCondition `json:"conditions,omitempty"`
}

// RelationFilter defines filters to apply to relations. Assumes AND logic between conditions.
type RelationFilter struct {
	Conditions []FilterCondition `json:"conditions,omitempty"`
}

// TraversalFilters bundles filters for traversal operations.
type TraversalFilters struct {
	NodeFilter     *NodeFilter     `json:"node_filter,omitempty"`     // Nodes must match to be visited/included in result.
	RelationFilter *RelationFilter `json:"relation_filter,omitempty"` // Relations must match to be traversed.
}

// SubgraphFilters bundles filters for subgraph extraction.
type SubgraphFilters struct {
	NodeFilter     *NodeFilter     `json:"node_filter,omitempty"`     // Nodes must match to be included in the initial BFS search *and* final result.
	RelationFilter *RelationFilter `json:"relation_filter,omitempty"` // Relations must match to be included in the final result.
}

// PathFilters bundles filters for path finding.
type PathFilters struct {
	NodeFilter     *NodeFilter     `json:"node_filter,omitempty"`     // Nodes must match to be part of the path.
	RelationFilter *RelationFilter `json:"relation_filter,omitempty"` // Relations must match to be part of the path.
}

// translateConditions converts tool filter conditions to internal graph filter conditions.
func translateConditions(toolConditions []FilterCondition) []graph.FilterCondition {
	if toolConditions == nil {
		return nil
	}
	graphConditions := make([]graph.FilterCondition, len(toolConditions))
	for i, tc := range toolConditions {
		graphConditions[i] = graph.FilterCondition{
			Property: tc.Property,
			Value:    tc.Value,
			// Operator: tc.Operator, // Map when operator is added
		}
	}
	return graphConditions
}
