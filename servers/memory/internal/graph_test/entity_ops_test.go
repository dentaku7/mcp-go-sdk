package graph_test

import (
	"testing"

	"mcp-memory/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateEntities tests the CreateEntities method
func TestCreateEntities(t *testing.T) {
	manager, _ := setupTestManager(t)

	entities := []types.Entity{
		{
			ID:   "e1",
			Type: "person",
			Name: "John Doe",
		},
		{
			ID:   "e2",
			Type: "person",
			Name: "Jane Smith",
		},
	}

	// Test successful creation
	created, err := manager.CreateEntities(entities)
	require.NoError(t, err, "Failed to create entities")
	assert.Len(t, created, len(entities), "Incorrect number of entities created")

	// Test creating duplicate entity
	_, err = manager.CreateEntities([]types.Entity{entities[0]})
	assert.Error(t, err, "Expected error when creating duplicate entity ID e1")

	// Test auto-generation of IDs
	autoGenEntities := []types.Entity{
		{
			Type: "person",
			Name: "Auto Gen 1",
		},
		{
			Type: "person",
			Name: "Auto Gen 2",
		},
	}

	createdAutoGen, err := manager.CreateEntities(autoGenEntities)
	require.NoError(t, err, "Failed to create entities with auto-generated IDs")
	assert.Len(t, createdAutoGen, len(autoGenEntities), "Incorrect number of auto-gen entities created")

	for _, entity := range createdAutoGen {
		assert.NotEmpty(t, entity.ID, "Expected auto-generated ID to be non-empty")
		// Check it's actually in the graph using ReadGraph
		loadedGraph := manager.ReadGraph()
		_, exists := loadedGraph.Entities[entity.ID]
		assert.True(t, exists, "Auto-generated entity %s not found in graph map", entity.ID)
	}
}

// TestUpdateEntities tests the UpdateEntities batch method
func TestUpdateEntities(t *testing.T) {
	manager, _ := setupTestManager(t)

	// --- Setup --- //
	initialEntities := []types.Entity{
		{
			ID:          "ue1",
			Type:        "widget",
			Name:        "Old Widget 1",
			Description: "Original Desc 1",
			Metadata:    map[string]interface{}{"color": "red", "size": 10},
		},
		{
			ID:          "ue2",
			Type:        "gadget",
			Name:        "Old Gadget 2",
			Description: "Original Desc 2",
			Metadata:    map[string]interface{}{"version": "1.0"},
		},
		{
			ID:          "ue3", // For rollback test
			Type:        "gizmo",
			Name:        "Gizmo 3",
			Description: "Will not be updated",
		},
	}
	_, err := manager.CreateEntities(initialEntities)
	require.NoError(t, err, "Failed to create initial entities for update test")

	// --- Test Successful Batch Update --- //
	updates := []types.Entity{
		{
			ID:       "ue1",
			Name:     "New Widget 1",                     // Update name
			Metadata: map[string]interface{}{"size": 12}, // Update metadata (merge)
			// Description and Type intentionally omitted
		},
		{
			ID:          "ue2",
			Description: "New Desc 2", // Update description
			// Type and Name intentionally omitted
			Metadata: map[string]interface{}{"new_field": true}, // Add metadata
		},
	}

	updated, err := manager.UpdateEntities(updates)
	require.NoError(t, err, "Failed to update entities")
	assert.Len(t, updated, len(updates), "Incorrect number of entities returned from update")

	// Verify ue1 changes (and unchanged fields) using ReadGraph
	loadedGraph := manager.ReadGraph()
	ue1, ok := loadedGraph.Entities["ue1"]
	require.True(t, ok, "Entity ue1 not found after update")
	assert.Equal(t, "New Widget 1", ue1.Name, "ue1 name incorrect")
	assert.Equal(t, "widget", ue1.Type, "ue1 type changed unexpectedly")                        // Should not change
	assert.Equal(t, "Original Desc 1", ue1.Description, "ue1 description changed unexpectedly") // Should not change
	assert.Equal(t, "red", ue1.Metadata["color"], "ue1 metadata 'color' lost")                  // Should be preserved
	assert.EqualValues(t, 12, ue1.Metadata["size"], "ue1 metadata 'size' not updated")          // Should be updated (Use EqualValues for numeric types)

	// Verify ue2 changes (and unchanged fields) using ReadGraph
	loadedGraph = manager.ReadGraph()
	ue2, ok := loadedGraph.Entities["ue2"]
	require.True(t, ok, "Entity ue2 not found after update")
	assert.Equal(t, "Old Gadget 2", ue2.Name, "ue2 name changed unexpectedly") // Should not change
	assert.Equal(t, "gadget", ue2.Type, "ue2 type changed unexpectedly")       // Should not change
	assert.Equal(t, "New Desc 2", ue2.Description, "ue2 description incorrect")
	assert.Equal(t, "1.0", ue2.Metadata["version"], "ue2 metadata 'version' lost")         // Should be preserved
	assert.Equal(t, true, ue2.Metadata["new_field"], "ue2 metadata 'new_field' not added") // Should be added

	// --- Test Error: Update Non-Existent Entity (Rollback) --- //
	badUpdatesNonExistent := []types.Entity{
		{ID: "ue1", Name: "Should Not Apply 1"}, // This change should be rolled back
		{ID: "non-existent-id", Name: "Bad Update"},
	}
	_, err = manager.UpdateEntities(badUpdatesNonExistent)
	assert.Error(t, err, "Expected error when updating non-existent entity in batch")

	// Verify ue1 was NOT updated (still has state from *successful* previous update) using ReadGraph
	loadedGraph = manager.ReadGraph()
	ue1, _ = loadedGraph.Entities["ue1"]
	assert.Equal(t, "New Widget 1", ue1.Name, "ue1 name was updated despite batch failure (non-existent ID)")

	// --- Test Error: Missing ID in Batch (Rollback) --- //
	badUpdatesMissingID := []types.Entity{
		{ID: "ue2", Name: "Should Not Apply 2"}, // This change should be rolled back
		{Name: "Missing ID"},
	}
	_, err = manager.UpdateEntities(badUpdatesMissingID)
	assert.Error(t, err, "Expected error when missing ID in update batch")

	// Verify ue2 was NOT updated (still has state from *successful* previous update) using ReadGraph
	loadedGraph = manager.ReadGraph()
	ue2, _ = loadedGraph.Entities["ue2"]
	assert.Equal(t, "Old Gadget 2", ue2.Name, "ue2 name was updated despite batch failure (missing ID)")
}

// TestDeleteEntities tests deleting entities
func TestDeleteEntities(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Setup
	_, _ = manager.CreateEntities([]types.Entity{
		{ID: "d1"}, {ID: "d2"},
	})
	// Note: Cascading deletes of relations/observations are not tested here,
	// as the current implementation doesn't support them.

	// Test delete existing
	err := manager.DeleteEntities([]string{"d1"})
	assert.NoError(t, err, "Failed to delete entity d1")
	loadedGraph := manager.ReadGraph()
	assert.Len(t, loadedGraph.Entities, 1, "Incorrect entity count after deleting d1")
	_, exists := loadedGraph.Entities["d1"]
	assert.False(t, exists, "Entity d1 still exists after deletion")

	// Test delete multiple
	err = manager.DeleteEntities([]string{"d2"}) // Delete remaining
	assert.NoError(t, err, "Failed to delete entity d2")
	loadedGraph = manager.ReadGraph()
	assert.Empty(t, loadedGraph.Entities, "Entities should be empty after deleting d2")

	// Test delete non-existent
	err = manager.DeleteEntities([]string{"d3"})
	assert.Error(t, err, "Expected error when deleting non-existent entity")
}
