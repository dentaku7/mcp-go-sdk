package graph

import (
	"fmt"
	"reflect"
	"strings"

	"mcp-memory/internal/types"
)

// --- Filter Logic Helpers (Internal) ---

// getPropertyValue dynamically retrieves a value from an entity or relation
// based on a property string (e.g., "Type", "Name", "Metadata.department").
// Returns the value and true if found, otherwise nil and false.
func getPropertyValue(item interface{}, property string) (interface{}, bool) {
	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, false
	}
	parts := strings.Split(property, ".")
	currentVal := val
	for i, part := range parts {
		if !(currentVal.Kind() == reflect.Struct || currentVal.Kind() == reflect.Map) {
			return nil, false
		}
		if currentVal.Kind() == reflect.Struct {
			field := currentVal.FieldByNameFunc(func(name string) bool {
				return strings.EqualFold(name, part)
			})
			if !field.IsValid() {
				return nil, false
			}
			currentVal = field
		} else if currentVal.Kind() == reflect.Map {
			// Ensure map key is of appropriate type for lookup
			// For metadata map[string]interface{}, key must be string
			if currentVal.Type().Key().Kind() == reflect.String {
				keyVal := reflect.ValueOf(part)
				mapValue := currentVal.MapIndex(keyVal)
				if !mapValue.IsValid() {
					return nil, false
				}
				currentVal = mapValue
			} else {
				return nil, false // Map key is not string
			}
		}
		if currentVal.Kind() == reflect.Interface {
			currentVal = currentVal.Elem()
		}
		if i == len(parts)-1 {
			if currentVal.IsValid() {
				return currentVal.Interface(), true
			}
			return nil, false
		}
	}
	return nil, false
}

// matchesConditions checks if an item (entity or relation) matches all filter conditions.
func matchesConditions(item interface{}, conditions []FilterCondition) bool {
	if len(conditions) == 0 {
		return true
	}
	for _, cond := range conditions {
		actualValue, found := getPropertyValue(item, cond.Property)
		if !found {
			return false
		}
		// TODO: Implement different operators (ne, in, contains, etc.)
		if fmt.Sprint(actualValue) != fmt.Sprint(cond.Value) {
			return false
		}
	}
	return true
}

// matchesEntity checks if an entity matches the node filter.
func matchesEntity(entity types.Entity, filter *NodeFilter) bool {
	if filter == nil || len(filter.Conditions) == 0 {
		return true
	}
	return matchesConditions(entity, filter.Conditions)
}

// matchesRelation checks if a relation matches the relation filter.
func matchesRelation(relation types.Relation, filter *RelationFilter) bool {
	if filter == nil || len(filter.Conditions) == 0 {
		return true
	}
	return matchesConditions(relation, filter.Conditions)
}

// --- Internal Filter Definitions (for graph package use) ---

// FilterCondition defines a basic filter condition for equality used internally.
type FilterCondition struct {
	Property string      // e.g., "type", "name", "metadata.department"
	Value    interface{} // The value to match
	// TODO: Add Operator field (e.g., "eq", "ne", "in", "contains")
}

// NodeFilter defines filters to apply to entities internally.
type NodeFilter struct {
	Conditions []FilterCondition
}

// RelationFilter defines filters to apply to relations internally.
type RelationFilter struct {
	Conditions []FilterCondition
}

// TraversalFiltersInternal bundles filters for traversal operations.
type TraversalFiltersInternal struct {
	NodeFilter     *NodeFilter
	RelationFilter *RelationFilter
}

// SubgraphFiltersInternal bundles filters for subgraph extraction.
type SubgraphFiltersInternal struct {
	NodeFilter     *NodeFilter
	RelationFilter *RelationFilter
}

// PathFiltersInternal bundles filters for path finding.
type PathFiltersInternal struct {
	NodeFilter     *NodeFilter
	RelationFilter *RelationFilter
}

// --- Traversal & Pathfinding Methods ---

// TraversalAlgorithm defines the type of traversal (BFS or DFS)
type TraversalAlgorithm string

const (
	BFSAlgorithm TraversalAlgorithm = "BFS"
	DFSAlgorithm TraversalAlgorithm = "DFS"
)

// TraverseParams holds parameters for the Traverse method
type TraverseParams struct {
	StartNodeIDs []string
	Algorithm    TraversalAlgorithm
	MaxDepth     int
	Filters      *TraversalFiltersInternal // Use internal filter type
}

// TraverseResult holds the result of a graph traversal
type TraverseResult struct {
	VisitedEntities []types.Entity `json:"visited_entities"`
	VisitedDepths   map[string]int `json:"visited_depths"`
}

