package kqlfilter

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type FilterToSpannerFieldColumnType int

const (
	FilterToSpannerFieldColumnTypeString = iota
	FilterToSpannerFieldColumnTypeInt64
	FilterToSpannerFieldColumnTypeFloat64
	FilterToSpannerFieldColumnTypeBool
	FilterToSpannerFieldColumnTypeTimestamp
)

func (c FilterToSpannerFieldColumnType) String() string {
	switch c {
	case FilterToSpannerFieldColumnTypeString:
		return "STRING"
	case FilterToSpannerFieldColumnTypeInt64:
		return "INT64"
	case FilterToSpannerFieldColumnTypeFloat64:
		return "FLOAT64"
	case FilterToSpannerFieldColumnTypeBool:
		return "BOOL"
	case FilterToSpannerFieldColumnTypeTimestamp:
		return "TIMESTAMP"
	default:
		return "???"
	}
}

type FilterToSpannerFieldConfig struct {
	// SQL table column name. Can be omitted if the column name is equal to the key in the fieldConfigs map.
	ColumnName string
	// SQL column type. Defaults to FilterToSpannerFieldColumnTypeString.
	ColumnType FilterToSpannerFieldColumnType
	// Allow prefix matching when a wildcard (`*`) is present at the end of a string.
	// Only applicable for FilterToSpannerFieldColumnTypeString. Defaults to false.
	AllowPrefixMatch bool
	// Allow multiple values for this field. Defaults to false.
	AllowMultipleValues bool
	// A function that takes a string value as provided by the user and converts it to `any` result that matches how it is
	// stored in the database. This should return an error when the user is providing a value that is illegal for this
	// particular field. Defaults to using the provided value as-is.
	MapValue func(string) (any, error)
}

func (f FilterToSpannerFieldConfig) mapValues(values []string) (any, error) {
	var outputValue any
	var err error
	if f.MapValue != nil {
		outputValue = make([]any, 0, len(values))
		for _, value := range values {
			mappedValue, err := f.MapValue(value)
			if err != nil {
				return nil, err
			}
			outputValue = append(outputValue.([]any), mappedValue)
		}
	} else {
		outputValue = values
	}

	// turn slice of one into a single value
	outputValue = unwrapSlice(outputValue)

	if !f.AllowMultipleValues && reflect.TypeOf(outputValue).Kind() == reflect.Slice {
		return nil, fmt.Errorf("multiple values are not allowed")
	}

	switch ov := outputValue.(type) {
	// convert single string value if needed
	case string:
		outputValue, err = f.convertValue(ov)
		if err != nil {
			return nil, err
		}

	// If output value is a slice of strings, convert each value in the slice if needed
	case []string:
		switch f.ColumnType {
		case FilterToSpannerFieldColumnTypeInt64:
			outSlice := make([]int64, len(ov))
			for i, v := range ov {
				val, err := f.convertValue(v)
				if err != nil {
					return nil, err
				}
				outSlice[i] = val.(int64)
			}
			outputValue = outSlice
		case FilterToSpannerFieldColumnTypeFloat64:
			outSlice := make([]float64, len(ov))
			for i, v := range ov {
				val, err := f.convertValue(v)
				if err != nil {
					return nil, err
				}
				outSlice[i] = val.(float64)
			}
			outputValue = outSlice
		case FilterToSpannerFieldColumnTypeBool:
			outSlice := make([]bool, len(ov))
			for i, v := range ov {
				val, err := f.convertValue(v)
				if err != nil {
					return nil, err
				}
				outSlice[i] = val.(bool)
			}
			outputValue = outSlice
		case FilterToSpannerFieldColumnTypeTimestamp:
			outSlice := make([]time.Time, len(ov))
			for i, v := range ov {
				val, err := f.convertValue(v)
				if err != nil {
					return nil, err
				}
				outSlice[i] = val.(time.Time)
			}
			outputValue = outSlice
		}
	}

	return outputValue, nil
}

func (f FilterToSpannerFieldConfig) convertValue(value string) (any, error) {
	switch f.ColumnType {
	case FilterToSpannerFieldColumnTypeInt64:
		intVal, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid INT64 value: %w", err)
		}
		// convert to int64 since int is not supported by spanner client
		return int64(intVal), nil

	case FilterToSpannerFieldColumnTypeFloat64:

		doubleVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid FLOAT64 value: %w", err)
		}
		return doubleVal, nil

	case FilterToSpannerFieldColumnTypeBool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid BOOL value: %w", err)
		}
		return boolVal, nil

	case FilterToSpannerFieldColumnTypeTimestamp:
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("invalid TIMESTAMP value: %w", err)
		}
		return t, nil
	default:
		return value, nil
	}
}

