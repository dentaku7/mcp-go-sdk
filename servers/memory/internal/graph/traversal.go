package graph

import (
	"container/list"
	"fmt"

	"mcp-memory/internal/types" // Assuming types are defined here
)

// GraphAccessor defines the methods needed by traversal algorithms to access graph data.
// This allows decoupling the algorithms from the specific storage implementation.
type GraphAccessor interface {
	GetEntity(id string) (types.Entity, bool)
	GetRelationsFrom(id string) []types.Relation // Relations where the entity is the source
	GetRelationsTo(id string) []types.Relation   // Relations where the entity is the target
	// Consider adding GetNeighbors(id string) []types.Entity if more efficient
}

// StandardGraphAccessor provides access to the graph data stored in the KnowledgeGraph struct.
// It assumes direct access to the maps for simplicity in this context.
// A more robust implementation might copy data or use read locks.
type StandardGraphAccessor struct {
	entities  map[string]types.Entity
	relations map[string]types.Relation
}

// NewStandardGraphAccessor creates an accessor for the graph data.
// IMPORTANT: This currently accesses the maps directly. The caller (Manager)
// must ensure appropriate locking during the access operations.
func NewStandardGraphAccessor(entities map[string]types.Entity, relations map[string]types.Relation) *StandardGraphAccessor {
	return &StandardGraphAccessor{
		entities:  entities,
		relations: relations,
	}
}

func (g *StandardGraphAccessor) GetEntity(id string) (types.Entity, bool) {
	entity, exists := g.entities[id]
	return entity, exists
}

// GetRelationsFrom returns relations originating from the given entity ID.
func (g *StandardGraphAccessor) GetRelationsFrom(id string) []types.Relation {
	var outgoing []types.Relation
	for _, rel := range g.relations {
		if rel.Source == id {
			outgoing = append(outgoing, rel)
		}
	}
	return outgoing
}

// GetRelationsTo returns relations pointing to the given entity ID.
func (g *StandardGraphAccessor) GetRelationsTo(id string) []types.Relation {
	var incoming []types.Relation
	for _, rel := range g.relations {
		if rel.Target == id {
			incoming = append(incoming, rel)
		}
	}
	return incoming
}

// --- Traversal Algorithms ---

// VisitFunc is a callback function executed for each visited node during traversal.
// It receives the node and current depth. Return false to stop exploring neighbors of this node.
// It should return true ONLY if the node itself matches any applicable node filters.
type VisitFunc func(entity types.Entity, depth int) (matchesFilterAndExplore bool)

// NeighborFunc defines how to get neighbors for a node during traversal.
// It should ONLY return neighbors connected by relations that match applicable relation filters.
type NeighborFunc func(accessor GraphAccessor, current types.Entity) []types.Entity

// DefaultNeighborFunc returns all direct neighbors (connected by any relation).
// Note: This default doesn't apply any filtering.
func DefaultNeighborFunc(accessor GraphAccessor, current types.Entity) []types.Entity {
	neighbors := make(map[string]types.Entity) // Use map to avoid duplicates

	// Outgoing relations
	relationsFrom := accessor.GetRelationsFrom(current.ID)
	for _, rel := range relationsFrom {
		if target, exists := accessor.GetEntity(rel.Target); exists {
			neighbors[target.ID] = target
		}
	}

	// Incoming relations (for bidirectional traversal if needed, adjust if only outgoing is desired)
	relationsTo := accessor.GetRelationsTo(current.ID)
	for _, rel := range relationsTo {
		if source, exists := accessor.GetEntity(rel.Source); exists {
			neighbors[source.ID] = source
		}
	}

	neighborSlice := make([]types.Entity, 0, len(neighbors))
	for _, neighbor := range neighbors {
		neighborSlice = append(neighborSlice, neighbor)
	}
	return neighborSlice
}

