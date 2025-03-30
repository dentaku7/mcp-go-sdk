package graph_test

import (
	"testing"

	"mcp-memory/internal/types"

	// No need to import "mcp-memory/internal/graph" as setupTestManager is likely in another _test.go file in the same package
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBulkUpdateMetadata tests the BulkUpdateMetadata method
func TestBulkUpdateMetadata(t *testing.T) {
	manager, _ := setupTestManager(t) // Assumes setupTestManager is available in the package

	// Create entities for bulk testing
	entities := []types.Entity{
		{ID: "bulk1", Type: "A", Name: "Bulk A One", Description: "First A", Metadata: map[string]interface{}{"common": 1, "unique": "bulk1", "nested": map[string]interface{}{"a": 1}}},
		{ID: "bulk2", Type: "A", Name: "Bulk A Two", Description: "Second A", Metadata: map[string]interface{}{"common": 2, "unique": "bulk2", "nested": map[string]interface{}{"a": 2}}},
		{ID: "bulk3", Type: "B", Name: "Bulk B One", Description: "First B", Metadata: map[string]interface{}{"common": 3, "unique": "bulk3"}},
		{ID: "bulk4", Type: "A", Name: "Bulk A Three (special)", Description: "Third A", Metadata: map[string]interface{}{"common": 4, "unique": "bulk4"}},
	}
	_, err := manager.CreateEntities(entities)
	require.NoError(t, err, "Failed to create entities for bulk update test")

	// --- Test Bulk Merge --- //
	filterA := types.EntityFilterCriteria{Type: "A"}
	mergeUpdates := map[string]interface{}{
		"status":   "merged",
		"common":   99, // Overwrite
		"nested.b": "merged_nested",
	}
	updated, err := manager.BulkUpdateMetadata(filterA, mergeUpdates, "merge")
	require.NoError(t, err, "Bulk merge failed")
	assert.Len(t, updated, 3, "Expected 3 entities of type A to be updated by merge")

	// Check updated entities using ReadGraph
	currentGraph := manager.ReadGraph()
	for _, id := range []string{"bulk1", "bulk2", "bulk4"} {
		entity := currentGraph.Entities[id]
		assert.Equal(t, "merged", entity.Metadata["status"], "Bulk merge failed for %s: status not added/updated", id)
		assert.EqualValues(t, 99, entity.Metadata["common"], "Bulk merge failed for %s: common not overwritten", id) // Use EqualValues for numeric comparison
		if id == "bulk1" || id == "bulk2" {                                                                          // Check nested merge only where it pre-existed
			nested, _ := entity.Metadata["nested"].(map[string]interface{})
			require.NotNil(t, nested, "Bulk merge check failed for %s: nested map lost", id)
			assert.Equal(t, "merged_nested", nested["b"], "Bulk merge failed for %s: nested.b not added", id)
			assert.NotNil(t, nested["a"], "Bulk merge failed for %s: nested.a lost", id)
		}
	}
	// Check non-matching entity was untouched using ReadGraph
	entityB := currentGraph.Entities["bulk3"]
	assert.Nil(t, entityB.Metadata["status"], "Entity bulk3 (type B) should not have been modified by merge")
	assert.EqualValues(t, 3, entityB.Metadata["common"], "Entity bulk3 (type B) common value changed unexpectedly") // Use EqualValues

	// --- Test Bulk Replace --- //
	filterNameContainsBulkA := types.EntityFilterCriteria{NameContains: "Bulk A"}
	replaceUpdates := map[string]interface{}{
		"status":   "replaced",
		"nested.a": "replaced_nested_a", // Replace nested
		"unique":   "replaced_value",    // Replace existing
	}
	updated, err = manager.BulkUpdateMetadata(filterNameContainsBulkA, replaceUpdates, "replace")
	require.NoError(t, err, "Bulk replace failed")
	assert.Len(t, updated, 3, "Expected 3 entities containing 'Bulk A' to be updated by replace")

	// Check updated entities using ReadGraph
	currentGraph = manager.ReadGraph()
	for _, id := range []string{"bulk1", "bulk2", "bulk4"} {
		entity := currentGraph.Entities[id]
		assert.Equal(t, "replaced", entity.Metadata["status"], "Bulk replace failed for %s: status not replaced", id)
		assert.Equal(t, "replaced_value", entity.Metadata["unique"], "Bulk replace failed for %s: unique not replaced", id)
		if id == "bulk1" || id == "bulk2" { // Check nested replace only where it pre-existed
			nested, _ := entity.Metadata["nested"].(map[string]interface{})
			require.NotNil(t, nested, "Bulk replace check failed for %s: nested map lost", id)
			assert.Equal(t, "replaced_nested_a", nested["a"], "Bulk replace failed for %s: nested.a not replaced", id)
			_, bExists := nested["b"] // From previous merge
			assert.True(t, bExists, "Bulk replace failed for %s: nested.b (from merge) lost unexpectedly", id)
		}
		// Check that 'common' field (not in replaceUpdates) is still there
		_, commonExists := entity.Metadata["common"]
		assert.True(t, commonExists, "Bulk replace failed for %s: common key removed unexpectedly", id)
	}

	// --- Test Bulk Delete --- //
	filterDescContainsSecond := types.EntityFilterCriteria{DescriptionContains: "Second"}
	deleteUpdates := map[string]interface{}{ // Values ignored
		"common":   nil,
		"nested.b": nil, // Delete nested key added by merge
	}
	updated, err = manager.BulkUpdateMetadata(filterDescContainsSecond, deleteUpdates, "delete")
	require.NoError(t, err, "Bulk delete failed")
	assert.Len(t, updated, 1, "Expected 1 entity with description containing 'Second' to be updated by delete")
	assert.Equal(t, "bulk2", updated[0].ID, "Incorrect entity matched for bulk delete")

	// Check updated entity using ReadGraph
	currentGraph = manager.ReadGraph()
	entityBulk2 := currentGraph.Entities["bulk2"]
	_, commonExists := entityBulk2.Metadata["common"]
	assert.False(t, commonExists, "Bulk delete failed for bulk2: common key not deleted")
	nestedBulk2, _ := entityBulk2.Metadata["nested"].(map[string]interface{})
	require.NotNil(t, nestedBulk2, "Bulk delete check failed for bulk2: nested map lost")
	_, bExists := nestedBulk2["b"]
	assert.False(t, bExists, "Bulk delete failed for bulk2: nested.b not deleted")
	// Ensure other keys remain
	assert.True(t, nestedBulk2["a"] == "replaced_nested_a", "Bulk delete failed for bulk2: nested.a removed unexpectedly")
	assert.True(t, entityBulk2.Metadata["status"] == "replaced", "Bulk delete failed for bulk2: status removed unexpectedly")

	// --- Test Edge Cases --- //
	// No matches
	filterNoMatch := types.EntityFilterCriteria{Type: "C"}
	updated, err = manager.BulkUpdateMetadata(filterNoMatch, map[string]interface{}{"a": 1}, "merge")
	require.NoError(t, err, "Bulk update with no matches should not error")
	assert.Len(t, updated, 0, "Expected 0 updated entities when filter matches none")

	// Error during update (e.g., nested into non-map)
	// Setup: Update bulk3.common to be a string using UpdateMetadataForEntity
	_, err = manager.UpdateMetadataForEntity("bulk3", map[string]interface{}{"common": "not_a_map"}, "replace")
	require.NoError(t, err, "Failed to setup non-map value for bulk update error test")

	filterB := types.EntityFilterCriteria{Type: "B"}
	updateIntoString := map[string]interface{}{"common.sub": 1}
	_, err = manager.BulkUpdateMetadata(filterB, updateIntoString, "merge")
	assert.Error(t, err, "Expected error when bulk update tries to set nested value into non-map")

	// Check rollback using ReadGraph
	rolledBackGraph := manager.ReadGraph()
	rolledBackEntityB := rolledBackGraph.Entities["bulk3"]
	// Verify the value is back to the state *before* the failed BulkUpdateMetadata call
	assert.Equal(t, "not_a_map", rolledBackEntityB.Metadata["common"], "Rollback check failed after bulk update error: common value is incorrect")
	_, subExists := rolledBackEntityB.Metadata["common.sub"] // Check explicitly if sub-key exists, which it shouldn't
	assert.False(t, subExists, "Rollback check failed: sub-key exists after failed bulk update")
}
