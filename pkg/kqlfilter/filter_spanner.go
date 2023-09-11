package kqlfilter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FilterToSpannerFieldColumnType int

const (
	FilterToSpannerFieldColumnTypeString = iota
	FilterToSpannerFieldColumnTypeInt
	FilterToSpannerFieldColumnTypeDouble
	FilterToSpannerFieldColumnTypeBool
	FilterToSpannerFieldColumnTypeDateTime
)

type FilterToSpannerFieldConfig struct {
	// SQL table column name. Can be omitted if the column name is equal to the key in the fieldConfigs map.
	ColumnName string
	// SQL column type. Defaults to FilterToSpannerFieldColumnTypeString.
	ColumnType FilterToSpannerFieldColumnType
	// Allow prefix matching when a wildcard (`*`) is present at the end of a string.
	// Only applicable for FilterToSpannerFieldColumnTypeString. Defaults to false.
	AllowPrefixMatch bool
	// A function that takes a value as provided by the user and converts it to an interface{} that matches how it is
	// stored in the database. This should return an error when the user is providing a value that is illegal for this
	// particular field. Defaults to using the provided value as-is.
	// Important: if this provided, no additional type-aware conversions are done to the value (e.g. date strings will
	// no longer be converted to `Time` objects).
	ValueMap func(string) (interface{}, error)
}

// ToSpannerSQL turns a Filter into a partial StandardSQL statement.
// It takes a map of fields that are allowed to be queried via this filter (as a user should not be able to query all
// db columns via a filter). It returns a partial SQL statement that can be added to a WHERE clause, along with
// associated params. An example follows.
//
// Given a Filter that looks like this:
//
//	[(Field: "userId", Operator: "=", Value: "12345"), (Field: "email", Operator: "=", Value: "john@example.*")]
//
// and fieldConfigs that looks like this:
//
//	{
//		"userId": (ColumnName: "user_id", ColumnType: FilterToSpannerFieldColumnTypeInt,    AllowPartialMatch: false),
//		"email":  (ColumnName: "email",   ColumnType: FilterToSpannerFieldColumnTypeString, AllowPartialMatch: true)
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
func (f Filter) ToSpannerSQL(fieldConfigs map[string]FilterToSpannerFieldConfig) ([]string, map[string]interface{}, error) {
	var condAnds []string
	params := map[string]interface{}{}

	for i, clause := range f.Clauses {
		if fieldConfig, ok := fieldConfigs[clause.Field]; ok {
			placeholderName := fmt.Sprintf("%s%d", "GeneratedPlaceholder", i)
			columnName := fieldConfig.ColumnName
			if columnName == "" {
				columnName = clause.Field
			}
			inputValue := clause.Value
			var mappedValue interface{}
			if fieldConfig.ValueMap != nil {
				var err error
				mappedValue, err = fieldConfig.ValueMap(clause.Value)
				if err != nil {
					return []string{}, map[string]interface{}{}, err
				}
			}
			switch fieldConfig.ColumnType {
			case FilterToSpannerFieldColumnTypeString:
				if fieldConfig.AllowPrefixMatch {
					useValue := inputValue
					if mappedValue_, ok := mappedValue.(string); ok {
						useValue = mappedValue_
					}
					if fieldConfig.AllowPrefixMatch && strings.HasSuffix(useValue, "*") {
						// TODO: Handle escaped asterisk (*) characters that should not serve as wildcards
						condAnds = append(condAnds, fmt.Sprintf("%s LIKE @%s", columnName, placeholderName))
						escapedValue := strings.ReplaceAll(useValue, "%", "\\%")
						params[placeholderName] = escapedValue[0:len(escapedValue)-1] + "%"
					} else {
						condAnds = append(condAnds, fmt.Sprintf("%s=@%s", columnName, placeholderName))
						params[placeholderName] = useValue
					}
				} else {
					condAnds = append(condAnds, fmt.Sprintf("%s=@%s", columnName, placeholderName))
					if mappedValue != nil {
						params[placeholderName] = mappedValue
					} else {
						params[placeholderName] = inputValue
					}
				}
			case FilterToSpannerFieldColumnTypeInt:
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				if mappedValue != nil {
					params[placeholderName] = mappedValue
				} else {
					intVal, err := strconv.Atoi(inputValue)
					if err != nil {
						return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
					}
					params[placeholderName] = intVal
				}
			case FilterToSpannerFieldColumnTypeDouble:
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				if mappedValue != nil {
					params[placeholderName] = mappedValue
				} else {
					doubleVal, err := strconv.ParseFloat(inputValue, 64)
					if err != nil {
						return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
					}
					params[placeholderName] = doubleVal
				}
			case FilterToSpannerFieldColumnTypeBool:
				condAnds = append(condAnds, fmt.Sprintf("%s IS @%s", columnName, placeholderName))
				if mappedValue != nil {
					params[placeholderName] = mappedValue
				} else {
					boolVal, _ := strconv.ParseBool(inputValue)
					params[placeholderName] = boolVal
				}
			case FilterToSpannerFieldColumnTypeDateTime:
				condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, clause.Operator, placeholderName))
				if mappedValue != nil {
					params[placeholderName] = mappedValue
				} else {
					t, err := time.Parse(time.RFC3339, inputValue)
					if err != nil {
						return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
					}
					params[placeholderName] = t
				}
			}
		} else {
			return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
		}
	}

	return condAnds, params, nil
}