// BFS performs a Breadth-First Search on the graph.
// graph: Provides access to graph data (nodes, edges).
// startNodeIDs: A slice of entity IDs from which to start the traversal.
// maxDepth: Maximum depth to traverse (-1 for unlimited).
// visitFn: Function called for each *potential* node visit. Should return true if node matches filters and exploration should continue.
// neighborFn: Function to retrieve *filtered* neighbors for a given node.
func BFS(graph GraphAccessor, startNodeIDs []string, maxDepth int, visitFn VisitFunc, neighborFn NeighborFunc) error {
	if neighborFn == nil {
		neighborFn = DefaultNeighborFunc
	}

	queue := list.New()
	visited := make(map[string]int) // Store visited node ID and its depth

	// Initialize queue with start nodes at depth 0
	for _, startNodeID := range startNodeIDs {
		if _, exists := visited[startNodeID]; exists {
			continue
		}
		startNode, exists := graph.GetEntity(startNodeID)
		if !exists {
			return fmt.Errorf("start node with ID '%s' not found", startNodeID)
		}

		// Apply visitFn (which includes node filter check) to start nodes
		if visitFn != nil && !visitFn(startNode, 0) {
			continue // Start node doesn't match filter
		}

		queue.PushBack(struct {
			entity types.Entity
			depth  int
		}{startNode, 0})
		visited[startNodeID] = 0
	}

	// Process the queue
	for queue.Len() > 0 {
		element := queue.Front()
		queue.Remove(element)

		currentItem := element.Value.(struct {
			entity types.Entity
			depth  int
		})
		currentEntity := currentItem.entity
		currentDepth := currentItem.depth

		// Depth limit check (already handled by queueing logic below)
		nextDepth := currentDepth + 1
		if maxDepth >= 0 && nextDepth > maxDepth {
			continue
		}

		// Get neighbors (neighborFn should handle relation filtering)
		neighbors := neighborFn(graph, currentEntity)
		for _, neighbor := range neighbors {
			// Check if already visited
			if _, alreadyVisited := visited[neighbor.ID]; alreadyVisited {
				continue
			}

			// Apply visitFn (node filter) before adding to queue
			if visitFn != nil && !visitFn(neighbor, nextDepth) {
				continue // Neighbor node doesn't match filter
			}

			// Mark visited and enqueue
			visited[neighbor.ID] = nextDepth
			queue.PushBack(struct {
				entity types.Entity
				depth  int
			}{neighbor, nextDepth})
		}
	}

	return nil
}

// DFS performs a Depth-First Search on the graph.
// Parameters are similar to BFS.
// visitFn: Function called for each *potential* node visit. Should return true if node matches filters and exploration should continue.
// neighborFn: Function to retrieve *filtered* neighbors for a given node.
func DFS(graph GraphAccessor, startNodeIDs []string, maxDepth int, visitFn VisitFunc, neighborFn NeighborFunc) error {
	if neighborFn == nil {
		neighborFn = DefaultNeighborFunc
	}

	stack := list.New()             // Use list as a stack (PushBack, Back, Remove)
	visited := make(map[string]int) // Store visited node ID and its depth

	// Initialize stack with start nodes at depth 0
	for i := len(startNodeIDs) - 1; i >= 0; i-- {
		startNodeID := startNodeIDs[i]
		startNode, exists := graph.GetEntity(startNodeID)
		if !exists {
			return fmt.Errorf("start node with ID '%s' not found", startNodeID)
		}

		// Apply visitFn (node filter check) to start nodes before pushing
		if visitFn != nil && !visitFn(startNode, 0) {
			continue // Start node doesn't match filter
		}

		stack.PushBack(struct {
			entity types.Entity
			depth  int
		}{startNode, 0})
	}

	// Process the stack
	for stack.Len() > 0 {
		element := stack.Back() // Get top element
		stack.Remove(element)   // Pop

		currentItem := element.Value.(struct {
			entity types.Entity
			depth  int
		})
		currentEntity := currentItem.entity
		currentDepth := currentItem.depth

		// Check if already visited *at a shallower or equal depth*.
		if existingDepth, alreadyVisited := visited[currentEntity.ID]; alreadyVisited && existingDepth <= currentDepth {
			continue
		}

		// Mark visited (or update depth)
		visited[currentEntity.ID] = currentDepth

		// Check depth limit before exploring neighbors
		nextDepth := currentDepth + 1
		if maxDepth >= 0 && nextDepth > maxDepth {
			continue
		}

		// Get neighbors (neighborFn handles relation filtering)
		neighbors := neighborFn(graph, currentEntity)
		for i := len(neighbors) - 1; i >= 0; i-- {
			neighbor := neighbors[i]

			// Check visited status before applying node filter
			if existingDepth, alreadyVisited := visited[neighbor.ID]; alreadyVisited && existingDepth <= nextDepth {
				continue // Already visited at a better or equal depth
			}

			// Apply visitFn (node filter) before pushing to stack
			if visitFn != nil && !visitFn(neighbor, nextDepth) {
				continue // Neighbor node doesn't match filter
			}

			// Push to stack
			stack.PushBack(struct {
				entity types.Entity
				depth  int
			}{neighbor, nextDepth})
		}
	}

	return nil
}

