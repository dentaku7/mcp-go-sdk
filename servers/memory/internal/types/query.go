package types

// QueryInput defines the structure for incoming query requests.
type QueryInput struct {
	Filters    []Filter `json:"filters,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"`    // Field to sort by (e.g., "id", "type", "metadata.name")
	SortOrder  string   `json:"sort_order,omitempty"` // "asc" or "desc"
	Limit      int      `json:"limit,omitempty"`      // Max number of results to return
	Offset     int      `json:"offset,omitempty"`     // Number of results to skip
	TargetType string   `json:"target_type"`          // "entity" or "relation" (determines result type)
}

// Filter defines a single filtering condition.
type Filter struct {
	Field    string      `json:"field"`    // Field to filter on (e.g., "id", "type", "metadata.status")
	Operator string      `json:"operator"` // Operator (e.g., "eq", "neq") - initially basic
	Value    interface{} `json:"value"`    // Value to compare against
}

// QueryOutput defines the structure for query results.
type QueryOutput struct {
	Results []interface{} `json:"results"` // Holds the actual results (e.g., []Entity, []Relation)
	Total   int           `json:"total"`   // Total number of matching items before pagination
	Limit   int           `json:"limit"`   // The limit applied
	Offset  int           `json:"offset"`  // The offset applied
}

const (
	QueryTargetEntity          = "entity"
	QueryTargetRelation        = "relation"
	SortOrderAsc               = "asc"
	SortOrderDesc              = "desc"
	OperatorEqual              = "eq"
	OperatorNotEqual           = "neq"
	OperatorIn                 = "in"
	OperatorNotIn              = "nin"
	OperatorContains           = "contains"
	OperatorGreaterThan        = "gt"
	OperatorGreaterThanOrEqual = "gte"
	OperatorLessThan           = "lt"
	OperatorLessThanOrEqual    = "lte"
)
