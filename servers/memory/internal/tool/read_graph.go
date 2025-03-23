package tool

import (
	"encoding/json"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
)

// ReadGraphTool implements the Tool interface for reading the entire graph
type ReadGraphTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewReadGraphTool creates a new ReadGraphTool instance
func NewReadGraphTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &ReadGraphTool{
		manager: manager,
	}
}

// Name returns the name of the tool
func (t *ReadGraphTool) Name() string {
	return "read_graph"
}

// Description returns the description of the tool
func (t *ReadGraphTool) Description() string {
	return "Returns the current state of the knowledge graph"
}

// Schema returns the JSON schema for the tool's parameters
func (t *ReadGraphTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)
}

// Execute returns the current state of the knowledge graph
func (t *ReadGraphTool) Execute(params json.RawMessage) (interface{}, error) {
	graph := t.manager.ReadGraph()

	return formatResponse(
		"Successfully retrieved graph state",
		map[string]interface{}{
			"entity_count":      len(graph.Entities),
			"relation_count":    len(graph.Relations),
			"observation_count": len(graph.Observations),
		},
		graph,
	), nil
}
