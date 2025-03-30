package tool

// ContextKey defines a type for context keys to avoid collisions.
type ContextKey string

// GraphManagerKey is the key used to store the KnowledgeGraphManager in the runtime context.
const GraphManagerKey ContextKey = "graphManager"

// formatResponse formats a successful response in the MCP tool format
func formatResponse(message string, metadata map[string]interface{}, data interface{}) interface{} {
	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": message,
			},
		},
		"metadata": metadata,
		"isError":  false,
	}

	if data != nil {
		response["data"] = data
	}

	return response
}

// formatError formats an error response in the MCP tool format
func formatError(err error) interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": err.Error(),
			},
		},
		"isError": true,
	}
}
