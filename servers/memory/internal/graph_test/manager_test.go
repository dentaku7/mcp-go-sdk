package graph_test

import (
	"os"
	"path/filepath"
	"testing"

	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"

	"github.com/stretchr/testify/assert"
)

// setupTestManager creates a new KnowledgeGraphManager with a temporary file for testing.
// It returns the manager and a cleanup function to remove the temp file.
func setupTestManager(t *testing.T) (*graph.KnowledgeGraphManager, string) {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_memory.json")
	manager := graph.NewKnowledgeGraphManager(tmpFile)

	// Ensure the temp file is removed after the test function completes
	t.Cleanup(func() { os.Remove(tmpFile) })

	return manager, tmpFile
}

// Test file persistence
func TestKnowledgeGraphPersistence(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_memory.json")

	// Create a new manager instance
	manager1 := graph.NewKnowledgeGraphManager(tmpFile)

	// Add some data
	entities := []types.Entity{
		{
			ID:   "e1",
			Type: "person",
			Name: "John Doe",
		},
	}

	_, err := manager1.CreateEntities(entities)
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a new manager instance with the same file
	manager2 := graph.NewKnowledgeGraphManager(tmpFile)

	// Check if data persisted using ReadGraph
	loadedGraph := manager2.ReadGraph()
	if len(loadedGraph.Entities) != 1 {
		t.Errorf("Expected 1 entity after loading from file, got %d", len(loadedGraph.Entities))
	}

	if entity, exists := loadedGraph.Entities["e1"]; !exists {
		t.Error("Entity e1 not found after loading from file")
	} else if entity.Name != "John Doe" {
		t.Errorf("Expected entity name 'John Doe', got '%s'", entity.Name)
	}

	// Clean up - managed by t.Cleanup now, but explicit Remove is fine too if needed outside setup.
	// os.Remove(tmpFile)
}

// Test error cases
func TestKnowledgeGraphErrors(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_memory.json")

	// Create a new manager instance
	manager := graph.NewKnowledgeGraphManager(tmpFile)

	// Test creating relation with empty source/target
	t.Run("CreateRelationEmptySource", func(t *testing.T) {
		_, err := manager.CreateRelations([]types.Relation{{
			Type:   "knows",
			Source: "",
			Target: "e2",
		}})
		if err == nil {
			t.Error("Expected error when creating relation with empty source")
		}
	})

	// Test creating observation with empty entity ID
	t.Run("CreateObservationEmptyEntityID", func(t *testing.T) {
		_, err := manager.AddObservations([]types.Observation{{
			Type:    "hobby",
			Content: "Likes hiking",
		}})
		if err == nil {
			t.Error("Expected error when creating observation with empty entity ID")
		}
	})

	// Clean up - managed by t.Cleanup
	// os.Remove(tmpFile)
}

// TestReadGraph tests the ReadGraph method
func TestReadGraph(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Setup: Add some data
	entities := []types.Entity{{ID: "e1"}, {ID: "e2"}}
	relations := []types.Relation{{ID: "r1", Source: "e1", Target: "e2"}}
	observations := []types.Observation{{ID: "o1", EntityID: "e1"}}

	_, _ = manager.CreateEntities(entities)
	_, _ = manager.CreateRelations(relations)
	_, _ = manager.AddObservations(observations)

	// Test
	loadedGraph := manager.ReadGraph()

	assert.Len(t, loadedGraph.Entities, 2, "ReadGraph: Incorrect number of entities")
	assert.Len(t, loadedGraph.Relations, 1, "ReadGraph: Incorrect number of relations")
	assert.Len(t, loadedGraph.Observations, 1, "ReadGraph: Incorrect number of observations")
}
