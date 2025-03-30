package graph

import (
	"strings"

	"mcp-memory/internal/types"
)

// SearchNodes searches for entities based on type, metadata, and text content
func (m *KnowledgeGraphManager) SearchNodes(entityType string, metadata map[string]interface{}) []types.Entity {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []types.Entity
	for _, entity := range m.graph.Entities {
		// Match by type if specified
		if entityType != "" && entity.Type != entityType {
			continue
		}

		// Match by metadata if specified
		match := true
		for key, value := range metadata {
			if entityValue, ok := entity.Metadata[key]; !ok || entityValue != value {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		results = append(results, entity)
	}

	return results
}

// SearchByText searches for entities based on text content in names, types, and observations
func (m *KnowledgeGraphManager) SearchByText(query string) []types.Entity {
	if query == "" {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	queryLower := strings.ToLower(query)
	seen := make(map[string]bool)
	var results []types.Entity

	// Search in entities
	for _, entity := range m.graph.Entities {
		if seen[entity.ID] {
			continue
		}

		// Check name and type
		if strings.Contains(strings.ToLower(entity.Name), queryLower) ||
			strings.Contains(strings.ToLower(entity.Type), queryLower) ||
			strings.Contains(strings.ToLower(entity.Description), queryLower) {
			results = append(results, entity)
			seen[entity.ID] = true
			continue
		}

		// Check metadata values
		for _, value := range entity.Metadata {
			if str, ok := value.(string); ok {
				if strings.Contains(strings.ToLower(str), queryLower) {
					results = append(results, entity)
					seen[entity.ID] = true
					break
				}
			}
		}
	}

	// Search in observations
	for _, obs := range m.graph.Observations {
		if seen[obs.EntityID] {
			continue
		}

		if strings.Contains(strings.ToLower(obs.Content), queryLower) ||
			strings.Contains(strings.ToLower(obs.Type), queryLower) ||
			strings.Contains(strings.ToLower(obs.Description), queryLower) {
			if entity, exists := m.graph.Entities[obs.EntityID]; exists {
				results = append(results, entity)
				seen[obs.EntityID] = true
			}
		}
	}

	// Search in relations
	for _, relation := range m.graph.Relations {
		if strings.Contains(strings.ToLower(relation.Type), queryLower) ||
			strings.Contains(strings.ToLower(relation.Description), queryLower) {

			// Add source entity if not seen
			if !seen[relation.Source] {
				if entity, exists := m.graph.Entities[relation.Source]; exists {
					results = append(results, entity)
					seen[relation.Source] = true
				}
			}
			// Add target entity if not seen
			if !seen[relation.Target] {
				if entity, exists := m.graph.Entities[relation.Target]; exists {
					results = append(results, entity)
					seen[relation.Target] = true
				}
			}
		}
	}

	return results
}
