package graph_test

import (
	"testing"

	"mcp-memory/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateRelations tests the CreateRelations method
func TestCreateRelations(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Prerequisite: Create source and target entities
	entities := []types.Entity{
		{ID: "e1", Type: "person", Name: "Source Person"},
		{ID: "e2", Type: "person", Name: "Target Person"},
	}
	_, err := manager.CreateEntities(entities)
	require.NoError(t, err, "Failed to create prerequisite entities for relation test")

	// Test successful creation
	relations := []types.Relation{
		{
			ID:     "r1",
			Type:   "knows",
			Source: "e1",
			Target: "e2",
		},
	}
	created, err := manager.CreateRelations(relations)
	require.NoError(t, err, "Failed to create relations")
	assert.Len(t, created, len(relations), "Incorrect number of relations created")
	// Check default values (optional, but good practice if defaults are expected)
	assert.Equal(t, 0.0, created[0].Weight, "Default weight should be 0.0")
	assert.False(t, created[0].Bidirectional, "Default bidirectional should be false")

	// Test creation with specific attributes
	weightedRelations := []types.Relation{
		{
			ID:            "r_weighted",
			Type:          "distance",
			Source:        "e1",
			Target:        "e2",
			Weight:        12.5,
			Bidirectional: true,
		},
	}
	createdWeighted, err := manager.CreateRelations(weightedRelations)
	require.NoError(t, err, "Failed to create weighted relation")
	assert.Len(t, createdWeighted, 1, "Incorrect number of weighted relations created")
	assert.Equal(t, 12.5, createdWeighted[0].Weight, "Incorrect weight stored")
	assert.True(t, createdWeighted[0].Bidirectional, "Incorrect bidirectional stored")

	// Test creating duplicate relation ID
	_, err = manager.CreateRelations(relations) // Try creating r1 again
	assert.Error(t, err, "Expected error when creating duplicate relation ID r1")

	// Test creating relation with non-existent entity (source)
	_, err = manager.CreateRelations([]types.Relation{{
		ID:     "r2",
		Type:   "works_with",
		Source: "e3", // Non-existent
		Target: "e1",
	}})
	assert.Error(t, err, "Expected error when creating relation with non-existent source entity")

	// Test creating relation with non-existent entity (target)
	_, err = manager.CreateRelations([]types.Relation{{
		ID:     "r3",
		Type:   "works_with",
		Source: "e1",
		Target: "e4", // Non-existent
	}})
	assert.Error(t, err, "Expected error when creating relation with non-existent target entity")

	// Test auto-generation of IDs
	autoGenRelations := []types.Relation{
		{
			Type:   "friend_of",
			Source: "e1",
			Target: "e2",
		},
	}
	createdAutoGen, err := manager.CreateRelations(autoGenRelations)
	require.NoError(t, err, "Failed to create relations with auto-generated IDs")
	assert.Len(t, createdAutoGen, len(autoGenRelations), "Incorrect number of auto-gen relations created")

	for _, relation := range createdAutoGen {
		assert.NotEmpty(t, relation.ID, "Expected auto-generated relation ID to be non-empty")
		// Check it's actually in the graph using ReadGraph
		loadedGraph := manager.ReadGraph()
		_, exists := loadedGraph.Relations[relation.ID]
		assert.True(t, exists, "Auto-generated relation %s not found in graph map", relation.ID)
	}
}

// TestDeleteRelations tests deleting relations
func TestDeleteRelations(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Setup
	_, _ = manager.CreateEntities([]types.Entity{{ID: "e1"}, {ID: "e2"}})
	relations := []types.Relation{
		{ID: "r1", Source: "e1", Target: "e2", Type: "knows"},
		{ID: "r2", Source: "e2", Target: "e1", Type: "works_with"},
	}
	_, _ = manager.CreateRelations(relations)

	// Test delete existing by ID
	err := manager.DeleteRelations([]types.Relation{{ID: "r1"}})
	assert.NoError(t, err, "Failed to delete relation r1 by ID")
	loadedGraph := manager.ReadGraph()
	assert.Len(t, loadedGraph.Relations, 1, "Incorrect relation count after deleting r1")
	_, exists := loadedGraph.Relations["r1"]
	assert.False(t, exists, "Relation r1 still exists after deletion by ID")

	// Test delete existing by Source/Target/Type (when ID omitted)
	err = manager.DeleteRelations([]types.Relation{{
		Source: "e2",
		Target: "e1",
		Type:   "works_with",
	}})
	assert.NoError(t, err, "Failed to delete relation r2 by fields")
	loadedGraph = manager.ReadGraph()
	assert.Empty(t, loadedGraph.Relations, "Relations should be empty after deleting r2 by fields")

	// Test delete non-existent by ID (should not error, just do nothing)
	_, _ = manager.CreateRelations([]types.Relation{{ID: "r3", Source: "e1", Target: "e2"}}) // Add one back
	err = manager.DeleteRelations([]types.Relation{{ID: "r4"}})
	assert.NoError(t, err, "Deleting non-existent relation by ID should not error")
	loadedGraph = manager.ReadGraph()
	assert.Len(t, loadedGraph.Relations, 1, "Relation count changed after attempting to delete non-existent r4 by ID")

	// Test delete non-existent by fields (should not error)
	err = manager.DeleteRelations([]types.Relation{{
		Source: "e1",
		Target: "e2",
		Type:   "unknown_type",
	}})
	assert.NoError(t, err, "Deleting non-existent relation by fields should not error")
	loadedGraph = manager.ReadGraph()
	assert.Len(t, loadedGraph.Relations, 1, "Relation count changed after attempting to delete non-existent relation by fields")
}
