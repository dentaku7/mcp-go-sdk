// Package main implements a sequential thinking tool for MCP
package main

import (
	"encoding/json"
	"fmt"
)

// ThoughtData represents a single thought in the sequential thinking process.
// It contains all the necessary information about a thought, including its position
// in the sequence, branching information, and revision details.
type ThoughtData struct {
	Thought           string `json:"thought"`                     // The actual thought content
	ThoughtNumber     int    `json:"thoughtNumber"`               // Position in the sequence
	TotalThoughts     int    `json:"totalThoughts"`               // Total expected thoughts
	NextThoughtNeeded bool   `json:"nextThoughtNeeded"`           // Whether another thought is needed
	IsRevision        bool   `json:"isRevision,omitempty"`        // If this revises a previous thought
	RevisesThought    int    `json:"revisesThought,omitempty"`    // Which thought is being revised
	BranchFromThought int    `json:"branchFromThought,omitempty"` // Parent thought for branching
	BranchID          string `json:"branchId,omitempty"`          // Branch identifier
	NeedsMoreThoughts bool   `json:"needsMoreThoughts,omitempty"` // If more thoughts are needed
}

// Tool represents the sequential thinking tool implementation.
// It provides methods for processing thoughts in a sequence, with support
// for revisions and branching paths.
type Tool struct {
	description    string
	thoughtHistory []ThoughtData
	branches       map[string][]ThoughtData
}

// NewTool creates a new sequential thinking tool with initialized state.
func NewTool() *Tool {
	return &Tool{
		description: `A detailed tool for dynamic and reflective problem-solving through thoughts.
This tool helps analyze problems through a flexible thinking process that can adapt and evolve.
Each thought can build on, question, or revise previous insights as understanding deepens.

When to use this tool:
- Breaking down complex problems into steps
- Planning and design with room for revision
- Analysis that might need course correction
- Problems where the full scope might not be clear initially
- Problems that require a multi-step solution
- Tasks that need to maintain context over multiple steps
- Situations where irrelevant information needs to be filtered out

Key features:
- You can adjust total_thoughts up or down as you progress
- You can question or revise previous thoughts
- You can add more thoughts even after reaching what seemed like the end
- You can express uncertainty and explore alternative approaches
- Not every thought needs to build linearly - you can branch or backtrack
- Generates a solution hypothesis
- Verifies the hypothesis based on the Chain of Thought steps
- Repeats the process until satisfied
- Provides a correct answer

Parameters explained:
- thought: Your current thinking step, which can include:
* Regular analytical steps
* Revisions of previous thoughts
* Questions about previous decisions
* Realizations about needing more analysis
* Changes in approach
* Hypothesis generation
* Hypothesis verification
- next_thought_needed: True if you need more thinking, even if at what seemed like the end
- thought_number: Current number in sequence (can go beyond initial total if needed)
- total_thoughts: Current estimate of thoughts needed (can be adjusted up/down)
- is_revision: A boolean indicating if this thought revises previous thinking
- revises_thought: If is_revision is true, which thought number is being reconsidered
- branch_from_thought: If branching, which thought number is the branching point
- branch_id: Identifier for the current branch (if any)
- needs_more_thoughts: If reaching end but realizing more thoughts needed

You should:
1. Start with an initial estimate of needed thoughts, but be ready to adjust
2. Feel free to question or revise previous thoughts
3. Don't hesitate to add more thoughts if needed, even at the "end"
4. Express uncertainty when present
5. Mark thoughts that revise previous thinking or branch into new paths
6. Ignore information that is irrelevant to the current step
7. Generate a solution hypothesis when appropriate
8. Verify the hypothesis based on the Chain of Thought steps
9. Repeat the process until satisfied with the solution
10. Provide a single, ideally correct answer as the final output
11. Only set next_thought_needed to false when truly done and a satisfactory answer is reached`,
		thoughtHistory: make([]ThoughtData, 0),
		branches:       make(map[string][]ThoughtData),
	}
}

// Name returns the identifier for this tool.
func (t *Tool) Name() string {
	return "sequentialthinking"
}

// Description returns the detailed description of the tool.
func (t *Tool) Description() string {
	return t.description
}