func unwrapSlice(v any) any {
	if reflect.TypeOf(v).Kind() == reflect.Slice {
		if reflect.ValueOf(v).Len() != 1 {
			return v
		}
		return reflect.ValueOf(v).Index(0).Interface()
	}
	return v
}

// ToSpannerSQL turns a Filter into a partial StandardSQL statement.
// It takes a map of fields that are allowed to be queried via this filter (as a user should not be able to query all
// db columns via a filter). It returns a partial SQL statement that can be added to a WHERE clause, along with
// associated params. An example follows.
//
// Given a Filter that looks like this:
//
//	[(Field: "userId", Operator: "=", Values: []string{"12345"}), (Field: "email", Operator: "=", Values: []string{"john@example.*"})]
//
// and fieldConfigs that looks like this:
//
//	{
//		"userId": (ColumnName: "user_id", ColumnType: FilterToSpannerFieldColumnTypeInt64,  AllowPartialMatch: false),
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
//		"@KQL0": int64(12345),
//		"@KQL1": "john@example.%"
//	}
//
// For multi-value fields, the returned SQL conditions will use the IN operator:
//
//	[(Field: "team_id", Operator: "IN", Values: []string{"T1", "T2"})]
//
//	{
//		"team_id": (ColumnName: "user_id", ColumnType: FilterToSpannerFieldColumnTypeString, AllowMultipleValues: true),
//	}
//
// SQL would be:
//
//	["team_id IN (@KQL0,@KQL1)"]
//
// and params:
//
//	{
//		"@KQL0": "T1",
//		"@KQL1": "T2"
//	}
//
// Note: The Clause Operator is contextually used/ignored. It only works with INT64, FLOAT64 and TIMESTAMP types currently.
func (f Filter) ToSpannerSQL(fieldConfigs map[string]FilterToSpannerFieldConfig) ([]string, map[string]any, error) {
	var condAnds []string
	params := make(map[string]any)

	paramIndex := 0

	for _, clause := range f.Clauses {
		fieldConfig, ok := fieldConfigs[clause.Field]
		if !ok {
			return nil, nil, fmt.Errorf("unknown field: %s", clause.Field)
		}

		columnName := fieldConfig.ColumnName
		if columnName == "" {
			columnName = clause.Field
		}
		mappedValue, err := fieldConfig.mapValues(clause.Values)
		if err != nil {
			return nil, nil, fmt.Errorf("field %s: %w", clause.Field, err)
		}

		operator := clause.Operator

		if len(clause.Values) > 1 && operator != "IN" {
			return nil, nil, fmt.Errorf("operator %s doesn't support multiple values in field: %s", operator, clause.Field)
		}

		switch operator {
		case "IN":
			switch fieldConfig.ColumnType {
			case
				FilterToSpannerFieldColumnTypeString,
				FilterToSpannerFieldColumnTypeInt64,
				FilterToSpannerFieldColumnTypeFloat64,
				FilterToSpannerFieldColumnTypeTimestamp:
				break
			default:
				return nil, nil, fmt.Errorf("operator %s not supported for field type %s", operator, fieldConfig.ColumnType)
			}

			var sb strings.Builder
			sb.WriteString(columnName)
			sb.WriteString(" IN (")

			ss := reflect.ValueOf(mappedValue)

			for i := 0; i < ss.Len(); i++ {
				placeholderName := fmt.Sprintf("%s%d", "KQL", paramIndex)
				params[placeholderName] = ss.Index(i).Interface()

				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString("?")

				paramIndex++
			}
			sb.WriteString(")")

			condAnds = append(condAnds, sb.String())

			continue
		case "=":
			// Prefix match supported only for single string
			mappedString, isString := mappedValue.(string)
			if fieldConfig.AllowPrefixMatch && isString && strings.HasSuffix(mappedString, "*") && !strings.HasSuffix(mappedString, "\\*") {
				operator = " LIKE "
				// escape all instances of % in the string
				mappedString = strings.ReplaceAll(mappedString, "%", "\\%")
				// replace the trailing * with a %
				mappedValue = mappedString[0:len(mappedString)-1] + "%"
				break
			}

			if fieldConfig.ColumnType == FilterToSpannerFieldColumnTypeBool {
				operator = " IS "
				break
			}

		case ">=", "<=", ">", "<":
			switch fieldConfig.ColumnType {
			case FilterToSpannerFieldColumnTypeInt64, FilterToSpannerFieldColumnTypeFloat64, FilterToSpannerFieldColumnTypeTimestamp:
				break
			default:
				return nil, nil, fmt.Errorf("operator %s not supported for field type %s", operator, fieldConfig.ColumnType)
			}
		}

		paramName := fmt.Sprintf("%s%d", "KQL", paramIndex)
		condAnds = append(condAnds, fmt.Sprintf("%s%s@%s", columnName, operator, paramName))
		params[paramName] = mappedValue
		paramIndex++
	}

	return condAnds, params, nil
}
