package graph_test

import (
	"mcp-memory/internal/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define the common test data set used by query tests
var testEntities = []types.Entity{
	{ID: "e1", Type: "Person", Name: "Alice", Metadata: map[string]interface{}{"city": "New York", "age": 30.0}}, // Use float64 for JSON numbers
	{ID: "e2", Type: "Person", Name: "Bob", Metadata: map[string]interface{}{"city": "London", "age": 25.0}},
	{ID: "e3", Type: "Company", Name: "Acme Inc", Metadata: map[string]interface{}{"city": "New York"}},
	{ID: "e4", Type: "Person", Name: "Charlie", Metadata: map[string]interface{}{"city": "New York", "age": 35.0}},
}

var testRelations = []types.Relation{
	{ID: "r1", Type: "WORKS_AT", Source: "e1", Target: "e3", Metadata: map[string]interface{}{"role": "Engineer"}},
	{ID: "r2", Type: "FRIENDS_WITH", Source: "e1", Target: "e2"},
	{ID: "r3", Type: "WORKS_AT", Source: "e4", Target: "e3", Metadata: map[string]interface{}{"role": "Manager"}},
}

func TestQueryEntities(t *testing.T) {
	manager, _ := setupTestManager(t)

	_, err := manager.CreateEntities(testEntities)
	require.NoError(t, err, "Setup: Failed to create test entities")

	tests := []struct {
		name              string
		input             types.QueryInput
		expectedCount     int      // Count before pagination
		expectedResultIDs []string // IDs in the final result page
		expectError       bool
	}{
		{
			name: "Filter by Type Person",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "type", Operator: types.OperatorEqual, Value: "Person"}},
			},
			expectedCount:     3,
			expectedResultIDs: []string{"e1", "e2", "e4"}, // Order might vary without sort
		},
		{
			name: "Filter by Metadata City NY",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.city", Operator: types.OperatorEqual, Value: "New York"}},
			},
			expectedCount:     3,
			expectedResultIDs: []string{"e1", "e3", "e4"},
		},
		{
			name: "Filter by Name Bob",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "name", Operator: types.OperatorEqual, Value: "Bob"}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e2"},
		},
		{
			name: "Filter by ID e3",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "id", Operator: types.OperatorEqual, Value: "e3"}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e3"},
		},
		{
			name: "Filter Not Person",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "type", Operator: types.OperatorNotEqual, Value: "Person"}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e3"},
		},
		{
			name: "Filter Multiple Conditions (Person in NY)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters: []types.Filter{
					{Field: "type", Operator: types.OperatorEqual, Value: "Person"},
					{Field: "metadata.city", Operator: types.OperatorEqual, Value: "New York"},
				},
			},
			expectedCount:     2,
			expectedResultIDs: []string{"e1", "e4"},
		},
		{
			name: "Sort by Name Asc",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				SortBy:     "name",
				SortOrder:  types.SortOrderAsc,
			},
			expectedCount:     4,
			expectedResultIDs: []string{"e3", "e1", "e2", "e4"}, // Acme, Alice, Bob, Charlie
		},
		{
			name: "Sort by Metadata Age Desc",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "type", Operator: types.OperatorEqual, Value: "Person"}}, // Only people have age
				SortBy:     "metadata.age",
				SortOrder:  types.SortOrderDesc,
			},
			expectedCount:     3,
			expectedResultIDs: []string{"e4", "e1", "e2"}, // Charlie(35), Alice(30), Bob(25)
		},
		{
			name: "Pagination Limit 2",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				SortBy:     "id", // Ensure consistent order for pagination test
				SortOrder:  types.SortOrderAsc,
				Limit:      2,
			},
			expectedCount:     4,
			expectedResultIDs: []string{"e1", "e2"},
		},
		{
			name: "Pagination Limit 2 Offset 1",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				SortBy:     "id",
				SortOrder:  types.SortOrderAsc,
				Limit:      2,
				Offset:     1,
			},
			expectedCount:     4,
			expectedResultIDs: []string{"e2", "e3"},
		},
		{
			name: "Filter, Sort, and Paginate",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.city", Operator: types.OperatorEqual, Value: "New York"}},
				SortBy:     "name",
				SortOrder:  types.SortOrderDesc,
				Limit:      1,
				Offset:     1,
			},
			expectedCount:     3,              // e1, e3, e4 are in NY. Sorted Desc by name: Charlie, Alice, Acme.
			expectedResultIDs: []string{"e1"}, // Skip 1 (Charlie), take 1 (Alice)
		},
		{
			name: "No Matching Filter",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "name", Operator: types.OperatorEqual, Value: "DoesNotExist"}},
			},
			expectedCount:     0,
			expectedResultIDs: []string{},
		},
		{
			name: "Invalid Target Type",
			input: types.QueryInput{
				TargetType: "invalid",
			},
			expectError: true,
		},
		{
			name: "Invalid Filter Operator",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "name", Operator: "invalid", Value: "Alice"}},
			},
			expectError: true,
		},
		// --- New Operator Tests ---
		{
			name: "Operator IN (City)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.city", Operator: types.OperatorIn, Value: []interface{}{"London", "Paris"}}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e2"},
		},
		{
			name: "Operator IN (Age)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.age", Operator: types.OperatorIn, Value: []interface{}{25.0, 35.0, 40}}}, // Mix float/int
			},
			expectedCount:     2,
			expectedResultIDs: []string{"e2", "e4"},
		},
		{
			name: "Operator NIN (City)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.city", Operator: types.OperatorNotIn, Value: []interface{}{"New York"}}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e2"}, // Only Bob is not in NY
		},
		{
			name: "Operator Contains (Name)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "name", Operator: types.OperatorContains, Value: "Inc"}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e3"},
		},
		{
			name: "Operator GT (Age)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.age", Operator: types.OperatorGreaterThan, Value: 30.0}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e4"}, // Charlie (35)
		},
		{
			name: "Operator GTE (Age)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.age", Operator: types.OperatorGreaterThanOrEqual, Value: 30}}, // Use int
			},
			expectedCount:     2,
			expectedResultIDs: []string{"e1", "e4"}, // Alice (30), Charlie (35)
		},
		{
			name: "Operator LT (Age)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.age", Operator: types.OperatorLessThan, Value: 30.0}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"e2"}, // Bob (25)
		},
		{
			name: "Operator LTE (Age)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "metadata.age", Operator: types.OperatorLessThanOrEqual, Value: 30}}, // Use int
			},
			expectedCount:     2,
			expectedResultIDs: []string{"e1", "e2"}, // Alice (30), Bob (25)
		},
		// --- Improved Sorting Tests ---
		{
			name: "Sort by Metadata Age Asc (Numeric)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				Filters:    []types.Filter{{Field: "type", Operator: types.OperatorEqual, Value: "Person"}}, // Only people have age
				SortBy:     "metadata.age",
				SortOrder:  types.SortOrderAsc,
			},
			expectedCount:     3,
			expectedResultIDs: []string{"e2", "e1", "e4"}, // Bob(25), Alice(30), Charlie(35) - Correct numeric order
		},
		{
			name: "Sort by Metadata City Desc (String)",
			input: types.QueryInput{
				TargetType: types.QueryTargetEntity,
				SortBy:     "metadata.city",
				SortOrder:  types.SortOrderDesc,
			},
			expectedCount:     4,
			expectedResultIDs: []string{"e1", "e3", "e4", "e2"}, // New York, New York, New York, London
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := manager.Query(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, output.Total, "Total count mismatch")
			assert.Len(t, output.Results, len(tt.expectedResultIDs), "Result page length mismatch")

			// Extract IDs from results for comparison
			resultIDs := make([]string, len(output.Results))
			for i, res := range output.Results {
				if entity, ok := res.(types.Entity); ok {
					resultIDs[i] = entity.ID
				} else {
					t.Fatalf("Result item is not an Entity: %T", res)
				}
			}

			// Use ElementsMatch for slices where order might not be guaranteed (unless sorted)
			if tt.input.SortBy == "" && len(tt.expectedResultIDs) > 1 {
				assert.ElementsMatch(t, tt.expectedResultIDs, resultIDs, "Result IDs mismatch (order ignored)")
			} else {
				assert.Equal(t, tt.expectedResultIDs, resultIDs, "Result IDs mismatch (order matters)")
			}
		})
	}
}

