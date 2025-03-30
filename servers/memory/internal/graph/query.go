package graph

import (
	"fmt"
	"mcp-memory/internal/types"
	"reflect"
	"sort"
	"strings"
)

// compareValues attempts a type-aware comparison.
// It tries to handle common types like string, float64, int, bool.
// Returns true if values are considered equal, false otherwise, and error on failure.
func compareValues(actual interface{}, expected interface{}) (bool, error) {
	// Use reflect for more robust comparison, but start with common types
	actVal := reflect.ValueOf(actual)
	expVal := reflect.ValueOf(expected)

	// If types are directly comparable
	if actVal.Type() == expVal.Type() {
		return reflect.DeepEqual(actual, expected), nil
	}

	// Handle common numeric type conversions (e.g., filter provides int, actual is float64)
	if actVal.CanFloat() && expVal.CanFloat() {
		return actVal.Float() == expVal.Float(), nil
	}
	if actVal.CanInt() && expVal.CanInt() {
		return actVal.Int() == expVal.Int(), nil
	}
	// Add CanUint if needed

	// If types differ and aren't easily convertible numerics, try string comparison as fallback
	// This maintains some flexibility but is less precise than strict type matching.
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	return actualStr == expectedStr, nil

	// Alternative: Return error if types are incompatible and not handled above
	// return false, fmt.Errorf("type mismatch: cannot compare %T and %T", actual, expected)
}

// valueToFloat64 attempts to convert a reflect.Value to float64.
// Handles float, int, and uint types.
func valueToFloat64(val reflect.Value) (float64, error) {
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return val.Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// Be cautious with large uint64 values that might exceed float64 precision
		return float64(val.Uint()), nil
	default:
		return 0, fmt.Errorf("cannot convert type %s to float64", val.Type())
	}
}

