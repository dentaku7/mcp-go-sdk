package tool

import (
	"encoding/json"
	"testing"

	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

func TestDeleteRelationsTool_Execute(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantCount     int
		wantError     bool
		checkRelation string // relation ID to check if it still exists
		wantExists    bool   // whether the relation should exist after deletion
	}{
		{
			name: "successful deletion by from/to/type",
			input: `{
				"relations": [{
					"from": "e1",
					"to": "e2",
					"relationType": "test-relation"
				}]
			}`,
			wantCount:     1,
			wantError:     false,
			checkRelation: "r1",
			wantExists:    false,
		},
		{
			name: "relation doesn't exist",
			input: `{
				"relations": [{
					"from": "e1",
					"to": "e3",
					"relationType": "nonexistent"
				}]
			}`,
			wantCount:     0,
			wantError:     false,
			checkRelation: "r2",
			wantExists:    true,
		},
		{
			name: "multiple relations",
			input: `{
				"relations": [
					{
						"from": "e1",
						"to": "e2",
						"relationType": "test-relation"
					},
					{
						"from": "e2",
						"to": "e3",
						"relationType": "another-relation"
					}
				]
			}`,
			wantCount:     2,
			wantError:     false,
			checkRelation: "r2",
			wantExists:    false,
		},
		{
			name: "invalid input",
			input: `{
				"relations": [
					{
						"invalid": "format"
					}
				]
			}`,
			wantCount: 0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new graph manager for each test
			tmpFile := t.TempDir() + "/test_memory.json"
			manager := graph.NewKnowledgeGraphManager(tmpFile)
			tool := NewDeleteRelationsTool(manager)

			// Create test entities
			entities := []types.Entity{
				{ID: "e1", Type: "test", Name: "Entity 1"},
				{ID: "e2", Type: "test", Name: "Entity 2"},
				{ID: "e3", Type: "test", Name: "Entity 3"},
			}
			if _, err := manager.CreateEntities(entities); err != nil {
				t.Fatalf("Failed to create test entities: %v", err)
			}

			// Create test relations
			relations := []types.Relation{
				{
					ID:     "r1",
					Type:   "test-relation",
					Source: "e1",
					Target: "e2",
				},
				{
					ID:     "r2",
					Type:   "another-relation",
					Source: "e2",
					Target: "e3",
				},
			}
			if _, err := manager.CreateRelations(relations); err != nil {
				t.Fatalf("Failed to create test relations: %v", err)
			}

			// Verify initial graph state
			initialGraph := manager.ReadGraph()
			if len(initialGraph.Relations) != 2 {
				t.Fatalf("Expected 2 initial relations, got %d", len(initialGraph.Relations))
			}

			// Execute the tool
			result, err := tool.Execute(json.RawMessage(tt.input))

			// Check error condition
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check response format
			response, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("Execute() result is not a map[string]interface{}")
			}

			// Check deleted count
			metadata, ok := response["metadata"].(map[string]interface{})
			if !ok {
				t.Fatalf("Execute() result metadata is not a map[string]interface{}")
			}

			deletedCount, ok := metadata["deleted_count"].(int)
			if !ok {
				t.Fatalf("Execute() result deleted_count is not an int")
			}

			if deletedCount != tt.wantCount {
				t.Errorf("Execute() deleted_count = %v, want %v", deletedCount, tt.wantCount)
			}

			// Verify graph state if a relation needs to be checked
			if tt.checkRelation != "" {
				graph := manager.ReadGraph()
				_, exists := graph.Relations[tt.checkRelation]
				if exists != tt.wantExists {
					t.Errorf("Relation %s exists = %v, want %v", tt.checkRelation, exists, tt.wantExists)
				}
			}
		})
	}
}
