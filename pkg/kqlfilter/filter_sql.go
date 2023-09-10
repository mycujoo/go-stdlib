package kqlfilter

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	FilterSQLAllowedFieldsItemString = iota
	FilterSQLAllowedFieldsItemInt
	FilterSQLAllowedFieldsItemDouble
	FilterSQLAllowedFieldsItemBool
	FilterSQLAllowedFieldsItemDateTime
)

type FilterSQLAllowedFieldsItem struct {
	// SQL table column name. Can be omitted if the column name is equal to the key in the column map
	ColumnName string
	// SQL column type
	ColumnType int
	// Allow prefix matching when a wildcard (`*`) is present at the end of a string.
	// Only applicable for FilterSQLAllowedFieldsItemString
	AllowPrefixMatch bool
}

// toSQL turns a Filter into a partial SQL statement. It takes a map of fields that are allowed to be queried via this
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
//		"userId": (ColumnName: "user_id", ColumnType: FilterColumnMapValueInt,    AllowPartialMatch: false),
//		"email":  (ColumnName: "email",   ColumnType: FilterColumnMapValueString, AllowPartialMatch: true)
//	}
//
// This returns a partial SQL statement that can be appended to an existing WHERE clause:
//
//	(user_id=@GeneratedPlaceholder0 AND email LIKE @GeneratedPlaceholder1)
//
// and params:
//
//	{
//		"@GeneratedPlaceholder0": 12345,
//		"@GeneratedPlaceholder1": "john@example.%"
//	}
//
// Note: The Clause Operator is contextually used/ignored. It only works with int, double and datetime types currently.
func (f Filter) toSQL(allowedFields map[string]FilterSQLAllowedFieldsItem) (string, map[string]any, error) {
	var condAnds []string
	params := map[string]any{}

	for i, clause := range f.Clauses {
		if cmv, ok := allowedFields[clause.Field]; ok {
			columnName := cmv.ColumnName
			if columnName == "" {
				columnName = clause.Field
			}
			placeholderName := fmt.Sprintf("%s%d", "GeneratedPlaceholder", i)
			switch cmv.ColumnType {
			case FilterSQLAllowedFieldsItemString:
				if cmv.AllowPrefixMatch && strings.HasSuffix(clause.Value, "*") {
					// TODO: Handle escaped asterisk (*) characters that should not serve as wildcards
					condAnds = append(condAnds, fmt.Sprintf("%s LIKE @%s", columnName, placeholderName))
					escapedValue := strings.ReplaceAll(clause.Value, "%", "\\%")
					params[placeholderName] = escapedValue[0:len(escapedValue)-1] + "%"
				} else {
					condAnds = append(condAnds, fmt.Sprintf("%s=@%s", columnName, placeholderName))
					params[placeholderName] = clause.Value
				}
			case FilterSQLAllowedFieldsItemInt:
				intVal, err := strconv.Atoi(clause.Value)
				if err != nil {
					return "", map[string]any{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				params[placeholderName] = intVal
			case FilterSQLAllowedFieldsItemDouble:
				doubleVal, err := strconv.ParseFloat(clause.Value, 64)
				if err != nil {
					return "", map[string]any{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				params[placeholderName] = doubleVal
			case FilterSQLAllowedFieldsItemBool:
				boolVal, _ := strconv.ParseBool(clause.Value)
				condAnds = append(condAnds, fmt.Sprintf("%s IS @%s", columnName, placeholderName))
				params[placeholderName] = boolVal
			case FilterSQLAllowedFieldsItemDateTime:
				// Should we validate that this string is actually a datetime?
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				params[placeholderName] = clause.Value
			}
		} else {
			return "", map[string]any{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
		}
	}

	if len(condAnds) == 0 {
		return "", params, nil
	}
	sql := "(" + strings.Join(condAnds, " AND ") + ")"
	return sql, params, nil
}