// formatThought formats a thought with visual elements and context.
func (t *Tool) formatThought(data ThoughtData) string {
	var prefix, context string

	if data.IsRevision {
		prefix = "🔄 Revision"
		context = fmt.Sprintf(" (revising thought %d)", data.RevisesThought)
	} else if data.BranchFromThought > 0 {
		prefix = "🌿 Branch"
		context = fmt.Sprintf(" (from thought %d, ID: %s)", data.BranchFromThought, data.BranchID)
	} else {
		prefix = "💭 Thought"
		context = ""
	}

	header := fmt.Sprintf("%s %d/%d%s", prefix, data.ThoughtNumber, data.TotalThoughts, context)
	return fmt.Sprintf("%s\n%s", header, data.Thought)
}

// Execute processes a thought and returns the formatted result.
func (t *Tool) Execute(params json.RawMessage) (interface{}, error) {
	var thoughtData ThoughtData
	if err := json.Unmarshal(params, &thoughtData); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %v", err)
	}

	if err := t.validateThought(&thoughtData); err != nil {
		return t.formatError(err.Error()), nil
	}

	// Adjust total thoughts if needed
	if thoughtData.ThoughtNumber > thoughtData.TotalThoughts {
		thoughtData.TotalThoughts = thoughtData.ThoughtNumber
	}

	// Store thought in history
	t.thoughtHistory = append(t.thoughtHistory, thoughtData)

	// Handle branching
	if thoughtData.BranchFromThought > 0 && thoughtData.BranchID != "" {
		if _, exists := t.branches[thoughtData.BranchID]; !exists {
			t.branches[thoughtData.BranchID] = make([]ThoughtData, 0)
		}
		t.branches[thoughtData.BranchID] = append(t.branches[thoughtData.BranchID], thoughtData)
	}

	// Format the thought with visual elements
	formattedThought := t.formatThought(thoughtData)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": formattedThought,
			},
		},
		"metadata": map[string]interface{}{
			"thoughtNumber":        thoughtData.ThoughtNumber,
			"totalThoughts":        thoughtData.TotalThoughts,
			"nextThoughtNeeded":    thoughtData.NextThoughtNeeded,
			"branches":             t.getBranchIDs(),
			"thoughtHistoryLength": len(t.thoughtHistory),
		},
		"isError": false,
	}, nil
}

// Schema returns the JSON schema for the tool's parameters.
func (t *Tool) Schema() json.RawMessage {
	schema := `{
		"type": "object",
		"properties": {
			"thought": {
				"type": "string",
				"description": "Your current thinking step"
			},
			"nextThoughtNeeded": {
				"type": "boolean",
				"description": "Whether another thought step is needed"
			},
			"thoughtNumber": {
				"type": "integer",
				"minimum": 1,
				"description": "Current thought number"
			},
			"totalThoughts": {
				"type": "integer",
				"minimum": 1,
				"description": "Estimated total thoughts needed"
			},
			"isRevision": {
				"type": "boolean",
				"description": "Whether this revises previous thinking"
			},
			"revisesThought": {
				"type": "integer",
				"minimum": 1,
				"description": "Which thought is being reconsidered"
			},
			"branchFromThought": {
				"type": "integer",
				"minimum": 1,
				"description": "Branching point thought number"
			},
			"branchId": {
				"type": "string",
				"description": "Branch identifier"
			},
			"needsMoreThoughts": {
				"type": "boolean",
				"description": "If more thoughts are needed"
			}
		},
		"required": ["thought", "nextThoughtNeeded", "thoughtNumber", "totalThoughts"]
	}`
	return json.RawMessage(schema)
}

// validateThought checks if a thought meets all requirements.
func (t *Tool) validateThought(thought *ThoughtData) error {
	if thought.Thought == "" {
		return fmt.Errorf("thought must not be empty")
	}
	if thought.ThoughtNumber < 1 {
		return fmt.Errorf("thoughtNumber must be >= 1")
	}
	if thought.TotalThoughts < 1 {
		return fmt.Errorf("totalThoughts must be >= 1")
	}
	return nil
}

// formatError creates an error response in the expected format.
func (t *Tool) formatError(msg string) interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Validation error: %s", msg),
			},
		},
		"isError": true,
	}
}

// getBranchIDs returns a list of all branch IDs.
func (t *Tool) getBranchIDs() []string {
	keys := make([]string, 0, len(t.branches))
	for k := range t.branches {
		keys = append(keys, k)
	}
	return keys
}
