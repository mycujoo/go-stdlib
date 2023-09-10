package kqlfilter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FilterSQLAllowedFieldsColumnType int

const (
	FilterSQLAllowedFieldsColumnTypeString = iota
	FilterSQLAllowedFieldsColumnTypeInt
	FilterSQLAllowedFieldsColumnTypeDouble
	FilterSQLAllowedFieldsColumnTypeBool
	FilterSQLAllowedFieldsColumnTypeDateTime
)

type FilterSQLAllowedFieldsItem struct {
	// SQL table column name. Can be omitted if the column name is equal to the key in the allowedFields map.
	ColumnName string
	// SQL column type. Defaults to FilterSQLAllowedFieldsColumnTypeString.
	ColumnType FilterSQLAllowedFieldsColumnType
	// Allow prefix matching when a wildcard (`*`) is present at the end of a string.
	// Only applicable for FilterSQLAllowedFieldsColumnTypeString. Defaults to false.
	AllowPrefixMatch bool
	// The values that the user is allowed to use in the query. Typically used for enums. Does not work in combination
	// with prefix matching. Only applicable for FilterSQLAllowedFieldsColumnTypeString. Defaults to allowing any value.
	AllowedValues []FilterSQLAllowedFieldsItemAllowedValue
}

type FilterSQLAllowedFieldsItemAllowedValue struct {
	// The value that the user provides in the filter
	InputValue string
	// The value as it is stored in the database table. Defaults to the InputValue.
	ColumnValue string
}

// ToSQL turns a Filter into a partial SQL statement. It takes a map of fields that are allowed to be queried via this
// filter (as a user should not be able to query all db columns via a filter). It returns a partial SQL statement that
// can be added to a WHERE clause, along with associated params. An example follows.
//
// Given a Filter that looks like this:
//
//	[(Field: "userId", Operator: "=", Value: "12345"), (Field: "email", Operator: "=", Value: "john@example.*")]
//
// and an allowedFields that looks like this:
//
//	{
//		"userId": (ColumnName: "user_id", ColumnType: FilterSQLAllowedFieldsColumnTypeInt,    AllowPartialMatch: false),
//		"email":  (ColumnName: "email",   ColumnType: FilterSQLAllowedFieldsColumnTypeString, AllowPartialMatch: true)
//	}
//
// This returns a slice of SQL conditions that can be appended to an existing WHERE clause (make sure to AND these first):
//
//	["user_id=@GeneratedPlaceholder0", "email LIKE @GeneratedPlaceholder1"]
//
// and params:
//
//	{
//		"@GeneratedPlaceholder0": 12345,
//		"@GeneratedPlaceholder1": "john@example.%"
//	}
//
// Note: The Clause Operator is contextually used/ignored. It only works with int, double and datetime types currently.
func (f Filter) ToSQL(allowedFields map[string]FilterSQLAllowedFieldsItem) ([]string, map[string]interface{}, error) {
	var condAnds []string
	params := map[string]interface{}{}

	for i, clause := range f.Clauses {
		if cmv, ok := allowedFields[clause.Field]; ok {
			columnName := cmv.ColumnName
			if columnName == "" {
				columnName = clause.Field
			}
			placeholderName := fmt.Sprintf("%s%d", "GeneratedPlaceholder", i)
			switch cmv.ColumnType {
			case FilterSQLAllowedFieldsColumnTypeString:
				if cmv.AllowPrefixMatch && strings.HasSuffix(clause.Value, "*") {
					// TODO: Handle escaped asterisk (*) characters that should not serve as wildcards
					condAnds = append(condAnds, fmt.Sprintf("%s LIKE @%s", columnName, placeholderName))
					escapedValue := strings.ReplaceAll(clause.Value, "%", "\\%")
					params[placeholderName] = escapedValue[0:len(escapedValue)-1] + "%"
				} else if len(cmv.AllowedValues) > 0 {
					found := false
					for _, v := range cmv.AllowedValues {
						if v.InputValue == clause.Value {
							condAnds = append(condAnds, fmt.Sprintf("%s=@%s", columnName, placeholderName))
							params[placeholderName] = v.ColumnValue
							found = true
							break
						}
					}
					if !found {
						return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
					}
				} else {
					condAnds = append(condAnds, fmt.Sprintf("%s=@%s", columnName, placeholderName))
					params[placeholderName] = clause.Value
				}
			case FilterSQLAllowedFieldsColumnTypeInt:
				intVal, err := strconv.Atoi(clause.Value)
				if err != nil {
					return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				params[placeholderName] = intVal
			case FilterSQLAllowedFieldsColumnTypeDouble:
				doubleVal, err := strconv.ParseFloat(clause.Value, 64)
				if err != nil {
					return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				params[placeholderName] = doubleVal
			case FilterSQLAllowedFieldsColumnTypeBool:
				boolVal, _ := strconv.ParseBool(clause.Value)
				condAnds = append(condAnds, fmt.Sprintf("%s IS @%s", columnName, placeholderName))
				params[placeholderName] = boolVal
			case FilterSQLAllowedFieldsColumnTypeDateTime:
				t, err := time.Parse(time.RFC3339, clause.Value)
				if err != nil {
					return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				params[placeholderName] = t
			}
		} else {
			return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
		}
	}

	return condAnds, params, nil

}

func SliceMap[T any, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, item := range in {
		out[i] = f(item)
	}
	return out
}
