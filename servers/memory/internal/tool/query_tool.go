package tool

import (
	_ "embed" // Required for go:embed directive
	"encoding/json"
	"fmt"
	"reflect"

	"mcp-go-sdk"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/types"
)

//go:embed schemas/query.json
var querySchemaJSON []byte // Use []byte for json.RawMessage

// QueryTool provides functionality to query the knowledge graph.
type QueryTool struct {
	manager *graph.KnowledgeGraphManager
}

// NewQueryTool creates a new instance of QueryTool.
func NewQueryTool(manager *graph.KnowledgeGraphManager) mcp.Tool {
	return &QueryTool{manager: manager}
}

// Name returns the name of the tool.
func (t *QueryTool) Name() string {
	return "query"
}

// Description returns a description of the tool.
func (t *QueryTool) Description() string {
	return "Performs structured queries on the knowledge graph with filtering, sorting, and pagination."
}

// Schema defines the input parameters for the tool, returning raw JSON.
func (t *QueryTool) Schema() json.RawMessage {
	return querySchemaJSON
}

// Execute runs the query logic.
// It conforms to the mcp.Tool interface.
func (t *QueryTool) Execute(params json.RawMessage) (interface{}, error) {
	var input types.QueryInput

	// Unmarshal the raw JSON parameters into the QueryInput struct
	if err := json.Unmarshal(params, &input); err != nil {
		// Try unmarshalling from a string representation if direct unmarshal fails
		// This handles cases where the framework might pass the params as a JSON string
		var paramsStr string
		if errStr := json.Unmarshal(params, &paramsStr); errStr == nil {
			if errRetry := json.Unmarshal([]byte(paramsStr), &input); errRetry != nil {
				// Return the error within the expected format if needed by framework, else just error
				// Assuming simple error return is fine based on update_entities.go
				return nil, fmt.Errorf("failed to parse query parameters (tried string): %w", errRetry)
			}
		} else {
			// If it wasn't a string and direct unmarshal failed, return the original error
			return nil, fmt.Errorf("failed to parse query parameters: %w", err)
		}
	}

	// Validate required fields after unmarshalling
	if input.TargetType == "" {
		return nil, fmt.Errorf("missing required parameter 'target_type'")
	}
	if input.TargetType != types.QueryTargetEntity && input.TargetType != types.QueryTargetRelation {
		return nil, fmt.Errorf("invalid value for 'target_type': must be '%s' or '%s'", types.QueryTargetEntity, types.QueryTargetRelation)
	}

	// Validate optional fields if present
	if input.SortOrder != "" && input.SortOrder != types.SortOrderAsc && input.SortOrder != types.SortOrderDesc {
		return nil, fmt.Errorf("invalid value for 'sort_order': must be '%s' or '%s'", types.SortOrderAsc, types.SortOrderDesc)
	}

	for i, filter := range input.Filters {
		if filter.Field == "" || filter.Operator == "" {
			return nil, fmt.Errorf("filter at index %d is missing 'field' or 'operator'", i)
		}

		// Validate operator
		isValidOperator := false
		switch filter.Operator {
		case types.OperatorEqual, types.OperatorNotEqual, types.OperatorIn, types.OperatorNotIn,
			types.OperatorContains, types.OperatorGreaterThan, types.OperatorGreaterThanOrEqual,
			types.OperatorLessThan, types.OperatorLessThanOrEqual:
			isValidOperator = true
		}
		if !isValidOperator {
			return nil, fmt.Errorf("filter at index %d has invalid 'operator': %s", i, filter.Operator)
		}

		// Basic check for value presence
		if filter.Value == nil {
			// Allow nil value only for eq and neq operators
			if filter.Operator != types.OperatorEqual && filter.Operator != types.OperatorNotEqual {
				return nil, fmt.Errorf("filter at index %d is missing 'value' (required for operator '%s')", i, filter.Operator)
			}
		} else {
			// Type check for operators requiring arrays
			if filter.Operator == types.OperatorIn || filter.Operator == types.OperatorNotIn {
				// Use reflection to check if the value is a slice
				valType := reflect.TypeOf(filter.Value)
				if valType.Kind() != reflect.Slice {
					return nil, fmt.Errorf("filter at index %d: 'value' must be an array/slice for operator '%s', got %T", i, filter.Operator, filter.Value)
				}
				// Check if slice is empty - might be valid depending on desired behavior, allow for now.
				// valSlice := reflect.ValueOf(filter.Value)
				// if valSlice.Len() == 0 {
				// 	return nil, fmt.Errorf("filter at index %d: 'value' array/slice cannot be empty for operator '%s'", i, filter.Operator)
				// }
			}
			// Add more type checks here if needed for other operators (e.g., ensure string for 'contains')
			// For now, rely on execution logic in query.go to handle type mismatches for gt/lt/contains etc.
		}
	}

	// Call the manager's query method
	output, err := t.manager.Query(input) // Changed Manager to manager
	if err != nil {
		// Error might need formatting depending on tool execution requirements
		// Format the error using the helper
		return formatError(fmt.Errorf("query execution failed: %w", err)), nil
	}

	// Format the successful response using the helper
	return formatResponse(
		fmt.Sprintf("Query successful, found %d total matching items.", output.Total),
		map[string]interface{}{ // Optional metadata
			"total_matches": output.Total,
			"limit":         input.Limit,
			"offset":        input.Offset,
			"result_count":  len(output.Results),
		},
		output, // Include the full QueryOutput as data
	), nil
}