// matchesFilter checks if a single item matches a given filter.
func (m *KnowledgeGraphManager) matchesFilter(item interface{}, filter types.Filter) (bool, error) {
	value, err := m.extractValue(item, filter.Field)
	if err != nil {
		// If field doesn't exist for metadata, it's a non-match, not necessarily an error for filtering.
		if strings.HasPrefix(filter.Field, "metadata.") {
			// Check if the operator is NEQ. If filtering for field != value, and field doesn't exist, it *should* match.
			if filter.Operator == types.OperatorNotEqual {
				// We need to know if the filter.Value is nil. If filter.Value is non-nil, then missing field is != non-nil value.
				// This gets complex quickly. Let's simplify: treat missing field as non-match for EQ, potential match for NEQ later?
				// For now, keep simple: missing field is non-match for EQ.
				return false, nil
			}
			// If operator is EQ, missing field is definitely not equal to any filter.Value.
			return false, nil
		}
		// For non-metadata fields, extraction error is a real problem.
		return false, fmt.Errorf("error extracting field '%s': %w", filter.Field, err)
	}

	// --- Handle nil comparisons explicitly first ---
	// If the extracted value is nil
	if value == nil {
		switch filter.Operator {
		case types.OperatorEqual:
			return filter.Value == nil, nil // True only if filter value is also nil
		case types.OperatorNotEqual:
			return filter.Value != nil, nil // True only if filter value is not nil
		case types.OperatorIn:
			// Is nil in the list of filter values?
			filterValueSlice, ok := filter.Value.([]interface{})
			if !ok {
				// This should be caught by validation, but handle defensively
				return false, fmt.Errorf("invalid value type for 'in' operator with nil field value: expected slice, got %T", filter.Value)
			}
			for _, v := range filterValueSlice {
				if v == nil {
					return true, nil // Found nil in the list
				}
			}
			return false, nil // Did not find nil in the list
		case types.OperatorNotIn:
			// Is nil NOT in the list of filter values?
			filterValueSlice, ok := filter.Value.([]interface{})
			if !ok {
				return false, fmt.Errorf("invalid value type for 'nin' operator with nil field value: expected slice, got %T", filter.Value)
			}
			for _, v := range filterValueSlice {
				if v == nil {
					return false, nil // Found nil in the list, so it IS in the list
				}
			}
			return true, nil // Did not find nil in the list
		default:
			// For gt, lt, contains etc., comparing with nil doesn't make sense / results in false.
			return false, nil
		}
	}

	// If the filter value is nil (and extracted value is not nil)
	if filter.Value == nil {
		// We already handled value == nil above. So here, value is guaranteed non-nil.
		switch filter.Operator {
		case types.OperatorEqual:
			return false, nil // Non-nil value cannot equal nil filter value
		case types.OperatorNotEqual:
			return true, nil // Non-nil value is not equal to nil filter value
		default:
			// gt, lt, contains, in, nin with a nil filter value don't make sense / result in false.
			// Return false, but maybe error is better? Let's error for clarity.
			return false, fmt.Errorf("unsupported operator %s for nil filter value comparison on field '%s'", filter.Operator, filter.Field)
		}
	}

	// --- Handle non-nil comparisons ---
	switch filter.Operator {
	case types.OperatorEqual, types.OperatorNotEqual,
		types.OperatorGreaterThan, types.OperatorGreaterThanOrEqual,
		types.OperatorLessThan, types.OperatorLessThanOrEqual:
		// Delegate standard comparisons to performComparison
		// Note: compareValues was already called above for basic eq/neq check,
		// but performComparison will handle gt/lt/etc. and potentially refine eq/neq.
		compMatch, err := performComparison(value, filter.Value, filter.Operator)
		if err != nil {
			return false, fmt.Errorf("comparison error for field '%s' (operator %s): %w", filter.Field, filter.Operator, err)
		}
		return compMatch, nil

	case types.OperatorIn:
		filterValueSlice, ok := filter.Value.([]interface{})
		if !ok {
			// Should be caught by validation, but handle defensively
			return false, fmt.Errorf("invalid value type for 'in' operator: expected slice, got %T", filter.Value)
		}
		if len(filterValueSlice) == 0 {
			return false, nil // Cannot be 'in' an empty set
		}
		for _, v := range filterValueSlice {
			// Use compareValues for equality check within the 'in' list
			equal, err := compareValues(value, v)
			if err != nil {
				// If comparison fails for one element, maybe continue or fail? Let's fail fast for now.
				return false, fmt.Errorf("error comparing value for 'in' operator on field '%s': %w", filter.Field, err)
			}
			if equal {
				return true, nil // Found a match
			}
		}
		return false, nil // No match found in the list

	case types.OperatorNotIn:
		filterValueSlice, ok := filter.Value.([]interface{})
		if !ok {
			return false, fmt.Errorf("invalid value type for 'nin' operator: expected slice, got %T", filter.Value)
		}
		if len(filterValueSlice) == 0 {
			return true, nil // Is not 'in' an empty set
		}
		for _, v := range filterValueSlice {
			equal, err := compareValues(value, v)
			if err != nil {
				return false, fmt.Errorf("error comparing value for 'nin' operator on field '%s': %w", filter.Field, err)
			}
			if equal {
				return false, nil // Found a match, so it IS in the list
			}
		}
		return true, nil // No match found, so it is NOT in the list

	case types.OperatorContains:
		// Primarily for strings.
		valueStr, okValue := value.(string)
		filterStr, okFilter := filter.Value.(string)

		if !okValue || !okFilter {
			// Consider if we want to support 'contains' for slices/arrays later.
			// For now, strict string comparison.
			// Return false instead of error? Let's error for now to indicate misuse.
			return false, fmt.Errorf("invalid type for 'contains' operator on field '%s': both field value (%T) and filter value (%T) must be strings", filter.Field, value, filter.Value)
		}
		return strings.Contains(valueStr, filterStr), nil

	default:
		// Should be caught by tool input validation, but check anyway.
		return false, fmt.Errorf("unsupported filter operator '%s' encountered during evaluation", filter.Operator)
	}
}