func TestQueryRelations(t *testing.T) {
	manager, _ := setupTestManager(t)

	// ---> ADD THIS: Populate data needed for relation queries <---
	_, err := manager.CreateEntities(testEntities) // Relations need entities to exist
	require.NoError(t, err, "Setup: Failed to create test entities for relations test")
	_, err = manager.CreateRelations(testRelations)
	require.NoError(t, err, "Setup: Failed to create test relations")
	// <--- END ADD ---

	tests := []struct {
		name              string
		input             types.QueryInput
		expectedCount     int
		expectedResultIDs []string
		expectError       bool
	}{
		{
			name: "Filter by Type WORKS_AT",
			input: types.QueryInput{
				TargetType: types.QueryTargetRelation,
				Filters:    []types.Filter{{Field: "type", Operator: types.OperatorEqual, Value: "WORKS_AT"}},
			},
			expectedCount:     2,
			expectedResultIDs: []string{"r1", "r3"},
		},
		{
			name: "Filter by Source e1",
			input: types.QueryInput{
				TargetType: types.QueryTargetRelation,
				Filters:    []types.Filter{{Field: "source", Operator: types.OperatorEqual, Value: "e1"}},
			},
			expectedCount:     2,
			expectedResultIDs: []string{"r1", "r2"},
		},
		{
			name: "Filter by Target e3",
			input: types.QueryInput{
				TargetType: types.QueryTargetRelation,
				Filters:    []types.Filter{{Field: "target", Operator: types.OperatorEqual, Value: "e3"}},
			},
			expectedCount:     2,
			expectedResultIDs: []string{"r1", "r3"},
		},
		{
			name: "Filter by Metadata Role Engineer",
			input: types.QueryInput{
				TargetType: types.QueryTargetRelation,
				Filters:    []types.Filter{{Field: "metadata.role", Operator: types.OperatorEqual, Value: "Engineer"}},
			},
			expectedCount:     1,
			expectedResultIDs: []string{"r1"},
		},
		{
			name: "Sort by ID Desc, Limit 2",
			input: types.QueryInput{
				TargetType: types.QueryTargetRelation,
				SortBy:     "id",
				SortOrder:  types.SortOrderDesc,
				Limit:      2,
			},
			expectedCount:     3,
			expectedResultIDs: []string{"r3", "r2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := manager.Query(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, output.Total)
			assert.Len(t, output.Results, len(tt.expectedResultIDs))

			resultIDs := make([]string, len(output.Results))
			for i, res := range output.Results {
				if relation, ok := res.(types.Relation); ok {
					resultIDs[i] = relation.ID
				} else {
					t.Fatalf("Result item is not a Relation: %T", res)
				}
			}

			if tt.input.SortBy == "" && len(tt.expectedResultIDs) > 1 {
				assert.ElementsMatch(t, tt.expectedResultIDs, resultIDs)
			} else {
				assert.Equal(t, tt.expectedResultIDs, resultIDs)
			}
		})
	}
}
