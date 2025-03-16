package main

import (
	"encoding/json"
	"testing"
)

// Helper function to get numeric value from interface{}
func getNumericValue(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func TestNew(t *testing.T) {
	tool := NewTool()
	if tool == nil {
		t.Fatal("NewTool() returned nil")
	}
	if tool.Name() != "sequentialthinking" {
		t.Errorf("Expected tool name 'sequentialthinking', got '%s'", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name    string
		input   ThoughtData
		wantErr bool
		check   func(t *testing.T, result interface{})
	}{
		{
			name: "basic thought",
			input: ThoughtData{
				Thought:           "Test thought",
				ThoughtNumber:     1,
				TotalThoughts:     1,
				NextThoughtNeeded: false,
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				res := result.(map[string]interface{})
				if res["isError"].(bool) {
					t.Error("Expected success, got error")
				}
				metadata := res["metadata"].(map[string]interface{})
				if metadata["thoughtNumber"].(int) != 1 {
					t.Errorf("Expected thought number 1, got %v", metadata["thoughtNumber"])
				}
				if metadata["thoughtHistoryLength"].(int) != 1 {
					t.Errorf("Expected history length 1, got %v", metadata["thoughtHistoryLength"])
				}
			},
		},
		{
			name: "empty thought",
			input: ThoughtData{
				Thought:           "",
				ThoughtNumber:     1,
				TotalThoughts:     1,
				NextThoughtNeeded: false,
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				res := result.(map[string]interface{})
				if !res["isError"].(bool) {
					t.Error("Expected error for empty thought")
				}
				content := res["content"].([]map[string]interface{})
				if content[0]["text"].(string) != "Validation error: thought must not be empty" {
					t.Errorf("Unexpected error message: %v", content[0]["text"])
				}
			},
		},
		{
			name: "invalid thought number",
			input: ThoughtData{
				Thought:           "Test thought",
				ThoughtNumber:     0,
				TotalThoughts:     1,
				NextThoughtNeeded: false,
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				res := result.(map[string]interface{})
				if !res["isError"].(bool) {
					t.Error("Expected error for invalid thought number")
				}
			},
		},
		{
			name: "invalid total thoughts",
			input: ThoughtData{
				Thought:           "Test thought",
				ThoughtNumber:     1,
				TotalThoughts:     0,
				NextThoughtNeeded: false,
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				res := result.(map[string]interface{})
				if !res["isError"].(bool) {
					t.Error("Expected error for invalid total thoughts")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewTool()
			params, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			result, err := tool.Execute(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestThoughtSequences(t *testing.T) {
	tests := []struct {
		name     string
		thoughts []ThoughtData
		wantErr  bool
		check    func(t *testing.T, result interface{})
	}{
		{
			name: "thought revision",
			thoughts: []ThoughtData{
				{
					Thought:           "Initial thought",
					ThoughtNumber:     1,
					TotalThoughts:     2,
					NextThoughtNeeded: true,
				},
				{
					Thought:           "Revised thought",
					ThoughtNumber:     2,
					TotalThoughts:     2,
					NextThoughtNeeded: false,
					IsRevision:        true,
					RevisesThought:    1,
				},
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				res := result.(map[string]interface{})
				if res["isError"].(bool) {
					t.Error("Expected success for revision")
				}
				metadata := res["metadata"].(map[string]interface{})
				if metadata["thoughtHistoryLength"].(int) != 2 {
					t.Errorf("Expected history length 2, got %v", metadata["thoughtHistoryLength"])
				}
			},
		},
		{
			name: "thought branching",
			thoughts: []ThoughtData{
				{
					Thought:           "Main thought",
					ThoughtNumber:     1,
					TotalThoughts:     2,
					NextThoughtNeeded: true,
				},
				{
					Thought:           "Branch thought",
					ThoughtNumber:     2,
					TotalThoughts:     2,
					NextThoughtNeeded: false,
					BranchFromThought: 1,
					BranchID:          "test-branch",
				},
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				res := result.(map[string]interface{})
				if res["isError"].(bool) {
					t.Error("Expected success for branching")
				}
				metadata := res["metadata"].(map[string]interface{})
				branches := metadata["branches"].([]string)
				if len(branches) != 1 || branches[0] != "test-branch" {
					t.Errorf("Expected branch 'test-branch', got %v", branches)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewTool()
			var result interface{}
			var err error

			for i, thought := range tt.thoughts {
				params, err := json.Marshal(thought)
				if err != nil {
					t.Fatalf("Failed to marshal thought %d: %v", i, err)
				}
				result, err = tool.Execute(params)
				if err != nil {
					break
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestSchema(t *testing.T) {
	tool := NewTool()
	schema := tool.Schema()

	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Verify required fields
	properties, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema missing properties")
	}

	requiredFields := []string{"thought", "thoughtNumber", "totalThoughts", "nextThoughtNeeded"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Schema missing required field: %s", field)
		}
	}
}