// performComparison handles detailed comparison logic for operators like eq, neq, gt, gte, lt, lte.
// It attempts type-aware comparison, focusing on numbers and strings.
func performComparison(actual interface{}, expected interface{}, operator string) (bool, error) {
	actVal := reflect.ValueOf(actual)
	expVal := reflect.ValueOf(expected)

	// --- Handle Numeric Comparisons ---
	// Try converting both to float64 if possible
	canActualFloat := actVal.CanFloat() || actVal.CanInt() || actVal.CanUint()
	canExpectedFloat := expVal.CanFloat() || expVal.CanInt() || expVal.CanUint()

	if canActualFloat && canExpectedFloat {
		var actualF, expectedF float64
		var err error

		actualF, err = valueToFloat64(actVal)
		if err != nil {
			return false, fmt.Errorf("failed to convert actual value (%v) to float64: %w", actual, err)
		}
		expectedF, err = valueToFloat64(expVal)
		if err != nil {
			return false, fmt.Errorf("failed to convert expected value (%v) to float64: %w", expected, err)
		}

		switch operator {
		case types.OperatorEqual:
			return actualF == expectedF, nil
		case types.OperatorNotEqual:
			return actualF != expectedF, nil
		case types.OperatorGreaterThan:
			return actualF > expectedF, nil
		case types.OperatorGreaterThanOrEqual:
			return actualF >= expectedF, nil
		case types.OperatorLessThan:
			return actualF < expectedF, nil
		case types.OperatorLessThanOrEqual:
			return actualF <= expectedF, nil
		default:
			return false, fmt.Errorf("unknown numeric operator %s", operator)
		}
	}

	// --- Handle String Comparisons (if not numeric) ---
	// Use fmt.Sprintf as a fallback for general comparison if types allow string representation
	actualStr, okActualStr := actual.(string)
	expectedStr, okExpectedStr := expected.(string)

	// If both are explicitly strings, compare them directly
	if okActualStr && okExpectedStr {
		switch operator {
		case types.OperatorEqual:
			return actualStr == expectedStr, nil
		case types.OperatorNotEqual:
			return actualStr != expectedStr, nil
		case types.OperatorGreaterThan:
			return actualStr > expectedStr, nil // Lexicographical comparison
		case types.OperatorGreaterThanOrEqual:
			return actualStr >= expectedStr, nil
		case types.OperatorLessThan:
			return actualStr < expectedStr, nil
		case types.OperatorLessThanOrEqual:
			return actualStr <= expectedStr, nil
		default:
			return false, fmt.Errorf("unknown string operator %s", operator)
		}
	}

	// --- Fallback/Default Comparison (using original compareValues for eq/neq) ---
	// If not clearly numeric or string, fall back to the original compareValues for basic equality/inequality.
	// This handles cases like bools or where DeepEqual might work.
	// gt/lt comparisons for incompatible types will result in an error here.
	switch operator {
	case types.OperatorEqual:
		equal, err := compareValues(actual, expected) // Uses DeepEqual, numeric conversion, and string fallback
		if err != nil {
			return false, fmt.Errorf("equality comparison failed: %w", err)
		}
		return equal, nil
	case types.OperatorNotEqual:
		equal, err := compareValues(actual, expected)
		if err != nil {
			return false, fmt.Errorf("inequality comparison failed: %w", err)
		}
		return !equal, nil
	case types.OperatorGreaterThan, types.OperatorGreaterThanOrEqual,
		types.OperatorLessThan, types.OperatorLessThanOrEqual:
		// Placeholder - Requires proper numeric/string/etc. comparison logic
		// This will be implemented in the next step.
		return false, fmt.Errorf("operator %s not fully implemented yet in performComparison", operator)
	default:
		return false, fmt.Errorf("unknown operator %s passed to performComparison", operator)
	}
}