// --- Subgraph Extraction ---

// SubgraphResult holds the nodes and relations of the extracted subgraph.
type SubgraphResult struct {
	Entities  map[string]types.Entity
	Relations map[string]types.Relation
}

// RelationFilterFunc defines a function to filter relations during subgraph extraction.
// It receives the relation and the set of entity IDs included in the subgraph.
// Return true to include the relation.
type RelationFilterFunc func(relation types.Relation, entitiesInSubgraph map[string]struct{}) bool

// NodeVisitFunc defines a function to filter nodes during the initial BFS phase of subgraph extraction.
// Return true if the node should be included and its neighbors explored (within radius).
type NodeVisitFunc func(entity types.Entity, depth int) (includeAndExplore bool)

// DefaultRelationFilter includes any relation connecting two entities within the subgraph.
func DefaultRelationFilter(relation types.Relation, entitiesInSubgraph map[string]struct{}) bool {
	_, sourceIn := entitiesInSubgraph[relation.Source]
	_, targetIn := entitiesInSubgraph[relation.Target]
	return sourceIn && targetIn
}

// ExtractSubgraph finds all entities within a given radius of the start nodes
// and all relations connecting those entities, applying filters.
// graph: Provides access to graph data.
// startNodeIDs: IDs of the central entities.
// radius: The maximum distance (number of hops) from the start nodes.
// nodeVisitFn: Optional function to filter nodes during the initial BFS search. Return true to include/explore.
// relationFilterFn: Optional function to filter which relations are included in the final result.
func ExtractSubgraph(graph GraphAccessor, startNodeIDs []string, radius int, nodeVisitFn NodeVisitFunc, relationFilterFn RelationFilterFunc) (*SubgraphResult, error) {
	if radius < 0 {
		return nil, fmt.Errorf("radius cannot be negative")
	}
	if relationFilterFn == nil {
		relationFilterFn = DefaultRelationFilter
	}

	subgraphEntities := make(map[string]types.Entity)

	// Use BFS to find all nodes within the radius, applying nodeVisitFn
	bfsVisitFn := func(entity types.Entity, depth int) bool {
		include := true
		if nodeVisitFn != nil {
			include = nodeVisitFn(entity, depth)
		}

		if include && depth <= radius {
			subgraphEntities[entity.ID] = entity
			return true // Continue exploring neighbors if node included and within radius
		}
		return false // Stop exploring if node filtered out or radius exceeded
	}

	// We run BFS without a neighbor filter here; node filtering is done by bfsVisitFn.
	// Relation filtering happens *after* collecting nodes.
	err := BFS(graph, startNodeIDs, radius, bfsVisitFn, nil)
	if err != nil {
		return nil, fmt.Errorf("error during BFS for subgraph extraction: %w", err)
	}

	// --- Collect Relations --- //
	subgraphRelations := make(map[string]types.Relation)
	entitiesInSubgraphSet := make(map[string]struct{}, len(subgraphEntities))
	for id := range subgraphEntities {
		entitiesInSubgraphSet[id] = struct{}{}
	}

	// Iterate through all relations in the original graph accessor *once*
	// (Requires a way to access all relations, let's assume StandardGraphAccessor for now)
	stdAccessor, ok := graph.(*StandardGraphAccessor)
	if !ok {
		// If it's not the standard accessor, we might need a new method in the interface
		// like GetAllRelations() or iterate through nodes and their outgoing relations.
		// For now, return an error or handle based on expected usage.
		// Alternative: Iterate through subgraphEntities and get their relations.
		// This might duplicate relation checks but avoids modifying the interface now.
		for entityID := range subgraphEntities {
			relationsFrom := graph.GetRelationsFrom(entityID)
			for _, rel := range relationsFrom {
				if relationFilterFn(rel, entitiesInSubgraphSet) {
					subgraphRelations[rel.ID] = rel
				}
			}
			// Avoid double-adding by only checking outgoing, assuming the filter handles directionality if needed
			// If bidirectional relations need special handling, the filter must manage it.
		}
	} else {
		// Efficient path using direct access (requires lock safety handled by caller)
		for id, rel := range stdAccessor.relations {
			if relationFilterFn(rel, entitiesInSubgraphSet) {
				subgraphRelations[id] = rel
			}
		}
	}

	result := &SubgraphResult{
		Entities:  subgraphEntities,
		Relations: subgraphRelations,
	}

	return result, nil
}

