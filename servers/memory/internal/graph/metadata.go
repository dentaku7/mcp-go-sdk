package graph

import (
	"encoding/json"
	"fmt"
	"strings"

	"mcp-memory/internal/types"
)

// --- Helper function for nested map operations ---

// setNestedValue applies an update (merge, replace, delete) to a potentially nested path within a map.
// path is a dot-separated string (e.g., "a.b.c").
// operation can be "merge", "replace", or "delete".
func setNestedValue(data map[string]interface{}, path string, value interface{}, operation string) (map[string]interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - perform the operation
			switch operation {
			case "delete":
				delete(current, part)
				return data, nil // Return the modified original map
			case "replace":
				current[part] = value
				return data, nil // Return the modified original map
			case "merge":
				existing, exists := current[part]
				if !exists {
					// If target doesn't exist, it's the same as replace
					current[part] = value
					return data, nil
				}

				existingMap, existingIsMap := existing.(map[string]interface{})
				valueMap, valueIsMap := value.(map[string]interface{})

				if existingIsMap && valueIsMap {
					// Deep merge maps
					for k, v := range valueMap {
						existingMap[k] = v // Simple overwrite for nested merge for now
						// TODO: Implement deeper recursive merge if needed
					}
					current[part] = existingMap
				} else {
					// If types don't match or not both maps, replace
					current[part] = value
				}
				return data, nil
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}
		} else {
			// Traverse deeper
			next, exists := current[part]
			if !exists {
				if operation == "delete" {
					return data, nil // Path doesn't exist, nothing to delete
				}
				// Create intermediate map if it doesn't exist for replace/merge
				next = make(map[string]interface{})
				current[part] = next
			}

			nextMap, ok := next.(map[string]interface{})
			if !ok {
				// Cannot traverse into a non-map intermediate path part
				if operation == "delete" {
					// If deleting, and we hit a non-map, the path is effectively non-existent further down
					return data, nil
				}
				return nil, fmt.Errorf("cannot set value at path '%s': part '%s' is not a map", path, part)
			}
			current = nextMap
		}
	}
	// Should not be reached if parts is not empty, but return original data just in case
	return data, nil
}

// --- End Helper ---

// UpdateMetadataForEntity updates the metadata for a specific entity
// operation can be "merge", "replace", or "delete"
// updates is a map where keys can be dot-separated paths (e.g., "a.b.c")
func (m *KnowledgeGraphManager) UpdateMetadataForEntity(entityID string, updates map[string]interface{}, operation string) (types.Entity, error) {
	m.mu.Lock()

	entity, exists := m.graph.Entities[entityID]
	if !exists {
		m.mu.Unlock()
		return types.Entity{}, fmt.Errorf("entity with ID %s not found", entityID)
	}

	// Ensure metadata map exists
	if entity.Metadata == nil {
		if operation == "delete" {
			m.mu.Unlock()
			return entity, nil // Return unchanged entity
		}
		entity.Metadata = make(map[string]interface{})
	}

	originalMetadata := make(map[string]interface{})
	if err := deepCopyMap(entity.Metadata, &originalMetadata); err != nil {
		m.mu.Unlock()
		return types.Entity{}, fmt.Errorf("failed to clone original metadata for rollback: %w", err)
	}

	// Apply updates using the helper function
	var lastErr error
	for path, value := range updates {
		_, err := setNestedValue(entity.Metadata, path, value, operation)
		if err != nil {
			// Store the first error encountered, but continue trying other paths if possible?
			// Or perhaps stop on first error? Stopping is safer.
			lastErr = fmt.Errorf("failed to update path '%s': %w", path, err)
			break // Stop on first error
		}
	}

	if lastErr != nil {
		// Rollback in-memory changes if any update failed
		entity.Metadata = originalMetadata
		m.mu.Unlock()
		return types.Entity{}, lastErr // Return the error that caused the stop
	}

	// Update the entity in the graph map
	m.graph.Entities[entityID] = entity

	// --- Save ---
	// Unlock before saving to allow reads during potentially slow I/O
	m.mu.Unlock()
	err := m.saveGraph()

	if err != nil {
		// Attempt rollback on save error
		m.mu.Lock()
		entity.Metadata = originalMetadata
		m.graph.Entities[entityID] = entity
		m.mu.Unlock()
		return types.Entity{}, fmt.Errorf("failed to save graph after metadata update for entity %s, rollback attempted: %w", entityID, err)
	}

	// Return the updated entity
	return entity, nil
}