// applyFilters filters a slice based on the provided filter criteria.
func (m *KnowledgeGraphManager) applyFilters(items []interface{}, filters []types.Filter) ([]interface{}, error) {
	if len(filters) == 0 {
		return items, nil // No filters to apply
	}

	filteredItems := make([]interface{}, 0, len(items))
	for _, item := range items {
		matchesAll := true
		for _, filter := range filters {
			match, err := m.matchesFilter(item, filter)
			if err != nil {
				// Fail fast on error during filter evaluation
				return nil, fmt.Errorf("error evaluating filter (%v): %w", filter, err)
			}
			if !match {
				matchesAll = false
				break // No need to check other filters for this item
			}
		}
		if matchesAll {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems, nil
}

// extractValue retrieves the value of a specified field from an entity or relation.
func (m *KnowledgeGraphManager) extractValue(item interface{}, field string) (interface{}, error) {
	switch v := item.(type) {
	case types.Entity:
		switch field {
		case "id":
			return v.ID, nil
		case "type":
			return v.Type, nil
		case "name":
			return v.Name, nil
		case "description":
			return v.Description, nil
		default:
			// Handle metadata access
			if strings.HasPrefix(field, "metadata.") {
				key := strings.TrimPrefix(field, "metadata.")
				if v.Metadata != nil {
					if val, ok := v.Metadata[key]; ok {
						return val, nil
					}
				}
				return nil, fmt.Errorf("metadata key '%s' not found", key)
			}
			return nil, fmt.Errorf("unknown entity field: %s", field)
		}
	case types.Relation:
		switch field {
		case "id":
			return v.ID, nil
		case "type":
			return v.Type, nil
		case "source":
			return v.Source, nil
		case "target":
			return v.Target, nil
		default:
			// Handle relation metadata access
			if strings.HasPrefix(field, "metadata.") {
				key := strings.TrimPrefix(field, "metadata.")
				if v.Metadata != nil {
					if val, ok := v.Metadata[key]; ok {
						return val, nil
					}
				}
				return nil, fmt.Errorf("metadata key '%s' not found", key)
			}
			return nil, fmt.Errorf("unknown relation field: %s", field)
		}
	default:
		return nil, fmt.Errorf("unsupported item type for value extraction: %T", item)
	}
}

// applySorting sorts a slice based on the given field and order.
func (m *KnowledgeGraphManager) applySorting(items []interface{}, sortBy string, sortOrder string) error {
	if len(items) == 0 {
		return nil
	}

	order := types.SortOrderAsc
	if strings.ToLower(sortOrder) == types.SortOrderDesc {
		order = types.SortOrderDesc
	}

	sort.SliceStable(items, func(i, j int) bool {
		valI, errI := m.extractValue(items[i], sortBy)
		valJ, errJ := m.extractValue(items[j], sortBy)

		// Handle errors during value extraction (e.g., field not found)
		// Treat error/missing field as "less than" existing field for stable sort
		if errI != nil && errJ == nil {
			return order == types.SortOrderAsc // If ASC, item with error comes first
		}
		if errI == nil && errJ != nil {
			return order == types.SortOrderDesc // If DESC, item with error comes first
		}
		if errI != nil && errJ != nil {
			return false // Both errored, maintain order
		}

		// --- Type-Aware Sorting Logic ---

		// 1. Try Numeric Comparison
		valIRef := reflect.ValueOf(valI)
		valJRef := reflect.ValueOf(valJ)
		canIFloat := valIRef.CanFloat() || valIRef.CanInt() || valIRef.CanUint()
		canJFloat := valJRef.CanFloat() || valJRef.CanInt() || valJRef.CanUint()

		if canIFloat && canJFloat {
			floatI, errConvI := valueToFloat64(valIRef)
			floatJ, errConvJ := valueToFloat64(valJRef)

			// If conversion is successful for both, compare numerically
			if errConvI == nil && errConvJ == nil {
				if floatI == floatJ {
					return false // Equal, maintain order
				}
				if order == types.SortOrderAsc {
					return floatI < floatJ
				}
				// else order is DESC
				return floatI > floatJ
			}
			// If conversion failed for one but not the other, treat error as "less"
			if errConvI != nil && errConvJ == nil {
				return order == types.SortOrderAsc
			}
			if errConvI == nil && errConvJ != nil {
				return order == types.SortOrderDesc
			}
			// If both conversions failed, fall through to string comparison
		}

		// 2. Try String Comparison
		strI, okI := valI.(string)
		strJ, okJ := valJ.(string)
		if okI && okJ {
			if strI == strJ {
				return false // Equal, maintain order
			}
			if order == types.SortOrderAsc {
				return strI < strJ
			}
			// else order is DESC
			return strI > strJ
		}

		// 3. Fallback to fmt.Sprintf Comparison (for other types or mixed types)
		fmtStrI := fmt.Sprintf("%v", valI)
		fmtStrJ := fmt.Sprintf("%v", valJ)
		if fmtStrI == fmtStrJ {
			return false // Equal, maintain order
		}
		if order == types.SortOrderAsc {
			return fmtStrI < fmtStrJ
		}
		// else order is DESC
		return fmtStrI > fmtStrJ
	})

	return nil
}

// applyPagination slices the items based on limit and offset.
func (m *KnowledgeGraphManager) applyPagination(items []interface{}, limit, offset int) []interface{} {
	// Set default limit if not specified or invalid
	if limit <= 0 {
		limit = 100 // Or another sensible default/max
	}
	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	total := len(items)

	if offset >= total {
		return []interface{}{} // Offset is beyond the total number of items
	}

	end := offset + limit
	if end > total {
		end = total // Adjust end if it exceeds the total number of items
	}

	return items[offset:end]
}

// Helper to convert concrete slice to []interface{}
func interfaceSlice[T any](slice []T) []interface{} {
	is := make([]interface{}, len(slice))
	for i, v := range slice {
		is[i] = v
	}
	return is
}

// Query performs structured queries on the graph with filtering, sorting, and pagination.
func (m *KnowledgeGraphManager) Query(input types.QueryInput) (types.QueryOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var candidates []interface{}

	// --- 1. Select Target & Get Initial Candidates ---
	switch input.TargetType {
	case types.QueryTargetEntity:
		allEntities := make([]types.Entity, 0, len(m.graph.Entities))
		for _, entity := range m.graph.Entities {
			allEntities = append(allEntities, entity)
		}
		candidates = interfaceSlice(allEntities)
	case types.QueryTargetRelation:
		allRelations := make([]types.Relation, 0, len(m.graph.Relations))
		for _, relation := range m.graph.Relations {
			allRelations = append(allRelations, relation)
		}
		candidates = interfaceSlice(allRelations)
	default:
		return types.QueryOutput{}, fmt.Errorf("unsupported target_type: %s", input.TargetType)
	}

	// --- 2. Apply Filtering ---
	filteredResults, err := m.applyFilters(candidates, input.Filters)
	if err != nil {
		// If filtering itself fails (e.g., invalid operator), return the error
		return types.QueryOutput{}, fmt.Errorf("query filtering failed: %w", err)
	}

	// --- 3. Get Total Count (Before Pagination) ---
	totalCount := len(filteredResults)

	// --- 4. Apply Sorting ---
	if input.SortBy != "" {
		err = m.applySorting(filteredResults, input.SortBy, input.SortOrder)
		if err != nil {
			// If sorting fails, return the error
			return types.QueryOutput{}, fmt.Errorf("query sorting failed: %w", err)
		}
	}
	// Note: applySorting sorts in place, so filteredResults is now sorted

	// --- 5. Apply Pagination ---
	pagedResults := m.applyPagination(filteredResults, input.Limit, input.Offset)

	// --- 6. Construct Output ---
	// Ensure limit and offset used in output reflect defaults/adjustments
	limit := input.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	output := types.QueryOutput{
		Results: pagedResults,
		Total:   totalCount,
		Limit:   limit,  // Return the actual limit potentially used
		Offset:  offset, // Return the actual offset potentially used
	}

	return output, nil
}
