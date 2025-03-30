package graph

import (
	"fmt"

	"mcp-memory/internal/types"
)

// CreateEntities adds new entities to the graph
func (m *KnowledgeGraphManager) CreateEntities(entities []types.Entity) ([]types.Entity, error) {
	m.mu.Lock()
	createdEntities := make([]types.Entity, 0, len(entities))
	for _, entity := range entities {
		// Auto-generate ID if not provided
		if entity.ID == "" {
			entity.ID = m.generateID()
		}

		if _, exists := m.graph.Entities[entity.ID]; exists {
			m.mu.Unlock()
			return nil, fmt.Errorf("entity with ID %s already exists", entity.ID)
		}
		m.graph.Entities[entity.ID] = entity
		createdEntities = append(createdEntities, entity)
	}
	m.mu.Unlock()

	if err := m.saveGraph(); err != nil {
		return nil, err
	}

	return createdEntities, nil
}

// UpdateEntities performs partial updates on existing entities in batch
func (m *KnowledgeGraphManager) UpdateEntities(updates []types.Entity) ([]types.Entity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// --- Pass 1: Validation --- //
	for i, updateData := range updates {
		if updateData.ID == "" {
			return nil, fmt.Errorf("entity at index %d is missing the required 'id' field", i)
		}
		if _, exists := m.graph.Entities[updateData.ID]; !exists {
			return nil, fmt.Errorf("entity with ID %s (at index %d) does not exist", updateData.ID, i)
		}
	}

	// --- Pass 2: Application --- //
	updatedEntitiesResult := make([]types.Entity, 0, len(updates))
	originalEntities := make(map[string]types.Entity) // Store original state for potential rollback ON SAVE FAILURE

	for _, updateData := range updates {
		// We already validated existence in Pass 1
		existingEntity := m.graph.Entities[updateData.ID]

		// Store original before modifying (only needed for save rollback)
		originalEntities[updateData.ID] = existingEntity

		// Perform partial update on the entity in the map
		entityToUpdate := existingEntity

		if updateData.Type != "" {
			entityToUpdate.Type = updateData.Type
		}
		if updateData.Name != "" {
			entityToUpdate.Name = updateData.Name
		}
		// Only update description if it's explicitly provided and not empty.
		// To allow clearing, a different mechanism (e.g., pointers or explicit flags) would be needed.
		if updateData.Description != "" {
			entityToUpdate.Description = updateData.Description
		}

		// Merge metadata (simple shallow merge)
		if updateData.Metadata != nil {
			if entityToUpdate.Metadata == nil {
				entityToUpdate.Metadata = make(map[string]interface{})
			}
			for k, v := range updateData.Metadata {
				entityToUpdate.Metadata[k] = v
			}
		}

		m.graph.Entities[updateData.ID] = entityToUpdate
		// Append to results immediately after successful in-memory update
		updatedEntitiesResult = append(updatedEntitiesResult, entityToUpdate)
	}

	// --- Save --- //
	// Unlock before saving to allow reads during potentially slow I/O
	m.mu.Unlock()
	err := m.saveGraph()
	m.mu.Lock() // Relock

	if err != nil {
		// Attempt rollback on save error
		m.mu.Unlock() // Unlock for rollback modification
		m.mu.Lock()
		for id, original := range originalEntities {
			m.graph.Entities[id] = original // Restore original state
		}
		m.mu.Unlock()
		// Maybe attempt to save the rolled-back state? Or just log?
		// For now, just return the error after rollback attempt.
		return nil, fmt.Errorf("failed to save graph after updates, rollback attempted: %w", err)
	}

	// Return the entities as they were successfully updated in memory and saved
	return updatedEntitiesResult, nil
}

// DeleteEntities removes entities from the graph
func (m *KnowledgeGraphManager) DeleteEntities(ids []string) error {
	m.mu.Lock()
	for _, id := range ids {
		if _, exists := m.graph.Entities[id]; !exists {
			m.mu.Unlock()
			return fmt.Errorf("entity with ID %s does not exist", id)
		}
		delete(m.graph.Entities, id)
	}
	m.mu.Unlock()

	if err := m.saveGraph(); err != nil {
		return err
	}

	return nil
}