// BulkUpdateMetadata applies metadata updates to entities matching filter criteria
func (m *KnowledgeGraphManager) BulkUpdateMetadata(filter types.EntityFilterCriteria, updates map[string]interface{}, operation string) ([]types.Entity, error) {
	m.mu.Lock()

	matchedEntities := make([]types.Entity, 0)
	originalStates := make(map[string]map[string]interface{}) // Store original metadata for rollback

	// --- Pass 1: Filter and Validate ---
	for id, entity := range m.graph.Entities {
		match := true
		if filter.Type != "" && entity.Type != filter.Type {
			match = false
		}
		if filter.NameContains != "" && !strings.Contains(entity.Name, filter.NameContains) {
			match = false
		}
		// Add more filters as needed (e.g., DescriptionContains from original SearchNodes)
		if filter.DescriptionContains != "" && !strings.Contains(entity.Description, filter.DescriptionContains) {
			match = false
		}

		if match {
			// Store original metadata before modification attempts
			originalMetaCopy := make(map[string]interface{})
			if err := deepCopyMap(entity.Metadata, &originalMetaCopy); err != nil {
				m.mu.Unlock()
				return nil, fmt.Errorf("failed to clone original metadata for entity %s during bulk update: %w", id, err)
			}
			originalStates[id] = originalMetaCopy
			matchedEntities = append(matchedEntities, entity) // Add the original entity (will be updated in pass 2)
		}
	}

	if len(matchedEntities) == 0 {
		m.mu.Unlock()
		return []types.Entity{}, nil // No matches, nothing to do
	}

	// --- Pass 2: Apply Updates In Memory ---
	updatedEntitiesResult := make([]types.Entity, 0, len(matchedEntities))
	var firstUpdateError error

	for _, entityToUpdate := range matchedEntities {
		// Ensure metadata map exists if needed
		if entityToUpdate.Metadata == nil {
			if operation == "delete" {
				updatedEntitiesResult = append(updatedEntitiesResult, entityToUpdate) // No change needed
				continue
			}
			entityToUpdate.Metadata = make(map[string]interface{})
		}

		// Apply updates for the current entity
		for path, value := range updates {
			_, err := setNestedValue(entityToUpdate.Metadata, path, value, operation)
			if err != nil {
				firstUpdateError = fmt.Errorf("failed updating path '%s' for entity %s during bulk update: %w", path, entityToUpdate.ID, err)
				goto HandleUpdateError // Use goto to break out and handle rollback/error reporting
			}
		}
		// Update the entity in the main graph map *after* all its updates succeed
		m.graph.Entities[entityToUpdate.ID] = entityToUpdate
		updatedEntitiesResult = append(updatedEntitiesResult, entityToUpdate) // Add successfully updated entity to results
	}

HandleUpdateError:
	if firstUpdateError != nil {
		// Rollback all changes made in this bulk operation
		for id, originalMeta := range originalStates {
			if entity, exists := m.graph.Entities[id]; exists { // Check if entity still exists (should)
				entity.Metadata = originalMeta
				m.graph.Entities[id] = entity
			}
		}
		m.mu.Unlock()
		return nil, firstUpdateError // Return the first error encountered
	}

	// --- Save ---
	// Unlock before saving
	m.mu.Unlock()
	saveErr := m.saveGraph()

	if saveErr != nil {
		// Attempt rollback on save error
		m.mu.Lock()
		for id, originalMeta := range originalStates {
			if entity, exists := m.graph.Entities[id]; exists {
				entity.Metadata = originalMeta
				m.graph.Entities[id] = entity
			}
		}
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to save graph after bulk metadata update, rollback attempted: %w", saveErr)
	}

	// Return the successfully updated entities
	return updatedEntitiesResult, nil
}

// Helper function for deep copying maps (necessary for rollback)
// Uses JSON marshal/unmarshal for simplicity, consider other methods for performance if needed.
func deepCopyMap(src map[string]interface{}, dst *map[string]interface{}) error {
	if src == nil {
		*dst = nil // Handle nil source map
		return nil
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal map for deep copy: %w", err)
	}
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		return fmt.Errorf("failed to unmarshal map for deep copy: %w", err)
	}
	return nil
}
