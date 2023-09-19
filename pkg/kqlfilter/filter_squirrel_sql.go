package kqlfilter

import (
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

type FilterToSquirrelSqlFieldColumnType int

const (
	FilterToSquirrelSqlFieldColumnTypeString = iota
	FilterToSquirrelSqlFieldColumnTypeInt
	FilterToSquirrelSqlFieldColumnTypeFloat
	FilterToSquirrelSqlFieldColumnTypeBool
	FilterToSquirrelSqlFieldColumnTypeTimestamp
)

type FilterToSquirrelSqlFieldConfig struct {
	// SQL table column name. Can be omitted if the column name is equal to the key in the fieldConfigs map.
	ColumnName string
	// SQL column type. Defaults to FilterToSquirrelSqlFieldColumnTypeString.
	ColumnType FilterToSquirrelSqlFieldColumnType
	// Allow prefix matching when a wildcard (`*`) is present at the end of a string.
	// Only applicable for FilterToSpannerFieldColumnTypeString. Defaults to false.
	AllowPrefixMatch bool
	// Allow multiple values for this field. Defaults to false.
	AllowMultipleValues bool
	// A function that takes a string value as provided by the user and converts it to string result that matches how it
	// should be as users' input. This should return an error when the user is providing a value that is illegal or unexpected
	// for this particular field. Defaults to using the provided value as-is.
	MapValue func(string) (string, error)
	// A function that handle parsing the sql statement by itself.
	// If set, all other fields in the config will be ignored
	CustomBuilder func(stmt sq.SelectBuilder, operator string, values []string) (sq.SelectBuilder, error)
}

// ToSquirrelSql parses a Filter and attach the result the given squirrel sql select builder.
//
// It takes a map of fields that are allowed to be queried via this filter (as a user should not be able to query all
// db columns via a filter). It attaches the parsed result to a squirrel select builder as one or more where clauses.
// An example follows.
//
// Given a Filter that looks like this:
//
//	[(Field: "userId", Operator: "=", Values: []string{"12345"}), (Field: "status", Operator: "IN", Values: []string{"active", "frozen", "deleted})]
//
// and fieldConfigs that looks like this:
//
//	{
//		"userId": (ColumnName: "user_id", ColumnType: FilterToSquirrelSqlFieldColumnTypeInt64),
//		"status":  (ColumnName: "status",   ColumnType: FilterToSquirrelSqlFieldColumnTypeString, AllowMultipleValues: true)
//	}
//
// The given squirrel select builder will be attached with condition clauses as below by its Where() method:
//
//		stmt = stmt.Where(sq.Eq{"user_id":123456})
//		stmt = stmt.Where(sq.Eq{"status": []string{"active", "frozen", "deleted}})
//	 return stmt
//
// which will let you get the final sql where conditions like:
//
// ...... WHERE user_id = 123456 AND status in ("active","frozen","deleted") .....
//
// To query with Full Test Search, the column type must be FilterToSquirrelSqlFieldColumnTypeString and the Operator must be "=", and the
// column name should be the corresponding column name in which you store the search tokens
// For example:
//
// Filter:
//
//	[(Field: "title", Operator: "=", Values: []string{"Monday Wednesday Sunday"})]
//
// ConfigMap:
//
//	{ "title": (ColumnName: "title_search_token", ColumnType: FilterToSquirrelSqlFieldColumnTypeString) }
//
// This method will do:
//
//	stmt = stmt.Where(sq.Expr(columnName+" @@ to_tsquery(?)", search))
//	return stmt
//
// And result:
//
//	...... WHERE title_search_token @@ to_tsquery("Monday & Wednesday & Sunday") ......
//
// Note: the input timestamp format should always be time.RFC3339Nano
var unknownFieldErr = errors.Errorf("unknown field")

func (f Filter) ToSquirrelSql(stmt sq.SelectBuilder, fieldConfigs map[string]FilterToSquirrelSqlFieldConfig) (sq.SelectBuilder, error) {
	var err error

	for i, clause := range f.Clauses {
		fieldConfig, ok := fieldConfigs[clause.Field]
		if !ok {
			return stmt, errors.Wrapf(unknownFieldErr, "unknown field: %s", clause.Field)
		}

		stmt, err = clause.ToSquirrelSql(stmt, fieldConfig)
		if err != nil {
			return stmt, errors.Wrapf(err, "failed to parse clause %d to squirrel sql statement", i)
		}
	}
	return stmt, nil
}

var valueConvertErr = errors.Errorf("value convert error") // used in test cases

func (c *Clause) ToSquirrelSql(stmt sq.SelectBuilder, config FilterToSquirrelSqlFieldConfig) (sq.SelectBuilder, error) {
	var err error
	// use customer parser if provided
	if config.CustomBuilder != nil {
		stmt, err = config.CustomBuilder(stmt, c.Operator, c.Values)
		if err != nil {
			return stmt, err
		}
		return stmt, nil
	}

	// get field name
	columnName := config.ColumnName
	if columnName == "" {
		columnName = c.Field
	}

	// use MapValue function in config if provided
	rawValues := c.Values
	if config.MapValue != nil {
		mappedValues := make([]string, 0, len(rawValues))
		for i := range c.Values {
			mappedValue, err := config.MapValue(rawValues[i])
			if err != nil {
				return stmt, err
			}
			mappedValues = append(mappedValues, mappedValue)
		}
		rawValues = mappedValues
	}

	switch config.ColumnType {
	case FilterToSquirrelSqlFieldColumnTypeInt:
		nativeValues := make([]int64, 0, len(rawValues))
		for i, v := range rawValues {
			nativeValue, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return stmt, errors.Wrapf(valueConvertErr, "failed to convert value %s (index %d in filter c values) to int64", v, i)
			}
			nativeValues = append(nativeValues, nativeValue)
		}
		stmt, err = buildStmtByOperator[int64](stmt, columnName, c.Operator, nativeValues, config)
	case FilterToSquirrelSqlFieldColumnTypeFloat:
		nativeValues := make([]float64, 0, len(rawValues))
		for i, v := range rawValues {
			nativeValue, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return stmt, errors.Wrapf(valueConvertErr, "failed to convert value %s (index %d in filter c values) to float64", v, i)
			}
			nativeValues = append(nativeValues, nativeValue)
		}
		stmt, err = buildStmtByOperator[float64](stmt, columnName, c.Operator, nativeValues, config)
	case FilterToSquirrelSqlFieldColumnTypeBool:
		nativeValues := make([]bool, 0, len(rawValues))
		for i, v := range rawValues {
			nativeValue, err := strconv.ParseBool(v)
			if err != nil {
				return stmt, errors.Wrapf(valueConvertErr, "failed to convert value %s (index %d in filter c values) to bool", v, i)
			}
			nativeValues = append(nativeValues, nativeValue)
		}
		stmt, err = buildStmtByOperator[bool](stmt, columnName, c.Operator, nativeValues, config)
	case FilterToSquirrelSqlFieldColumnTypeTimestamp:
		nativeValues := make([]time.Time, 0, len(rawValues))
		for i, v := range rawValues {
			nativeValue, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				return stmt, errors.Wrapf(valueConvertErr, "failed to convert value %s (index %d in filter c values) to time.Time", v, i)
			}
			nativeValues = append(nativeValues, nativeValue)
		}
		stmt, err = buildStmtByOperator[time.Time](stmt, columnName, c.Operator, nativeValues, config)
	default:
		stmt, err = buildStmtByOperator[string](stmt, columnName, c.Operator, rawValues, config)
	}

	if err != nil {
		return stmt, errors.Wrapf(err, "failed to build statement by operator")
	}
	return stmt, nil
}

