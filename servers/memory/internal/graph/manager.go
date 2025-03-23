package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"mcp-memory/internal/types"

	"github.com/google/uuid"
)

// KnowledgeGraphManager manages the knowledge graph operations
type KnowledgeGraphManager struct {
	filePath string
	graph    types.KnowledgeGraph
	mu       sync.RWMutex
}

// NewKnowledgeGraphManager creates a new instance of KnowledgeGraphManager
func NewKnowledgeGraphManager(filePath string) *KnowledgeGraphManager {
	manager := &KnowledgeGraphManager{
		filePath: filePath,
		graph: types.KnowledgeGraph{
			Entities:     make(map[string]types.Entity),
			Relations:    make(map[string]types.Relation),
			Observations: make(map[string]types.Observation),
		},
	}
	manager.loadGraph()
	return manager
}

// loadGraph loads the graph from the file
func (m *KnowledgeGraphManager) loadGraph() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, start with empty graph
		}
		return fmt.Errorf("failed to read graph file: %w", err)
	}

	if err := json.Unmarshal(data, &m.graph); err != nil {
		return fmt.Errorf("failed to parse graph data: %w", err)
	}

	return nil
}

// saveGraph saves the graph to the file
func (m *KnowledgeGraphManager) saveGraph() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(m.graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal graph data: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write graph file: %w", err)
	}

	return nil
}

// generateID generates a new unique ID
func (m *KnowledgeGraphManager) generateID() string {
	return uuid.New().String()
}

// CreateEntities adds new entities to the graph
func (m *KnowledgeGraphManager) CreateEntities(entities []types.Entity) ([]types.Entity, error) {
	createdEntities := make([]types.Entity, 0, len(entities))
	for _, entity := range entities {
		// Auto-generate ID if not provided
		if entity.ID == "" {
			entity.ID = m.generateID()
		}

		if _, exists := m.graph.Entities[entity.ID]; exists {
			return nil, fmt.Errorf("entity with ID %s already exists", entity.ID)
		}
		m.graph.Entities[entity.ID] = entity
		createdEntities = append(createdEntities, entity)
	}

	if err := m.saveGraph(); err != nil {
		return nil, err
	}

	return createdEntities, nil
}

// CreateRelations adds new relations to the graph
func (m *KnowledgeGraphManager) CreateRelations(relations []types.Relation) ([]types.Relation, error) {
	createdRelations := make([]types.Relation, 0, len(relations))
	for _, relation := range relations {
		// Auto-generate ID if not provided
		if relation.ID == "" {
			relation.ID = m.generateID()
		}

		if _, exists := m.graph.Relations[relation.ID]; exists {
			return nil, fmt.Errorf("relation with ID %s already exists", relation.ID)
		}
		if _, exists := m.graph.Entities[relation.Source]; !exists {
			return nil, fmt.Errorf("source entity %s does not exist", relation.Source)
		}
		if _, exists := m.graph.Entities[relation.Target]; !exists {
			return nil, fmt.Errorf("target entity %s does not exist", relation.Target)
		}
		m.graph.Relations[relation.ID] = relation
		createdRelations = append(createdRelations, relation)
	}

	if err := m.saveGraph(); err != nil {
		return nil, err
	}

	return createdRelations, nil
}

// AddObservations adds new observations to the graph
func (m *KnowledgeGraphManager) AddObservations(observations []types.Observation) ([]types.Observation, error) {
	createdObservations := make([]types.Observation, 0, len(observations))
	for _, observation := range observations {
		// Auto-generate ID if not provided
		if observation.ID == "" {
			observation.ID = m.generateID()
		}

		if _, exists := m.graph.Observations[observation.ID]; exists {
			return nil, fmt.Errorf("observation with ID %s already exists", observation.ID)
		}
		if _, exists := m.graph.Entities[observation.EntityID]; !exists {
			return nil, fmt.Errorf("entity %s does not exist", observation.EntityID)
		}
		m.graph.Observations[observation.ID] = observation
		createdObservations = append(createdObservations, observation)
	}

	if err := m.saveGraph(); err != nil {
		return nil, err
	}

	return createdObservations, nil
}

// DeleteEntities removes entities from the graph
func (m *KnowledgeGraphManager) DeleteEntities(ids []string) error {
	for _, id := range ids {
		if _, exists := m.graph.Entities[id]; !exists {
			return fmt.Errorf("entity with ID %s does not exist", id)
		}
		delete(m.graph.Entities, id)
	}

	if err := m.saveGraph(); err != nil {
		return err
	}

	return nil
}

// DeleteRelations removes relations from the graph
func (m *KnowledgeGraphManager) DeleteRelations(relations []types.Relation) error {
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

	if err := m.saveGraph(); err != nil {
		return err
	}

	return nil
}

// DeleteObservations removes observations from the graph
func (m *KnowledgeGraphManager) DeleteObservations(ids []string) error {
	for _, id := range ids {
		if _, exists := m.graph.Observations[id]; !exists {
			return fmt.Errorf("observation with ID %s does not exist", id)
		}
		delete(m.graph.Observations, id)
	}

	if err := m.saveGraph(); err != nil {
		return err
	}

	return nil
}

// ReadGraph returns the current state of the graph
func (m *KnowledgeGraphManager) ReadGraph() types.KnowledgeGraph {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.graph
}

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

	return results
}

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

	// Add relations where either source or target is in the result
	for id, relation := range m.graph.Relations {
		if idSet[relation.Source] || idSet[relation.Target] {
			result.Relations[id] = relation
		}
	}

	return result, nil
}
