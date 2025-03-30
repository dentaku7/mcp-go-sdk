package graph

import (
	"fmt"
	"time"

	"mcp-memory/internal/types"
)

// AddObservations adds new observations to the graph
func (m *KnowledgeGraphManager) AddObservations(observations []types.Observation) ([]types.Observation, error) {
	m.mu.Lock()
	createdObservations := make([]types.Observation, 0, len(observations))
	for _, observation := range observations {
		// Auto-generate ID if not provided
		if observation.ID == "" {
			observation.ID = m.generateID()
		}

		// Automatically set timestamp if it's zero
		if observation.Timestamp.IsZero() {
			observation.Timestamp = time.Now()
		}

		if _, exists := m.graph.Observations[observation.ID]; exists {
			m.mu.Unlock()
			return nil, fmt.Errorf("observation with ID %s already exists", observation.ID)
		}
		if _, exists := m.graph.Entities[observation.EntityID]; !exists {
			m.mu.Unlock()
			return nil, fmt.Errorf("entity %s does not exist", observation.EntityID)
		}
		m.graph.Observations[observation.ID] = observation
		createdObservations = append(createdObservations, observation)
	}
	m.mu.Unlock()

	if err := m.saveGraph(); err != nil {
		return nil, err
	}

	return createdObservations, nil
}

// DeleteObservations removes observations from the graph
func (m *KnowledgeGraphManager) DeleteObservations(ids []string) error {
	m.mu.Lock()
	for _, id := range ids {
		if _, exists := m.graph.Observations[id]; !exists {
			m.mu.Unlock()
			return fmt.Errorf("observation with ID %s does not exist", id)
		}
		delete(m.graph.Observations, id)
	}
	m.mu.Unlock()

	if err := m.saveGraph(); err != nil {
		return err
	}

	return nil
}
