package graph_test

import (
	"fmt"
	"testing"

	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Helper Functions ---

// setupTestGraphManager creates a predefined graph for testing purposes.
func setupTestGraphManager(t *testing.T) *graph.KnowledgeGraphManager {
	t.Helper() // Mark this as a test helper
	// Using NewKnowledgeGraphManager with an empty path forces in-memory only
	// (it won't load/save from/to disk)
	mgr := graph.NewKnowledgeGraphManager("") // Empty path = no file persistence for test

	// Graph structure with more details:
	// person:Alice (age:30, verified:true) --knows (weight:0.8)--> person:Bob (age:35, dept:Sales)
	// person:Alice (age:30, verified:true) --knows (weight:0.5)--> person:Charlie (age:40, dept:Eng, verified:true)
	// person:Bob (age:35, dept:Sales) --works_at--> org:AcmeCorp (industry:Tech)
	// person:Charlie (age:40, dept:Eng, verified:true) --reports_to--> person:Bob (age:35, dept:Sales)
	// org:AcmeCorp (industry:Tech) --located_in--> location:CityA (country:USA)
	entities := []types.Entity{
		{ID: "person:Alice", Type: "Person", Name: "Alice", Metadata: map[string]interface{}{"age": 30, "verified": true}},
		{ID: "person:Bob", Type: "Person", Name: "Bob", Metadata: map[string]interface{}{"age": 35, "department": "Sales"}},
		{ID: "person:Charlie", Type: "Person", Name: "Charlie", Metadata: map[string]interface{}{"age": 40, "department": "Engineering", "verified": true}},
		{ID: "org:AcmeCorp", Type: "Organization", Name: "AcmeCorp", Metadata: map[string]interface{}{"industry": "Tech"}},
		{ID: "location:CityA", Type: "Location", Name: "City A", Metadata: map[string]interface{}{"country": "USA"}},
		// Add a disconnected node for some tests
		{ID: "person:Isolated", Type: "Person", Name: "Isolated Person", Metadata: map[string]interface{}{"age": 99}},
	}
	relations := []types.Relation{
		{ID: "r_alice_bob", Source: "person:Alice", Target: "person:Bob", Type: "knows", Metadata: map[string]interface{}{"weight": 0.8, "since": "2022-01-15"}},
		{ID: "r_bob_acme", Source: "person:Bob", Target: "org:AcmeCorp", Type: "works_at"},
		{ID: "r_alice_charlie", Source: "person:Alice", Target: "person:Charlie", Type: "knows", Metadata: map[string]interface{}{"weight": 0.5}},
		{ID: "r_charlie_bob", Source: "person:Charlie", Target: "person:Bob", Type: "reports_to"},
		{ID: "r_acme_citya", Source: "org:AcmeCorp", Target: "location:CityA", Type: "located_in"},
	}

	// Use require for critical setup steps
	_, err := mgr.CreateEntities(entities)
	require.NoError(t, err, "Setup: Failed to create entities")
	_, err = mgr.CreateRelations(relations)
	require.NoError(t, err, "Setup: Failed to create relations")

	return mgr
}

// extractPathNodeIDs extracts node IDs from a path result for easier comparison.
func extractPathNodeIDs(t *testing.T, path graph.Path) []string {
	t.Helper()
	ids := []string{}
	for i, item := range path {
		if i%2 == 0 { // Even indices are Entities
			if entity, ok := item.(types.Entity); ok {
				ids = append(ids, entity.ID)
			} else {
				t.Fatalf("Expected Entity at even index %d in path, got %T", i, item)
			}
		}
	}
	return ids
}

// pathsContainSequence checks if specific path node sequences exist in the results.
// It ensures the number of paths matches and each expected sequence is found exactly once.
func pathsContainSequence(t *testing.T, result *graph.PathsResult, expectedSequences [][]string) bool {
	t.Helper()
	if len(expectedSequences) != len(result.Paths) {
		// Log details for easier debugging
		// t.Errorf("Path count mismatch: expected %d, got %d", len(expectedSequences), len(result.Paths))
		return false // Mismatched number of paths
	}
	foundMatch := make([]bool, len(expectedSequences))
	for _, path := range result.Paths {
		actualIDs := extractPathNodeIDs(t, path) // Pass t to helper
		matchedCurrentPath := false
		for i, expectedSeq := range expectedSequences {
			if !foundMatch[i] && assert.ObjectsAreEqual(expectedSeq, actualIDs) {
				foundMatch[i] = true
				matchedCurrentPath = true
				break
			}
		}
		if !matchedCurrentPath {
			// A path was found that doesn't match any *remaining* expected sequence
			// t.Errorf("Unexpected path sequence found: %v", actualIDs)
			return false
		}
	}
	// Check if all expected sequences were found (all elements in foundMatch should be true)
	// Use blank identifier for unused index variable
	for _, found := range foundMatch {
		if !found {
			// t.Errorf("Expected path sequence not found: %v", expectedSequences[i]) // Can't use i here
			return false
		}
	}
	return true
}

// --- Actual Tests ---

func TestBFS(t *testing.T) {
	mgr := setupTestGraphManager(t)
	params := graph.TraverseParams{
		StartNodeIDs: []string{"person:Alice"},
		Algorithm:    graph.BFSAlgorithm,
		MaxDepth:     -1, // Unlimited depth
	}

	result, err := mgr.Traverse(params)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Expected BFS order starting from person:Alice (undirected default neighbor func)
	// Reaches all 5 connected nodes
	assert.Len(t, result.VisitedDepths, 5, "Should visit 5 nodes (A, B, C, Acme, CityA)") // Check VisitedDepths map size

	// Check depths (undirected)
	expectedDepths := map[string]int{
		"person:Alice":   0,
		"person:Bob":     1,
		"person:Charlie": 1,
		"org:AcmeCorp":   2,
		"location:CityA": 3, // A->B->Acme->CityA
	}
	assert.Equal(t, expectedDepths["person:Alice"], result.VisitedDepths["person:Alice"])
	assert.Equal(t, expectedDepths["person:Bob"], result.VisitedDepths["person:Bob"])
	assert.Equal(t, expectedDepths["person:Charlie"], result.VisitedDepths["person:Charlie"])
	assert.Equal(t, expectedDepths["org:AcmeCorp"], result.VisitedDepths["org:AcmeCorp"])
	assert.Equal(t, expectedDepths["location:CityA"], result.VisitedDepths["location:CityA"])

	// Test MaxDepth = 1 (undirected)
	params.MaxDepth = 1
	resultDepth1, err := mgr.Traverse(params)
	require.NoError(t, err)
	assert.Len(t, resultDepth1.VisitedDepths, 3, "Should visit person:Alice, person:Bob, person:Charlie at depth <= 1") // VisitedDepths is the map now
	assert.Contains(t, resultDepth1.VisitedDepths, "person:Alice")
	assert.Contains(t, resultDepth1.VisitedDepths, "person:Bob")
	assert.Contains(t, resultDepth1.VisitedDepths, "person:Charlie")
	assert.NotContains(t, resultDepth1.VisitedDepths, "org:AcmeCorp")

}

func TestDFS(t *testing.T) {
	mgr := setupTestGraphManager(t)
	params := graph.TraverseParams{
		StartNodeIDs: []string{"person:Alice"},
		Algorithm:    graph.DFSAlgorithm,
		MaxDepth:     -1, // Unlimited depth
	}

	result, err := mgr.Traverse(params)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check elements match undirected reachability - should reach all 5 connected nodes
	assert.Len(t, result.VisitedDepths, 5, "Should visit 5 nodes (A, B, C, Acme, CityA)") // Check VisitedDepths map size

	// Check depths (will differ from BFS)
	fmt.Println("DFS Depths:", result.VisitedDepths) // Print for observation

	// Test MaxDepth = 1 (undirected)
	params.MaxDepth = 1
	resultDepth1, err := mgr.Traverse(params)
	require.NoError(t, err)
	// DFS with depth 1 from person:Alice visits A, and then either B or C (but not their children)
	assert.Len(t, resultDepth1.VisitedDepths, 3, "Should visit person:Alice, person:Bob, person:Charlie at depth <= 1") // VisitedDepths is the map
	assert.Contains(t, resultDepth1.VisitedDepths, "person:Alice")
	assert.Contains(t, resultDepth1.VisitedDepths, "person:Bob")
	assert.Contains(t, resultDepth1.VisitedDepths, "person:Charlie")
	assert.NotContains(t, resultDepth1.VisitedDepths, "org:AcmeCorp")

}

func TestExtractSubgraph(t *testing.T) {
	mgr := setupTestGraphManager(t)

	// Test Radius 0 from person:Alice
	params0 := graph.GetSubgraphParams{StartNodeIDs: []string{"person:Alice"}, Radius: 0}
	subgraph0, err0 := mgr.GetSubgraph(params0)
	require.NoError(t, err0)
	require.NotNil(t, subgraph0)
	assert.Len(t, subgraph0.Entities, 1, "Radius 0 should contain only start node person:Alice")
	assert.Contains(t, subgraph0.Entities, "person:Alice")
	assert.Empty(t, subgraph0.Relations, "Radius 0 should have no relations if start node has no self-loops or edges to itself")

	// Test Radius 1 from person:Alice (undirected reach + relation filter)
	params1 := graph.GetSubgraphParams{StartNodeIDs: []string{"person:Alice"}, Radius: 1}
	subgraph1, err1 := mgr.GetSubgraph(params1)
	require.NoError(t, err1)
	require.NotNil(t, subgraph1)
	// Undirected BFS reaches A, B, C at radius 1
	assert.Len(t, subgraph1.Entities, 3, "Radius 1 (undirected) should contain A, B, C")
	assert.Contains(t, subgraph1.Entities, "person:Alice")
	assert.Contains(t, subgraph1.Entities, "person:Bob")
	assert.Contains(t, subgraph1.Entities, "person:Charlie")
	// Relations within {A, B, C} are included: A->B, A->C, C->B
	assert.Len(t, subgraph1.Relations, 3, "Radius 1 (undirected) should include relations A->B, A->C, C->B")
	assert.Contains(t, subgraph1.Relations, "r_alice_bob")
	assert.Contains(t, subgraph1.Relations, "r_alice_charlie")
	assert.Contains(t, subgraph1.Relations, "r_charlie_bob") // Included as both C and B are in the node set

	// Test Radius 2 from person:Alice (undirected reach + relation filter)
	params2 := graph.GetSubgraphParams{StartNodeIDs: []string{"person:Alice"}, Radius: 2}
	subgraph2, err2 := mgr.GetSubgraph(params2)
	require.NoError(t, err2)
	require.NotNil(t, subgraph2)
	// Undirected BFS reaches A, B, C (d=1), Acme (d=2). CityA (d=3) is excluded.
	assert.Len(t, subgraph2.Entities, 4, "Radius 2 (undirected) should contain A, B, C, Acme")
	assert.Contains(t, subgraph2.Entities, "person:Alice")
	assert.Contains(t, subgraph2.Entities, "person:Bob")
	assert.Contains(t, subgraph2.Entities, "person:Charlie")
	assert.Contains(t, subgraph2.Entities, "org:AcmeCorp")
	assert.NotContains(t, subgraph2.Entities, "location:CityA")
	// Relations within {A, B, C, Acme}: A->B, A->C, B->Acme, C->B
	assert.Len(t, subgraph2.Relations, 4, "Radius 2 (undirected) should include A->B, A->C, B->Acme, C->B")
	assert.Contains(t, subgraph2.Relations, "r_alice_bob")
	assert.Contains(t, subgraph2.Relations, "r_alice_charlie")
	assert.Contains(t, subgraph2.Relations, "r_bob_acme")
	assert.Contains(t, subgraph2.Relations, "r_charlie_bob")
	assert.NotContains(t, subgraph2.Relations, "r_acme_citya")

	// Test invalid start node - expect error from BFS
	paramsInvalid := graph.GetSubgraphParams{StartNodeIDs: []string{"nonexistent"}, Radius: 1}
	subgraphInvalid, errInvalid := mgr.GetSubgraph(paramsInvalid)
	assert.Error(t, errInvalid, "Getting subgraph from non-existent node should error")
	assert.Nil(t, subgraphInvalid, "Subgraph result should be nil on error") // Check nil on error

}

func TestFindPaths(t *testing.T) {
	mgr := setupTestGraphManager(t)

	// Test path person:Alice -> org:AcmeCorp (Expect 2 paths: A->B->Acme, A->C->B->Acme)
	paramsAB := graph.FindPathsParams{StartNodeID: "person:Alice", EndNodeID: "org:AcmeCorp", MaxLength: -1}
	resultAB, errAB := mgr.FindPaths(paramsAB)
	require.NoError(t, errAB)
	require.NotNil(t, resultAB)
	assert.Len(t, resultAB.Paths, 2, "Should find 2 DIRECTED paths from A to AcmeCorp")
	expectedPathsAB := [][]string{
		{"person:Alice", "person:Bob", "org:AcmeCorp"},
		{"person:Alice", "person:Charlie", "person:Bob", "org:AcmeCorp"},
	}
	assert.True(t, pathsContainSequence(t, resultAB, expectedPathsAB), "Path check failed for A->AcmeCorp")

	// Test path person:Alice -> location:CityA (Expect 2 paths: A->B->Acme->CityA, A->C->B->Acme->CityA)
	paramsAL := graph.FindPathsParams{StartNodeID: "person:Alice", EndNodeID: "location:CityA", MaxLength: -1}
	resultAL, errAL := mgr.FindPaths(paramsAL)
	require.NoError(t, errAL)
	require.NotNil(t, resultAL)
	assert.Len(t, resultAL.Paths, 2, "Should find 2 DIRECTED paths from A to CityA")
	expectedPathsAL := [][]string{
		{"person:Alice", "person:Bob", "org:AcmeCorp", "location:CityA"},
		{"person:Alice", "person:Charlie", "person:Bob", "org:AcmeCorp", "location:CityA"},
	}
	assert.True(t, pathsContainSequence(t, resultAL, expectedPathsAL), "Path check failed for A->CityA")

	// Test path person:Alice -> person:Charlie with MaxLength = 1 (expected: person:Alice-person:Charlie)
	paramsAC_L1 := graph.FindPathsParams{StartNodeID: "person:Alice", EndNodeID: "person:Charlie", MaxLength: 1}
	resultAC_L1, errAC_L1 := mgr.FindPaths(paramsAC_L1)
	require.NoError(t, errAC_L1)
	require.NotNil(t, resultAC_L1)
	assert.Len(t, resultAC_L1.Paths, 1, "Should find 1 path from person:Alice to person:Charlie with MaxLength 1")
	assert.True(t, pathsContainSequence(t, resultAC_L1, [][]string{{"person:Alice", "person:Charlie"}}), "Path person:Alice-person:Charlie should exist")

	// Test path person:Alice -> person:Charlie with MaxLength = 2 (expected: person:Alice-person:Charlie)
	paramsAC_L2 := graph.FindPathsParams{StartNodeID: "person:Alice", EndNodeID: "person:Charlie", MaxLength: 2}
	resultAC_L2, errAC_L2 := mgr.FindPaths(paramsAC_L2)
	require.NoError(t, errAC_L2)
	require.NotNil(t, resultAC_L2)
	assert.Len(t, resultAC_L2.Paths, 1, "Should find 1 path from person:Alice to person:Charlie with MaxLength 2")
	assert.True(t, pathsContainSequence(t, resultAC_L2, [][]string{{"person:Alice", "person:Charlie"}}), "Path person:Alice-person:Charlie should exist")

	// Test path person:Alice -> person:Charlie with MaxLength = 3 (expected: person:Alice-person:Charlie)
	paramsAC_L3 := graph.FindPathsParams{StartNodeID: "person:Alice", EndNodeID: "person:Charlie", MaxLength: 3}
	resultAC_L3, errAC_L3 := mgr.FindPaths(paramsAC_L3)
	require.NoError(t, errAC_L3)
	require.NotNil(t, resultAC_L3)
	assert.Len(t, resultAC_L3.Paths, 1, "Should find 1 path from person:Alice to person:Charlie with MaxLength 3")
	assert.True(t, pathsContainSequence(t, resultAC_L3, [][]string{{"person:Alice", "person:Charlie"}}), "Path person:Alice-person:Charlie should exist")

	// Test path where no directed path exists (org:AcmeCorp -> person:Alice)
	paramsCA := graph.FindPathsParams{StartNodeID: "org:AcmeCorp", EndNodeID: "person:Alice", MaxLength: -1}
	resultCA, errCA := mgr.FindPaths(paramsCA)
	require.NoError(t, errCA)
	require.NotNil(t, resultCA)
	assert.Empty(t, resultCA.Paths, "Should find 0 directed paths from Acme to Alice")

	// Test path Start node does not exist - expect error
	paramsZ := graph.FindPathsParams{StartNodeID: "nonexistent", EndNodeID: "person:Alice", MaxLength: -1}
	resultZ, errZ := mgr.FindPaths(paramsZ)
	assert.Error(t, errZ, "FindPaths from non-existent node should error")
	assert.Nil(t, resultZ, "PathsResult should be nil on error") // Check nil on error

	// Test path End node does not exist (should find 0 paths, no error) - already correct
	paramsAZ := graph.FindPathsParams{StartNodeID: "person:Alice", EndNodeID: "nonexistent", MaxLength: -1}
	resultAZ, errAZ := mgr.FindPaths(paramsAZ)
	require.NoError(t, errAZ)
	require.NotNil(t, resultAZ)
	assert.Empty(t, resultAZ.Paths, "Should find 0 paths if end node doesn't exist")
}

func TestGetSubgraph_WithFilters(t *testing.T) {
	mgr := setupTestGraphManager(t)

	tests := []struct {
		name              string
		params            graph.GetSubgraphParams
		expectedEntityIDs []string
		expectedRelIDs    []string
		expectError       bool
	}{
		{
			name: "No filters (matches undirected radius 2)",
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"},
				Radius:       2,
			},
			// Undirected reach: {A, B, C, Acme}, Relations within: {A->B, A->C, B->Acme, C->B}
			expectedEntityIDs: []string{"person:Alice", "person:Bob", "person:Charlie", "org:AcmeCorp"},
			expectedRelIDs:    []string{"r_alice_bob", "r_alice_charlie", "r_bob_acme", "r_charlie_bob"},
		},
		{
			name: "Node filter (Type: Person)",
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"},
				Radius:       2,
				Filters: &graph.SubgraphFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Person"}},
					},
				},
			},
			// Undirected BFS visits A, B, C (all Persons). Acme is filtered out by visitFn.
			// Relations within {A, B, C} remain: A->B, A->C, C->B.
			expectedEntityIDs: []string{"person:Alice", "person:Bob", "person:Charlie"},
			expectedRelIDs:    []string{"r_alice_bob", "r_alice_charlie", "r_charlie_bob"},
		},
		{
			name: "Node filter (metadata.verified: true)",
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"},
				Radius:       2,
				Filters: &graph.SubgraphFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Metadata.verified", Value: true}},
					},
				},
			},
			// Should include only Alice and Charlie (verified=true).
			expectedEntityIDs: []string{"person:Alice", "person:Charlie"},
			// Only relation between Alice and Charlie remains.
			expectedRelIDs: []string{"r_alice_charlie"},
		},
		{
			name: "Relation filter (Type: knows)",
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"},
				Radius:       2,
				Filters: &graph.SubgraphFiltersInternal{
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "knows"}},
					},
				},
			},
			// Undirected BFS includes nodes {A, B, C, Acme}.
			// Final relation filter keeps only 'knows' relations within this set: A->B, A->C.
			expectedEntityIDs: []string{"person:Alice", "person:Bob", "person:Charlie", "org:AcmeCorp"},
			expectedRelIDs:    []string{"r_alice_bob", "r_alice_charlie"},
		},
		{
			name: "Relation filter (metadata.weight=0.8)", // Corrected name
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"},
				Radius:       2,
				Filters: &graph.SubgraphFiltersInternal{
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Metadata.weight", Value: 0.8}},
					},
				},
			},
			// Undirected BFS includes nodes {A, B, C, Acme}.
			// Final relation filter keeps only A->B (weight=0.8).
			expectedEntityIDs: []string{"person:Alice", "person:Bob", "person:Charlie", "org:AcmeCorp"},
			expectedRelIDs:    []string{"r_alice_bob"},
		},
		{
			name: "Combined filter (Node: Person, Relation: knows)",
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"},
				Radius:       2,
				Filters: &graph.SubgraphFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Person"}},
					},
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "knows"}},
					},
				},
			},
			// Nodes: Alice, Bob, Charlie (filtered by node type)
			// Relations: alice->bob, alice->charlie (filtered by relation type)
			// Relation charlie->bob is removed because its type is 'reports_to'
			expectedEntityIDs: []string{"person:Alice", "person:Bob", "person:Charlie"},
			expectedRelIDs:    []string{"r_alice_bob", "r_alice_charlie"},
		},
		{
			name: "Filter resulting in empty set (was only start node)", // Renamed
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"},
				Radius:       2,
				Filters: &graph.SubgraphFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						// Filter that Alice herself doesn't match
						Conditions: []graph.FilterCondition{{Property: "Metadata.nonexistent", Value: "value"}},
					},
				},
			},
			// Alice is filtered out by the bfsVisitFn, so BFS doesn't start.
			expectedEntityIDs: []string{},
			expectedRelIDs:    []string{},
		},
		{
			name: "Start node does not match filter",
			params: graph.GetSubgraphParams{
				StartNodeIDs: []string{"person:Alice"}, // Alice age=30
				Radius:       1,
				Filters: &graph.SubgraphFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Metadata.age", Value: 99}},
					},
				},
			},
			// Since Alice doesn't match, the traversal doesn't even start effectively for filter application.
			expectedEntityIDs: []string{}, // Expect empty subgraph
			expectedRelIDs:    []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			subgraph, err := mgr.GetSubgraph(tc.params)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, subgraph)
			} else {
				require.NoError(t, err)
				require.NotNil(t, subgraph)

				// Extract entity and relation IDs from the result for comparison
				actualEntityIDs := make([]string, 0, len(subgraph.Entities))
				for id := range subgraph.Entities {
					actualEntityIDs = append(actualEntityIDs, id)
				}
				actualRelIDs := make([]string, 0, len(subgraph.Relations))
				for id := range subgraph.Relations {
					actualRelIDs = append(actualRelIDs, id)
				}

				// Use assert.ElementsMatch for unordered comparison
				assert.ElementsMatch(t, tc.expectedEntityIDs, actualEntityIDs, "Mismatch in expected entities")
				assert.ElementsMatch(t, tc.expectedRelIDs, actualRelIDs, "Mismatch in expected relations")
			}
		})
	}
}