// --- Path Finding ---

// Path represents a sequence of entities and relations connecting them.
// Format: [Entity1, Relation1, Entity2, Relation2, Entity3, ...]
// Note: Using interface{} for flexibility, but requires type assertions by the caller.
type Path []interface{}

// PathResult holds the list of paths found.
type PathsResult struct {
	Paths []Path
}

// PathFilterFunc defines a function to filter entities or relations during path finding.
// It receives the item (Entity or Relation) and the current path length.
// Return true to allow the entity/relation in the path.
type PathFilterFunc func(item interface{}, currentLength int) bool

// FindPaths searches for all simple paths (no repeated nodes) between start and end nodes.
// graph: Provides access to graph data.
// startNodeID: The ID of the starting entity.
// endNodeID: The ID of the target entity.
// maxLength: Maximum path length (number of relations). -1 for unlimited.
// filterFn: Optional function to filter nodes/relations along the path.
func FindPaths(graph GraphAccessor, startNodeID, endNodeID string, maxLength int, filterFn PathFilterFunc) (*PathsResult, error) {
	startNode, exists := graph.GetEntity(startNodeID)
	if !exists {
		return nil, fmt.Errorf("start node with ID '%s' not found", startNodeID)
	}

	// Check if start node itself matches the filter
	if filterFn != nil && !filterFn(startNode, 0) {
		return &PathsResult{Paths: []Path{}}, nil // Start node filtered out, no paths possible
	}

	var foundPaths []Path
	queue := list.New()

	// Queue stores current path and visited nodes *for that specific path*
	initialPath := Path{startNode}
	initialVisited := map[string]struct{}{startNodeID: {}}
	queue.PushBack(struct {
		currentPath Path
		visited     map[string]struct{}
	}{initialPath, initialVisited})

	for queue.Len() > 0 {
		element := queue.Front()
		queue.Remove(element)

		currentItem := element.Value.(struct {
			currentPath Path
			visited     map[string]struct{}
		})
		currentPath := currentItem.currentPath
		currentVisited := currentItem.visited

		// Get the last node in the current path
		lastNode := currentPath[len(currentPath)-1].(types.Entity)

		// Check if we reached the target
		if lastNode.ID == endNodeID {
			foundPaths = append(foundPaths, currentPath)
			continue
		}

		// Check max length
		currentLength := len(currentPath) / 2
		if maxLength >= 0 && currentLength >= maxLength {
			continue
		}

		// Explore neighbors (outgoing relations only)
		relations := graph.GetRelationsFrom(lastNode.ID)
		for _, rel := range relations {
			neighborNode, neighborExists := graph.GetEntity(rel.Target)
			if !neighborExists {
				continue
			}

			// --- Check for Cycles --- //
			if _, visitedOnThisPath := currentVisited[neighborNode.ID]; visitedOnThisPath {
				continue
			}

			// --- Apply Filters --- //
			nextLength := currentLength + 1
			if filterFn != nil {
				if !filterFn(rel, nextLength) || !filterFn(neighborNode, nextLength) {
					continue // Skip if relation or neighbor node is filtered out
				}
			}

			// --- Extend Path and Add to Queue --- //
			newPath := make(Path, len(currentPath)+2)
			copy(newPath, currentPath)
			newPath[len(currentPath)] = rel            // Add relation
			newPath[len(currentPath)+1] = neighborNode // Add neighbor node

			// Copy visited map for the new path state
			newVisited := make(map[string]struct{}, len(currentVisited)+1)
			for k, v := range currentVisited {
				newVisited[k] = v
			}
			newVisited[neighborNode.ID] = struct{}{}

			queue.PushBack(struct {
				currentPath Path
				visited     map[string]struct{}
			}{newPath, newVisited})
		}
	}

	return &PathsResult{Paths: foundPaths}, nil
}