var emptyValuesErr = errors.Errorf("no values provided")
var valuesNumError = errors.Errorf("wrong values num")
var operatorError = errors.Errorf("unsupported operator")

func buildStmtByOperator[T string | int64 | float64 | bool | time.Time](stmt sq.SelectBuilder, columnName string, op string, values []T, config FilterToSquirrelSqlFieldConfig) (sq.SelectBuilder, error) {
	switch op {
	case "IN":
		if len(values) == 0 {
			return stmt, emptyValuesErr
		}
		if len(values) > 1 && !config.AllowMultipleValues {
			return stmt, errors.Wrapf(valuesNumError, "values num %d doesn't match the operator %s", len(values), op)
		}
		stmt = stmt.Where(sq.Eq{columnName: values})
	case "=", ">", ">=", "<", "<=":
		if len(values) != 1 {
			return stmt, errors.Wrapf(valuesNumError, "values num %d doesn't match the operator %s", len(values), op)
		}
		switch op {
		case "=":
			if vStr, ok := any(values[0]).(string); ok && config.AllowPrefixMatch && strings.HasSuffix(vStr, "*") && !strings.HasSuffix(vStr, `\*`) {
				vStr = vStr[:len(vStr)-1]                  // trim the suffix * ( don't use the TrimRightFunc because it'll also remove the first start from suffix "**"
				vStr = strings.ReplaceAll(vStr, `\`, `\\`) // escape all `\`
				vStr = strings.ReplaceAll(vStr, `%`, `\%`) // escape all `%`
				vStr = strings.ReplaceAll(vStr, `_`, `\_`) // escape all `_`
				stmt = stmt.Where(sq.Like{columnName: vStr + "%"})
			} else {
				stmt = stmt.Where(sq.Eq{columnName: values[0]})
			}
		case ">":
			stmt = stmt.Where(sq.Gt{columnName: values[0]})
		case ">=":
			stmt = stmt.Where(sq.GtOrEq{columnName: values[0]})
		case "<":
			stmt = stmt.Where(sq.Lt{columnName: values[0]})
		case "<=":
			stmt = stmt.Where(sq.LtOrEq{columnName: values[0]})
		}
	default:
		return stmt, errors.Wrapf(operatorError, "unsupported operator %s", op)
	}
	return stmt, nil
}
