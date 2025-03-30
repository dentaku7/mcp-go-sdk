package graph

import (
	"encoding/json"
	"fmt"
	"os"
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
	// If filePath is empty, do not attempt to save (for testing or in-memory mode)
	if m.filePath == "" {
		return nil
	}

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

// ReadGraph returns the current state of the graph
func (m *KnowledgeGraphManager) ReadGraph() types.KnowledgeGraph {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.graph
}