func TestTraverse_WithFilters(t *testing.T) {
	mgr := setupTestGraphManager(t)

	tests := []struct {
		name               string
		params             graph.TraverseParams
		expectedVisitedIDs []string // Order might matter depending on algorithm & filters
		expectError        bool
	}{
		{
			name: "BFS No filters (undirected)",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Alice"},
				Algorithm:    graph.BFSAlgorithm,
				MaxDepth:     -1,
			},
			// Undirected reach: {A, B, C, Acme, CityA}
			expectedVisitedIDs: []string{"person:Alice", "person:Bob", "person:Charlie", "org:AcmeCorp", "location:CityA"},
		},
		{
			name: "BFS Node filter (Type: Person)",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Alice"},
				Algorithm:    graph.BFSAlgorithm,
				MaxDepth:     -1,
				Filters: &graph.TraversalFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Person"}},
					},
				},
			},
			// Visits Alice, Bob, Charlie. Traversal stops at Bob/Charlie as org:AcmeCorp isn't Person.
			expectedVisitedIDs: []string{"person:Alice", "person:Bob", "person:Charlie"},
		},
		{
			name: "BFS Relation filter (Type: knows)",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Alice"},
				Algorithm:    graph.BFSAlgorithm,
				MaxDepth:     -1,
				Filters: &graph.TraversalFiltersInternal{
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "knows"}},
					},
				},
			},
			// Can only traverse knows links: Alice -> Bob, Alice -> Charlie.
			expectedVisitedIDs: []string{"person:Alice", "person:Bob", "person:Charlie"},
		},
		{
			name: "BFS Combined filter (Node: Person, Relation: knows)",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Alice"},
				Algorithm:    graph.BFSAlgorithm,
				MaxDepth:     -1,
				Filters: &graph.TraversalFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Person"}},
					},
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "knows"}},
					},
				},
			},
			// Same as above in this case, as knows relations only go to Persons.
			expectedVisitedIDs: []string{"person:Alice", "person:Bob", "person:Charlie"},
		},
		{
			name: "BFS Combined filter (Node: verified=true, Relation: any)",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Alice"},
				Algorithm:    graph.BFSAlgorithm,
				MaxDepth:     -1,
				Filters: &graph.TraversalFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Metadata.verified", Value: true}},
					},
				},
			},
			// Visits Alice (verified), then Charlie (verified). Stops at Charlie as Bob isn't verified.
			expectedVisitedIDs: []string{"person:Alice", "person:Charlie"},
		},
		{
			name: "BFS Start node does not match filter",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Bob"}, // Bob is not verified
				Algorithm:    graph.BFSAlgorithm,
				MaxDepth:     -1,
				Filters: &graph.TraversalFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Metadata.verified", Value: true}},
					},
				},
			},
			// Traversal shouldn't start effectively.
			expectedVisitedIDs: []string{}, // Expect no nodes visited
		},
		{
			name: "DFS No filters (undirected)",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Alice"},
				Algorithm:    graph.DFSAlgorithm,
				MaxDepth:     -1,
			},
			// Undirected reach: {A, B, C, Acme, CityA}
			expectedVisitedIDs: []string{"person:Alice", "person:Bob", "person:Charlie", "org:AcmeCorp", "location:CityA"},
		},
		{
			name: "DFS Node filter (Type: Person)",
			params: graph.TraverseParams{
				StartNodeIDs: []string{"person:Alice"},
				Algorithm:    graph.DFSAlgorithm,
				MaxDepth:     -1,
				Filters: &graph.TraversalFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Person"}},
					},
				},
			},
			// Visits Alice, then maybe Bob, then maybe Charlie. Stops branches leading to non-persons.
			expectedVisitedIDs: []string{"person:Alice", "person:Bob", "person:Charlie"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Traverse returns *graph.TraverseResult which contains the map
			result, err := mgr.Traverse(tc.params)

			if tc.expectError {
				assert.Error(t, err)
				// Check if result itself is nil or if the map inside is nil/empty
				// assert.Nil(t, result) // Depending on error handling strategy
			} else {
				require.NoError(t, err)
				require.NotNil(t, result, "TraverseResult should not be nil on success")
				require.NotNil(t, result.VisitedDepths, "VisitedDepths map should not be nil on success")

				// Extract just the visited IDs (keys of the map inside the result struct)
				actualVisitedIDs := make([]string, 0, len(result.VisitedDepths))
				for entityID := range result.VisitedDepths { // Range over the map inside the struct
					actualVisitedIDs = append(actualVisitedIDs, entityID)
				}

				// Use assert.ElementsMatch for unordered comparison of visited nodes
				assert.ElementsMatch(t, tc.expectedVisitedIDs, actualVisitedIDs, "Mismatch in expected visited entities")

				// Optional: Add checks for depths using result.VisitedDepths[someID]
				// e.g., assert.Equal(t, expectedDepth, result.VisitedDepths[someID])
			}
		})
	}
}

