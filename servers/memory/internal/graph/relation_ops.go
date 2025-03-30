package graph

import (
	"fmt"

	"mcp-memory/internal/types"
)

// CreateRelations adds new relations to the graph
func (m *KnowledgeGraphManager) CreateRelations(relations []types.Relation) ([]types.Relation, error) {
	m.mu.Lock()
	createdRelations := make([]types.Relation, 0, len(relations))
	for _, relation := range relations {
		// Auto-generate ID if not provided
		if relation.ID == "" {
			relation.ID = m.generateID()
		}

		if _, exists := m.graph.Relations[relation.ID]; exists {
			m.mu.Unlock()
			return nil, fmt.Errorf("relation with ID %s already exists", relation.ID)
		}
		if _, exists := m.graph.Entities[relation.Source]; !exists {
			m.mu.Unlock()
			return nil, fmt.Errorf("source entity %s does not exist", relation.Source)
		}
		if _, exists := m.graph.Entities[relation.Target]; !exists {
			m.mu.Unlock()
			return nil, fmt.Errorf("target entity %s does not exist", relation.Target)
		}
		m.graph.Relations[relation.ID] = relation
		createdRelations = append(createdRelations, relation)
	}
	m.mu.Unlock()

	if err := m.saveGraph(); err != nil {
		return nil, err
	}

	return createdRelations, nil
}

// DeleteRelations removes relations from the graph
func (m *KnowledgeGraphManager) DeleteRelations(relations []types.Relation) error {
	m.mu.Lock()
	newRelations := make(map[string]types.Relation)
	for id, relation := range m.graph.Relations {
		shouldKeep := true
		for _, toDelete := range relations {
			// First try to match by ID if provided
			if toDelete.ID != "" {
				if id == toDelete.ID {
					shouldKeep = false
					break
				}
			} else {
				// Fall back to matching by Source/Target/Type if ID is not provided
				if relation.Source == toDelete.Source &&
					relation.Target == toDelete.Target &&
					relation.Type == toDelete.Type {
					shouldKeep = false
					break
				}
			}
		}
		if shouldKeep {
			newRelations[id] = relation
		}
	}
	m.graph.Relations = newRelations
	m.mu.Unlock()

	if err := m.saveGraph(); err != nil {
		return err
	}

	return nil
}
