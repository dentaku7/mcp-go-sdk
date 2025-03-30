package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/add_observations.json
var addObservationsSchemaJSON []byte // Use []byte for json.RawMessage

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
	return addObservationsSchemaJSON
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