func TestFindPaths_WithFilters(t *testing.T) {
	mgr := setupTestGraphManager(t)

	tests := []struct {
		name                 string
		params               graph.FindPathsParams
		expectedPathNodeSeqs [][]string // Expected sequences of node IDs for each found path
		expectError          bool
	}{
		{
			name: "No filters A->CityA (Directed)",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice",
				EndNodeID:   "location:CityA",
				MaxLength:   -1,
			},
			// Expect TWO directed paths
			expectedPathNodeSeqs: [][]string{
				{"person:Alice", "person:Bob", "org:AcmeCorp", "location:CityA"},
				{"person:Alice", "person:Charlie", "person:Bob", "org:AcmeCorp", "location:CityA"},
			},
		},
		{
			name: "Node filter (Type: Person) A->CityA",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice",
				EndNodeID:   "location:CityA",
				MaxLength:   -1,
				Filters: &graph.PathFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Person"}},
					},
				},
			},
			// Paths blocked because org:AcmeCorp and location:CityA are not Persons.
			expectedPathNodeSeqs: [][]string{},
		},
		{
			name: "Node filter (metadata.verified: true) A->CityA",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice",
				EndNodeID:   "location:CityA",
				MaxLength:   -1,
				Filters: &graph.PathFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Metadata.verified", Value: true}},
					},
				},
			},
			// Blocked because Bob (unverified) is on all paths to CityA.
			expectedPathNodeSeqs: [][]string{},
		},
		{
			name: "Relation filter (Type: knows) A->CityA",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice",
				EndNodeID:   "location:CityA",
				MaxLength:   -1,
				Filters: &graph.PathFiltersInternal{
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "knows"}},
					},
				},
			},
			// Blocked because path requires 'works_at', 'reports_to', 'located_in' which are not 'knows'.
			expectedPathNodeSeqs: [][]string{},
		},
		{
			name: "Relation filter (metadata.weight=0.8) A->Bob",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice",
				EndNodeID:   "person:Bob",
				MaxLength:   -1,
				Filters: &graph.PathFiltersInternal{
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Metadata.weight", Value: 0.8}},
					},
				},
			},
			// Only the direct path A->Bob matches the weight filter.
			expectedPathNodeSeqs: [][]string{
				{"person:Alice", "person:Bob"},
			},
		},
		{
			name: "Combined filter (Node: Person, Relation: knows) A->Bob",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice",
				EndNodeID:   "person:Bob",
				MaxLength:   -1,
				Filters: &graph.PathFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Person"}},
					},
					RelationFilter: &graph.RelationFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "knows"}},
					},
				},
			},
			// Path A->C->B is blocked because relation C->B is 'reports_to', not 'knows'.
			// Path A->B matches both node and relation filters.
			expectedPathNodeSeqs: [][]string{
				{"person:Alice", "person:Bob"},
			},
		},
		{
			name: "Filter blocks start node",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice", // Alice type = Person
				EndNodeID:   "person:Bob",
				MaxLength:   -1,
				Filters: &graph.PathFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Organization"}},
					},
				},
			},
			// Start node doesn't match, no paths possible.
			expectedPathNodeSeqs: [][]string{},
		},
		{
			name: "Filter blocks end node",
			params: graph.FindPathsParams{
				StartNodeID: "person:Alice",
				EndNodeID:   "person:Bob", // Bob type = Person
				MaxLength:   -1,
				Filters: &graph.PathFiltersInternal{
					NodeFilter: &graph.NodeFilter{
						Conditions: []graph.FilterCondition{{Property: "Type", Value: "Location"}},
					},
				},
			},
			// End node doesn't match, no paths possible.
			expectedPathNodeSeqs: [][]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mgr.FindPaths(tc.params)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result, "PathsResult should not be nil on success")

				// Use the top-level helper function
				assert.True(t, pathsContainSequence(t, result, tc.expectedPathNodeSeqs),
					"Mismatch between found paths and expected path node sequences. Expected: %v, Found paths: %v",
					tc.expectedPathNodeSeqs, result.Paths) // Log paths for debugging

				// Also explicitly check the count
				assert.Len(t, result.Paths, len(tc.expectedPathNodeSeqs), "Mismatch in number of paths found")
			}
		})
	}
}
