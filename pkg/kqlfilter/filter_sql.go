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

// toStandardSQL turns a Filter into a partial SQL statement. It takes a map of columns that are allowed to be queried via this
// filter. It returns a SQL clause that can be added to a WHERE clause, along with associated params.
// An example follows.
//
// Given a Filter that looks like this:
//
//	[(Field: "userId", Value: "12345"), (Field: "email", Value: "*@example.com")]
//
// and an allowedFields that looks like this:
//
//	{
//		"userId": (ColumnName: "user_id", ColumnType: FilterColumnMapValueInt,    AllowPartialMatch: false),
//		"email":  (ColumnName: "email",   ColumnType: FilterColumnMapValueString, AllowPartialMatch: true)
//	}
//
// This returns a (partial) WHERE clause:
//
//	(user_id=@GeneratedPlaceholder0 AND email LIKE %@GeneratedPlaceholder1)
//
// and params:
//
//	{
//		"@GeneratedPlaceholder0": 12345,
//		"@GeneratedPlaceholder1": "@example.com"
//	}
func (f Filter) toStandardSQL(allowedFields map[string]FilterSQLAllowedFieldsItem) (string, map[string]any, error) {
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
					condAnds = append(condAnds, columnName+" LIKE @"+placeholderName)
					escapedValue := strings.ReplaceAll(clause.Value, "%", "\\%")
					params[placeholderName] = escapedValue[0:len(escapedValue)-1] + "%"
				} else {
					condAnds = append(condAnds, columnName+" = @"+placeholderName)
					params[placeholderName] = clause.Value
				}
			case FilterSQLAllowedFieldsItemInt:
				intVal, err := strconv.Atoi(clause.Value)
				if err != nil {
					return "", map[string]any{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				condAnds = append(condAnds, columnName+" = @"+placeholderName)
				params[placeholderName] = intVal
			case FilterSQLAllowedFieldsItemDouble:
				doubleVal, err := strconv.ParseFloat(clause.Value, 64)
				if err != nil {
					return "", map[string]any{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				condAnds = append(condAnds, columnName+" = @"+placeholderName)
				params[placeholderName] = doubleVal
			case FilterSQLAllowedFieldsItemBool:
				boolVal, _ := strconv.ParseBool(clause.Value)
				condAnds = append(condAnds, columnName+" IS @"+placeholderName)
				params[placeholderName] = boolVal
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
