package graph_test

import (
	"testing"

	"mcp-memory/internal/types"

	"github.com/stretchr/testify/assert"
)

// TestSearchNodes tests searching by type and metadata
func TestSearchNodes(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Setup
	entities := []types.Entity{
		{ID: "s1", Type: "person", Metadata: map[string]interface{}{"role": "dev"}},
		{ID: "s2", Type: "person", Metadata: map[string]interface{}{"role": "qa"}},
		{ID: "s3", Type: "org"},
	}
	_, _ = manager.CreateEntities(entities)

	// Test: Search by type
	resultsType := manager.SearchNodes("person", nil)
	assert.Len(t, resultsType, 2, "SearchNodes by type 'person' failed")

	// Test: Search by metadata
	resultsMeta := manager.SearchNodes("", map[string]interface{}{"role": "dev"})
	assert.Len(t, resultsMeta, 1, "SearchNodes by metadata 'role:dev' failed")
	assert.Equal(t, "s1", resultsMeta[0].ID, "SearchNodes metadata returned wrong entity")

	// Test: Search by type and metadata
	resultsBoth := manager.SearchNodes("person", map[string]interface{}{"role": "qa"})
	assert.Len(t, resultsBoth, 1, "SearchNodes by type and metadata failed")
	assert.Equal(t, "s2", resultsBoth[0].ID, "SearchNodes type+metadata returned wrong entity")

	// Test: No results
	resultsNone := manager.SearchNodes("company", nil)
	assert.Empty(t, resultsNone, "SearchNodes expected no results for type 'company'")
}

// TestSearchByText tests the text search across different fields
func TestSearchByText(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Setup
	_, _ = manager.CreateEntities([]types.Entity{
		{ID: "txt1", Type: "doc", Name: "Alpha Report", Description: "Contains beta info"},
		{ID: "txt2", Type: "log", Name: "System Log", Description: "Critical system events"},
	})
	_, _ = manager.CreateRelations([]types.Relation{
		{ID: "txtr1", Source: "txt1", Target: "txt2", Type: "references", Description: "Report references log"},
	})
	_, _ = manager.AddObservations([]types.Observation{
		{ID: "txto1", EntityID: "txt1", Type: "status", Content: "Beta version", Description: "Observation about alpha"},
	})

	tests := []struct {
		name     string
		query    string
		expected int // Expected number of matches (simple count for now)
	}{
		{"Entity Name", "alpha", 1},
		{"Entity Description", "beta", 1},
		{"Entity Type", "log", 2},                     // txt2 (type), txt1 (relation desc contains 'log')
		{"Relation Type", "references", 2},            // txt1, txt2 (relation type)
		{"Relation Description", "references log", 2}, // txt1, txt2 (relation desc)
		{"Observation Type", "status", 1},
		{"Observation Content", "version", 1},
		{"Observation Description", "observation", 1},
		{"Partial Match", "rep", 2}, // Alpha Report, Report references log
		{"Case Insensitive Name", "ALPHA", 1},
		{"Case Insensitive Content", "BETA", 1},
		{"No Match", "gamma", 0},
		{"Empty Query", "", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := manager.SearchByText(tc.query)
			// For simplicity, just check count. A more thorough test might check IDs.
			assert.Len(t, results, tc.expected, "Query: %s", tc.query)
		})
	}
}