// Traverse executes a BFS or DFS traversal on the graph.
func (m *KnowledgeGraphManager) Traverse(params TraverseParams) (*TraverseResult, error) {
	m.mu.RLock() // Use Read Lock for traversal
	defer m.mu.RUnlock()

	accessor := NewStandardGraphAccessor(m.graph.Entities, m.graph.Relations)

	visitedEntitiesResult := make([]types.Entity, 0)
	visitedDepthsResult := make(map[string]int)

	// Create VisitFunc incorporating the node filter
	visitFn := func(entity types.Entity, depth int) bool {
		// Check node filter first
		if params.Filters != nil && params.Filters.NodeFilter != nil {
			if !matchesEntity(entity, params.Filters.NodeFilter) {
				return false // Node doesn't match filter, don't visit or explore
			}
		}
		// If node matches (or no filter), record the visit
		if _, exists := visitedDepthsResult[entity.ID]; !exists {
			visitedEntitiesResult = append(visitedEntitiesResult, entity)
		}
		visitedDepthsResult[entity.ID] = depth
		return true // Node matches filter (or no filter), okay to explore neighbors
	}

	// Create NeighborFunc incorporating the relation filter
	var neighborFn NeighborFunc
	if params.Filters != nil && params.Filters.RelationFilter != nil {
		relFilter := params.Filters.RelationFilter // Capture loop variable
		neighborFn = func(accessor GraphAccessor, current types.Entity) []types.Entity {
			neighbors := make(map[string]types.Entity)
			// Outgoing relations
			relationsFrom := accessor.GetRelationsFrom(current.ID)
			for _, rel := range relationsFrom {
				if matchesRelation(rel, relFilter) { // Check relation filter
					if target, exists := accessor.GetEntity(rel.Target); exists {
						neighbors[target.ID] = target
					}
				}
			}
			// Incoming relations (if DefaultNeighborFunc includes them, this should too)
			relationsTo := accessor.GetRelationsTo(current.ID)
			for _, rel := range relationsTo {
				if matchesRelation(rel, relFilter) { // Check relation filter
					if source, exists := accessor.GetEntity(rel.Source); exists {
						neighbors[source.ID] = source
					}
				}
			}
			neighborSlice := make([]types.Entity, 0, len(neighbors))
			for _, neighbor := range neighbors {
				neighborSlice = append(neighborSlice, neighbor)
			}
			return neighborSlice
		}
	} else {
		neighborFn = DefaultNeighborFunc // Use default if no relation filter
	}

	var err error
	switch params.Algorithm {
	case BFSAlgorithm, "":
		err = BFS(accessor, params.StartNodeIDs, params.MaxDepth, visitFn, neighborFn)
	case DFSAlgorithm:
		err = DFS(accessor, params.StartNodeIDs, params.MaxDepth, visitFn, neighborFn)
	default:
		return nil, fmt.Errorf("unknown traversal algorithm: %s. Please specify a valid algorithm: (%s or %s)", params.Algorithm, BFSAlgorithm, DFSAlgorithm)
	}

	if err != nil {
		return nil, fmt.Errorf("traversal failed: %w", err)
	}

	result := &TraverseResult{
		VisitedEntities: visitedEntitiesResult,
		VisitedDepths:   visitedDepthsResult,
	}

	return result, nil
}

// GetSubgraphParams holds parameters for the GetSubgraph method
type GetSubgraphParams struct {
	StartNodeIDs []string
	Radius       int
	Filters      *SubgraphFiltersInternal // Use internal filter type
}

// GetSubgraph extracts a subgraph centered around specific nodes.
func (m *KnowledgeGraphManager) GetSubgraph(params GetSubgraphParams) (*SubgraphResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accessor := NewStandardGraphAccessor(m.graph.Entities, m.graph.Relations)

	// Create NodeVisitFunc based on NodeFilter
	var nodeVisitFn NodeVisitFunc = nil
	if params.Filters != nil && params.Filters.NodeFilter != nil {
		nodeFilter := params.Filters.NodeFilter // Capture loop variable
		nodeVisitFn = func(entity types.Entity, depth int) bool {
			// Node must match the filter to be included/explored
			return matchesEntity(entity, nodeFilter)
		}
	}

	// Create RelationFilterFunc based on RelationFilter
	var relationFilterFn RelationFilterFunc = nil
	if params.Filters != nil && params.Filters.RelationFilter != nil {
		relFilter := params.Filters.RelationFilter // Capture loop variable
		relationFilterFn = func(relation types.Relation, entitiesInSubgraph map[string]struct{}) bool {
			// Default logic: both source and target must be in the subgraph entity set
			_, sourceIn := entitiesInSubgraph[relation.Source]
			_, targetIn := entitiesInSubgraph[relation.Target]
			if !(sourceIn && targetIn) {
				return false
			}
			// Additionally, the relation must match the filter conditions
			return matchesRelation(relation, relFilter)
		}
	} else {
		// Use default if no relation filter provided
		relationFilterFn = DefaultRelationFilter
	}

	// Call ExtractSubgraph with both filter functions
	subgraph, err := ExtractSubgraph(accessor, params.StartNodeIDs, params.Radius, nodeVisitFn, relationFilterFn)
	if err != nil {
		return nil, fmt.Errorf("subgraph extraction failed: %w", err)
	}

	// Note: ExtractSubgraph currently returns maps pointing to the *original* graph data
	// via the StandardGraphAccessor. For true isolation/copying, we might need deep copies here.
	// For now, assuming read-only access is sufficient for the caller.

	return subgraph, nil
}

// FindPathsParams holds parameters for the FindPaths method
type FindPathsParams struct {
	StartNodeID string
	EndNodeID   string
	MaxLength   int
	Filters     *PathFiltersInternal // Use internal filter type
}

// FindPaths searches for paths between two nodes.
func (m *KnowledgeGraphManager) FindPaths(params FindPathsParams) (*PathsResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accessor := NewStandardGraphAccessor(m.graph.Entities, m.graph.Relations)

	// Create PathFilterFunc based on combined node and relation filters
	var filterFn PathFilterFunc = nil
	if params.Filters != nil && (params.Filters.NodeFilter != nil || params.Filters.RelationFilter != nil) {
		filterFn = func(item interface{}, currentLength int) bool {
			switch concreteItem := item.(type) {
			case types.Entity:
				return matchesEntity(concreteItem, params.Filters.NodeFilter)
			case types.Relation:
				return matchesRelation(concreteItem, params.Filters.RelationFilter)
			default:
				return false // Unknown item type
			}
		}
	}

	paths, err := FindPaths(accessor, params.StartNodeID, params.EndNodeID, params.MaxLength, filterFn)
	if err != nil {
		return nil, fmt.Errorf("path finding failed: %w", err)
	}

	return paths, nil
}
