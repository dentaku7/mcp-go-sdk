package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"
	"time"

	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/get_entity_timeline.json
var getEntityTimelineSchemaJSON []byte // Use []byte for json.RawMessage

// GetEntityTimelineToolInput defines the expected input structure for the tool
type GetEntityTimelineToolInput struct {
	EntityID        string    `json:"entity_id"`            // Required: The ID of the entity to retrieve the timeline for.
	StartTime       *string   `json:"start_time,omitempty"` // Optional: The start of the time range (RFC3339 format). If omitted, no lower time bound is applied.
	EndTime         *string   `json:"end_time,omitempty"`   // Optional: The end of the time range (RFC3339 format). If omitted, no upper time bound is applied.
	ObservationType *string   `json:"type,omitempty"`       // Optional: Filter observations by this specific type.
	Tags            *[]string `json:"tags,omitempty"`       // Optional: Filter observations that have ANY of these tags.
}

// GetEntityTimelineToolOutput defines the expected output structure for the tool
type GetEntityTimelineToolOutput struct {
	Timeline []types.Observation `json:"timeline"` // Chronologically sorted list of matching observations.
}

// GetEntityTimelineTool implements the tool interface
type GetEntityTimelineTool struct {
	Manager *graph.KnowledgeGraphManager
}

// NewGetEntityTimelineTool creates a new instance of GetEntityTimelineTool
func NewGetEntityTimelineTool(manager *graph.KnowledgeGraphManager) *GetEntityTimelineTool {
	return &GetEntityTimelineTool{Manager: manager}
}

// Name returns the name of the tool
func (t *GetEntityTimelineTool) Name() string {
	return "get_entity_timeline"
}

// Description returns the description of the tool
func (t *GetEntityTimelineTool) Description() string {
	return "Retrieves a chronological timeline of observations for a specific entity, optionally filtered by time range, observation type, and tags."
}

// Schema returns the JSON schema for the tool's parameters (input only).
func (t *GetEntityTimelineTool) Schema() json.RawMessage {
	// Return the embedded input schema content directly
	return getEntityTimelineSchemaJSON
}

// Execute runs the tool with the given input, conforming to the mcp.Tool interface
func (t *GetEntityTimelineTool) Execute(inputRaw json.RawMessage) (interface{}, error) {
	var input GetEntityTimelineToolInput
	if err := json.Unmarshal(inputRaw, &input); err != nil {
		return formatError(fmt.Errorf("failed to unmarshal input JSON for %s: %w", t.Name(), err)), nil
	}

	// Validate required fields
	if input.EntityID == "" {
		return formatError(fmt.Errorf("missing required field 'entity_id' in input for %s", t.Name())), nil
	}

	// Parse time strings
	var startTime, endTime time.Time
	var err error
	if input.StartTime != nil && *input.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, *input.StartTime)
		if err != nil {
			return formatError(fmt.Errorf("invalid 'start_time' format (expected RFC3339): %w", err)), nil
		}
	}
	if input.EndTime != nil && *input.EndTime != "" {
		endTime, err = time.Parse(time.RFC3339, *input.EndTime)
		if err != nil {
			return formatError(fmt.Errorf("invalid 'end_time' format (expected RFC3339): %w", err)), nil
		}
	}

	// Get optional filter values
	var observationType string
	if input.ObservationType != nil {
		observationType = *input.ObservationType
	}
	var tags []string
	if input.Tags != nil {
		tags = *input.Tags
	}

	// Call the manager function
	timeline, err := t.Manager.GetEntityTimeline(input.EntityID, startTime, endTime, observationType, tags)
	if err != nil {
		return formatError(fmt.Errorf("failed to get entity timeline: %w", err)), nil
	}

	// Prepare the output struct
	output := GetEntityTimelineToolOutput{
		Timeline: timeline,
	}

	// Format the successful response
	return formatResponse(
		fmt.Sprintf("Successfully retrieved timeline for entity '%s' with %d observations.", input.EntityID, len(timeline)),
		map[string]interface{}{
			"entity_id":        input.EntityID,
			"observation_type": input.ObservationType, // Include filters used
			"tags":             input.Tags,
			"start_time":       input.StartTime,
			"end_time":         input.EndTime,
			"result_count":     len(timeline),
		},
		output, // Include the timeline data
	), nil
}
