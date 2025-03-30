package graph_test

import (
	"testing"

	"mcp-memory/internal/types"

	"mcp-memory/internal/graph"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateMetadataForEntity tests the UpdateMetadataForEntity method with various operations and nested paths
func TestUpdateMetadataForEntity(t *testing.T) {
	manager, tmpFile := setupTestManager(t)

	// Create an entity
	entity := types.Entity{
		ID:   "meta_e1",
		Type: "test",
		Name: "Metadata Test Entity",
		Metadata: map[string]interface{}{
			"initial_key": "initial_value",
			"nested": map[string]interface{}{
				"level1": "value1",
			},
			"to_delete":  "delete_me",
			"to_replace": "replace_me",
		},
	}
	_, err := manager.CreateEntities([]types.Entity{entity})
	require.NoError(t, err, "Failed to create entity for metadata test")

	// --- Test Merge (Existing functionality, now with nested) ---
	mergeUpdates := map[string]interface{}{
		"new_key":       "new_value",
		"nested.level2": "value2",        // Add nested
		"initial_key":   "updated_value", // Overwrite existing
	}
	updatedEntity, err := manager.UpdateMetadataForEntity("meta_e1", mergeUpdates, "merge")
	require.NoError(t, err, "Failed to merge metadata")
	assert.Equal(t, "new_value", updatedEntity.Metadata["new_key"], "Merge failed: new_key not added")
	assert.Equal(t, "updated_value", updatedEntity.Metadata["initial_key"], "Merge failed: initial_key not updated")
	nestedMapMerge, _ := updatedEntity.Metadata["nested"].(map[string]interface{})
	require.NotNil(t, nestedMapMerge, "Merge failed: nested map not found")
	assert.Equal(t, "value1", nestedMapMerge["level1"], "Merge failed: nested.level1 changed unexpectedly")
	assert.Equal(t, "value2", nestedMapMerge["level2"], "Merge failed: nested.level2 not added")

	// --- Test Replace --- //
	replaceUpdates := map[string]interface{}{
		"to_replace":      "has_been_replaced", // Replace existing top-level
		"nested.level1":   "replaced_value1",   // Replace nested
		"new_replace_key": "added_by_replace",  // Add new key via replace
	}
	updatedEntity, err = manager.UpdateMetadataForEntity("meta_e1", replaceUpdates, "replace")
	require.NoError(t, err, "Failed to replace metadata")
	assert.Equal(t, "has_been_replaced", updatedEntity.Metadata["to_replace"], "Replace failed: to_replace not updated")
	assert.Equal(t, "added_by_replace", updatedEntity.Metadata["new_replace_key"], "Replace failed: new_replace_key not added")
	nestedMapReplace, _ := updatedEntity.Metadata["nested"].(map[string]interface{})
	require.NotNil(t, nestedMapReplace, "Replace failed: nested map not found after replace")
	assert.Equal(t, "replaced_value1", nestedMapReplace["level1"], "Replace failed: nested.level1 not replaced")
	// Ensure merge additions still exist if not replaced
	assert.Equal(t, "new_value", updatedEntity.Metadata["new_key"], "Replace failed: merged key 'new_key' missing")
	assert.Equal(t, "value2", nestedMapReplace["level2"], "Replace failed: merged key 'nested.level2' missing")

	// --- Test Delete --- //
	deleteUpdates := map[string]interface{}{ // Values are ignored for delete
		"to_delete":        nil,
		"nested.level2":    nil, // Delete nested key
		"non_existent.key": nil, // Delete non-existent path (should be no-op)
	}
	updatedEntity, err = manager.UpdateMetadataForEntity("meta_e1", deleteUpdates, "delete")
	require.NoError(t, err, "Failed to delete metadata")
	_, exists := updatedEntity.Metadata["to_delete"]
	assert.False(t, exists, "Delete failed: to_delete key still exists")
	nestedMapDelete, _ := updatedEntity.Metadata["nested"].(map[string]interface{})
	require.NotNil(t, nestedMapDelete, "Delete failed: nested map not found after delete")
	_, exists = nestedMapDelete["level2"]
	assert.False(t, exists, "Delete failed: nested.level2 key still exists")
	// Ensure other keys remain
	assert.True(t, nestedMapDelete["level1"] == "replaced_value1", "Delete failed: nested.level1 removed unexpectedly")
	assert.True(t, updatedEntity.Metadata["initial_key"] == "updated_value", "Delete failed: initial_key removed unexpectedly")

	// --- Test Error Cases --- //
	// Non-existent entity
	_, err = manager.UpdateMetadataForEntity("non_existent_id", map[string]interface{}{"a": 1}, "merge")
	assert.Error(t, err, "Expected error when updating metadata for non-existent entity")

	// Invalid operation
	_, err = manager.UpdateMetadataForEntity("meta_e1", map[string]interface{}{"a": 1}, "invalid_op")
	assert.Error(t, err, "Expected error for invalid operation")

	// Nested update into non-map
	_, err = manager.UpdateMetadataForEntity("meta_e1", map[string]interface{}{"initial_key.subkey": 1}, "merge")
	assert.Error(t, err, "Expected error when trying to merge into a non-map nested path")

	// --- Test Persistence --- //
	// Create a new manager instance pointing to the *same* file
	newManager := graph.NewKnowledgeGraphManager(tmpFile)

	loadedGraph := newManager.ReadGraph()
	loadedEntity, exists := loadedGraph.Entities["meta_e1"]
	require.True(t, exists, "Entity meta_e1 not found after reloading graph")
	assert.False(t, loadedEntity.Metadata["to_delete"] != nil, "Persistence check failed: deleted key reappeared")
	assert.Equal(t, "has_been_replaced", loadedEntity.Metadata["to_replace"], "Persistence check failed: replaced value incorrect")
	loadedNested, _ := loadedEntity.Metadata["nested"].(map[string]interface{})
	require.NotNil(t, loadedNested)
	assert.Equal(t, "replaced_value1", loadedNested["level1"], "Persistence check failed: nested value incorrect")
	_, level2Exists := loadedNested["level2"]
	assert.False(t, level2Exists, "Persistence check failed: deleted nested key reappeared")
}

// Note: TestBulkUpdateMetadata moved to bulk_metadata_test.go
