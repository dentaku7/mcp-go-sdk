package graph

import (
	"fmt"

	"mcp-memory/internal/types"
)

// OpenNodes returns all entities connected to the given entity IDs
func (m *KnowledgeGraphManager) OpenNodes(ids []string) (*types.KnowledgeGraphResult, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one entity ID must be provided")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &types.KnowledgeGraphResult{
		Entities:  make(map[string]types.Entity),
		Relations: make(map[string]types.Relation),
	}

	// Create a map for quick lookup
	idSet := make(map[string]bool)
	for _, id := range ids {
		if id == "" {
			return nil, fmt.Errorf("entity ID cannot be empty")
		}
		idSet[id] = true
	}

	// Add requested entities
	for _, entity := range m.graph.Entities {
		if idSet[entity.ID] {
			result.Entities[entity.ID] = entity
		}
	}

	// Verify all requested IDs were found
	for id := range idSet {
		if _, exists := result.Entities[id]; !exists {
			return nil, fmt.Errorf("entity with ID '%s' not found", id)
		}
	}

	// Add relations where either source or target is in the result
	for id, relation := range m.graph.Relations {
		if idSet[relation.Source] || idSet[relation.Target] {
			result.Relations[id] = relation
		}
	}

	return result, nil
}
