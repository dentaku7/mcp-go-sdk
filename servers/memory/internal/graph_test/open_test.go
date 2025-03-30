package graph_test

import (
	"testing"

	"mcp-memory/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenNodes tests retrieving a node and its immediate relations
func TestOpenNodes(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Setup
	_, _ = manager.CreateEntities([]types.Entity{
		{ID: "op1"}, {ID: "op2"}, {ID: "op3"}, {ID: "op4"},
	})
	_, _ = manager.CreateRelations([]types.Relation{
		{ID: "opr1", Source: "op1", Target: "op2"}, // outgoing from op1
		{ID: "opr2", Source: "op3", Target: "op1"}, // incoming to op1
		{ID: "opr3", Source: "op2", Target: "op3"}, // unrelated to op1 direct open
	})

	// Test opening single node
	result, err := manager.OpenNodes([]string{"op1"})
	require.NoError(t, err, "Failed to open node op1")

	assert.Len(t, result.Entities, 1, "OpenNodes: Expected 1 entity")
	assert.Contains(t, result.Entities, "op1", "OpenNodes: Result should contain op1")

	assert.Len(t, result.Relations, 2, "OpenNodes: Expected 2 relations")
	assert.Contains(t, result.Relations, "opr1", "OpenNodes: Result should contain relation opr1")
	assert.Contains(t, result.Relations, "opr2", "OpenNodes: Result should contain relation opr2")
	assert.NotContains(t, result.Relations, "opr3", "OpenNodes: Result should not contain relation opr3")

	// Test opening non-existent node
	_, err = manager.OpenNodes([]string{"non-existent"})
	assert.Error(t, err, "Expected error when opening non-existent node")
}
