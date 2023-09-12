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
//	["user_id=@KQL0", "email LIKE @KQL1"]
//
// and params:
//
//	{
//		"@KQL0": 12345,
//		"@KQL1": "john@example.%"
//	}
//
// Note: The Clause Operator is contextually used/ignored. It only works with int, double and datetime types currently.
func (f Filter) ToSpannerSQL(fieldConfigs map[string]FilterToSpannerFieldConfig) ([]string, map[string]interface{}, error) {
	var condAnds []string
	params := map[string]interface{}{}

	for i, clause := range f.Clauses {
		fieldConfig, ok := fieldConfigs[clause.Field]
		if !ok {
			return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
		}

		placeholderName := fmt.Sprintf("%s%d", "KQL", i)
		columnName := fieldConfig.ColumnName
		if columnName == "" {
			columnName = clause.Field
		}
		var mappedValue interface{}
		mappedValue = clause.Value

		var operator string
		operator = clause.Operator

		if fieldConfig.ValueMap != nil {
			var err error
			mappedValue, err = fieldConfig.ValueMap(clause.Value)
			if err != nil {
				return []string{}, map[string]interface{}{}, err
			}
		}
		mappedString, isString := mappedValue.(string)
		switch fieldConfig.ColumnType {
		case FilterToSpannerFieldColumnTypeString:
			if fieldConfig.AllowPrefixMatch && isString && strings.HasSuffix(mappedString, "*") && !strings.HasSuffix(mappedString, "\\*") {
				operator = " LIKE "
				// escape all instances of % in the string
				mappedString = strings.ReplaceAll(mappedString, "%", "\\%")
				// replace the trailing * with a %
				mappedValue = mappedString[0:len(mappedString)-1] + "%"
				break
			}

			operator = "="

		case FilterToSpannerFieldColumnTypeInt:
			// if mappedValue is a string - convert it to int64
			if isString {
				intVal, err := strconv.Atoi(mappedString)
				if err != nil {
					return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter value found in field: %s", clause.Field)
				}
				// convert to int64 since int is not supported by spanner client
				mappedValue = int64(intVal)
			}

		case FilterToSpannerFieldColumnTypeDouble:
			// if mappedValue is a string - convert it to float64
			if isString {
				doubleVal, err := strconv.ParseFloat(mappedString, 64)
				if err != nil {
					return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter value found in field: %s", clause.Field)
				}
				mappedValue = doubleVal
			}

		case FilterToSpannerFieldColumnTypeBool:
			operator = " IS "
			// if mappedValue is a string - convert it to bool
			if isString {
				boolVal, _ := strconv.ParseBool(mappedString)
				mappedValue = boolVal
			}

		case FilterToSpannerFieldColumnTypeDateTime:
			// if mappedValue is a string - convert it to time.Time
			if isString {
				t, err := time.Parse(time.RFC3339, mappedString)
				if err != nil {
					return []string{}, map[string]interface{}{}, fmt.Errorf("disallowed filter found in field: %s", clause.Field)
				}
				mappedValue = t
			}
		}

		condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, operator, placeholderName))
		params[placeholderName] = mappedValue
	}

	return condAnds, params, nil
}
