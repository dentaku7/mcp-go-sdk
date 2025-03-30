package graph_test

import (
	"testing"

	"mcp-memory/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddObservations tests the AddObservations method
func TestAddObservations(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Prerequisite: Create an entity
	entities := []types.Entity{
		{ID: "e1", Type: "person", Name: "Observed Person"},
	}
	_, err := manager.CreateEntities(entities)
	require.NoError(t, err, "Failed to create prerequisite entity for observation test")

	// Test successful addition
	observations := []types.Observation{
		{
			ID:       "o1",
			EntityID: "e1",
			Type:     "hobby",
			Content:  "Likes hiking",
		},
	}
	created, err := manager.AddObservations(observations)
	require.NoError(t, err, "Failed to add observations")
	assert.Len(t, created, len(observations), "Incorrect number of observations added")
	// Verify timestamp was automatically set
	assert.False(t, created[0].Timestamp.IsZero(), "Timestamp should have been automatically set")
	// Verify tags are empty by default
	assert.Empty(t, created[0].Tags, "Tags should be empty by default")

	// Test adding observation with tags
	taggedObservations := []types.Observation{
		{
			ID:       "o_tagged",
			EntityID: "e1",
			Type:     "skill",
			Content:  "Go Programming",
			Tags:     []string{"programming", "backend", "golang"},
		},
	}
	createdTagged, err := manager.AddObservations(taggedObservations)
	require.NoError(t, err, "Failed to add tagged observation")
	assert.Len(t, createdTagged, 1, "Incorrect number of tagged observations added")
	assert.Equal(t, []string{"programming", "backend", "golang"}, createdTagged[0].Tags, "Incorrect tags stored")
	assert.False(t, createdTagged[0].Timestamp.IsZero(), "Timestamp should have been automatically set for tagged observation")

	// Test creating duplicate observation ID
	_, err = manager.AddObservations(observations) // Try adding o1 again
	assert.Error(t, err, "Expected error when adding duplicate observation ID o1")

	// Test adding observation to non-existent entity
	_, err = manager.AddObservations([]types.Observation{{
		ID:       "o2",
		EntityID: "e3", // Non-existent
		Type:     "hobby",
		Content:  "Likes swimming",
	}})
	assert.Error(t, err, "Expected error when adding observation to non-existent entity")

	// Test auto-generation of IDs
	autoGenObservations := []types.Observation{
		{
			EntityID: "e1",
			Type:     "skill",
			Content:  "Expert in Python programming",
		},
	}
	createdAutoGen, err := manager.AddObservations(autoGenObservations)
	require.NoError(t, err, "Failed to add observations with auto-generated IDs")
	assert.Len(t, createdAutoGen, len(autoGenObservations), "Incorrect number of auto-gen observations added")

	for _, observation := range createdAutoGen {
		assert.NotEmpty(t, observation.ID, "Expected auto-generated observation ID to be non-empty")
		// Check it's actually in the graph using ReadGraph
		loadedGraph := manager.ReadGraph()
		_, exists := loadedGraph.Observations[observation.ID]
		assert.True(t, exists, "Auto-generated observation %s not found in graph map", observation.ID)
	}
}

// TestDeleteObservations tests the DeleteObservations method
func TestDeleteObservations(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Setup
	_, _ = manager.CreateEntities([]types.Entity{{ID: "e1"}})
	_, _ = manager.AddObservations([]types.Observation{
		{ID: "o1", EntityID: "e1"}, {ID: "o2", EntityID: "e1"},
	})

	// Test delete existing
	err := manager.DeleteObservations([]string{"o1"})
	assert.NoError(t, err, "Failed to delete observation o1")
	loadedGraph := manager.ReadGraph()
	assert.Len(t, loadedGraph.Observations, 1, "Incorrect observation count after deleting o1")
	_, exists := loadedGraph.Observations["o1"]
	assert.False(t, exists, "Observation o1 still exists after deletion")

	// Test delete multiple
	err = manager.DeleteObservations([]string{"o2"}) // Delete remaining
	assert.NoError(t, err, "Failed to delete observation o2")
	loadedGraph = manager.ReadGraph()
	assert.Empty(t, loadedGraph.Observations, "Observations should be empty after deleting o2")

	// Test delete non-existent
	err = manager.DeleteObservations([]string{"o3"})
	assert.Error(t, err, "Expected error when deleting non-existent observation")
}
