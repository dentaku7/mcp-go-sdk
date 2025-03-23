package tool

import (
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

// AddObservationsTool implements the Tool interface for adding observations
type AddObservationsTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewAddObservationsTool creates a new AddObservationsTool instance
func NewAddObservationsTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &AddObservationsTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *AddObservationsTool) Name() string {
	return "add_observations"
}

// Description returns the description of the tool
func (t *AddObservationsTool) Description() string {
	return "Adds new observations to entities in the knowledge graph"
}

// Schema returns the JSON schema for the tool's parameters
func (t *AddObservationsTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"observations": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": {
							"type": "string",
							"description": "Optional. A unique identifier for the observation. If not provided, one will be generated."
						},
						"entity_id": {
							"type": "string",
							"description": "The ID of the entity this observation belongs to"
						},
						"type": {
							"type": "string",
							"description": "The type of the observation"
						},
						"content": {
							"type": "string",
							"description": "The content of the observation"
						},
						"description": {
							"type": "string",
							"description": "Optional. A description of the observation"
						},
						"metadata": {
							"type": "object",
							"description": "Optional. Additional metadata for the observation",
							"additionalProperties": true
						}
					},
					"required": ["entity_id", "type", "content"]
				}
			}
		},
		"required": ["observations"]
	}`)
}

// Execute adds new observations to the knowledge graph
func (t *AddObservationsTool) Execute(params json.RawMessage) (interface{}, error) {
	var input struct {
		Observations []types.Observation `json:"observations"`
	}

	if err := json.Unmarshal(params, &input); err != nil {
		return formatError(fmt.Errorf("failed to parse input: %w", err)), nil
	}

	createdObservations, err := t.manager.AddObservations(input.Observations)
	if err != nil {
		return formatError(fmt.Errorf("failed to add observations: %w", err)), nil
	}

	return formatResponse(
		fmt.Sprintf("Successfully added %d observations", len(createdObservations)),
		map[string]interface{}{
			"input_count":   len(input.Observations),
			"created_count": len(createdObservations),
			"created_items": createdObservations,
		},
		createdObservations,
	), nil
}
