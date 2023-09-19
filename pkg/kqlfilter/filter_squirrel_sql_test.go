package kqlfilter

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"
)

func TestToSquirrelSql(t *testing.T) {
	// All of those should return an error.
	testCases := []struct {
		name          string
		input         string
		withRanges    bool
		columnMap     map[string]FilterToSquirrelSqlFieldConfig
		expectedError error
		expectedSQL   string
		expectedArgs  []any
	}{
		{
			"one string field",
			"name:Beau",
			false,
			map[string]FilterToSquirrelSqlFieldConfig{
				"name": {
					ColumnName: "name",
					ColumnType: FilterToSpannerFieldColumnTypeString,
				},
			},
			nil,
			"SELECT * FROM users WHERE name = ?",
			[]any{"Beau"},
		},
		{
			"one integer field",
			"age:30",
			false,
			map[string]FilterToSquirrelSqlFieldConfig{
				"age": {
					ColumnName: "age",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
			},
			nil,
			"SELECT * FROM users WHERE age = ?",
			[]any{int64(30)},
		},
		{
			"one boolean field",
			"local:false",
			false,
			map[string]FilterToSquirrelSqlFieldConfig{
				"local": {
					ColumnName: "local",
					ColumnType: FilterToSpannerFieldColumnTypeBool,
				},
			},
			nil,
			"SELECT * FROM users WHERE local = ?",
			[]any{false},
		},
		{
			"one float field",
			"weight:70.7",
			false,
			map[string]FilterToSquirrelSqlFieldConfig{
				"weight": {
					ColumnName: "weight",
					ColumnType: FilterToSpannerFieldColumnTypeFloat64,
				},
			},
			nil,
			"SELECT * FROM users WHERE weight = ?",
			[]any{70.7},
		},
		{
			"one timestamp field",
			"birthdate>\"1993-11-26T07:00:00Z\"",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"birthdate": {
					ColumnName: "birthdate",
					ColumnType: FilterToSpannerFieldColumnTypeTimestamp,
				},
			},
			nil,
			"SELECT * FROM users WHERE birthdate > ?",
			[]any{time.Date(1993, 11, 26, 7, 0, 0, 0, time.UTC)},
		},
		{
			"all type of values together",
			"name:Beau age:30 weight:70.7 local:false favorite_day: (Monday OR Tuesday)",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"name": {
					ColumnName: "name",
					ColumnType: FilterToSpannerFieldColumnTypeString,
				},
				"age": {
					ColumnName: "age",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
				"local": {
					ColumnName: "local",
					ColumnType: FilterToSpannerFieldColumnTypeBool,
				},
				"weight": {
					ColumnName: "weight",
					ColumnType: FilterToSpannerFieldColumnTypeFloat64,
				},
				"favorite_day": {
					ColumnName:          "favorite_day",
					ColumnType:          FilterToSpannerFieldColumnTypeString,
					AllowMultipleValues: true,
				},
			},
			nil,
			"SELECT * FROM users WHERE name = ? AND age = ? AND weight = ? AND local = ? AND favorite_day IN (?,?)",
			[]any{"Beau", int64(30), 70.7, false, "Monday", "Tuesday"},
		},
		{
			"one string field with IN operator",
			"favorite_day: (Monday OR Tuesday)",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"favorite_day": {
					ColumnName:          "favorite_day",
					ColumnType:          FilterToSpannerFieldColumnTypeString,
					AllowMultipleValues: true,
				},
			},
			nil,
			"SELECT * FROM users WHERE favorite_day IN (?,?)",
			[]any{"Monday", "Tuesday"},
		},
		{
			"one string field with prefix matching",
			`self_intro:"Monday_%a\\_\\%\\**"`,
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"self_intro": {
					ColumnName:       "self_intro",
					ColumnType:       FilterToSpannerFieldColumnTypeString,
					AllowPrefixMatch: true,
				},
			},
			nil,
			"SELECT * FROM users WHERE self_intro LIKE ?",
			[]any{`Monday\_\%a\\\_\\\%\\*%`},
		},
		{
			"one string field with values map",
			"favorite_day:(Monday OR Tuesday)",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"favorite_day": {
					ColumnName:          "favorite_day",
					ColumnType:          FilterToSpannerFieldColumnTypeString,
					AllowMultipleValues: true,
					MapValue: func(s string) (string, error) {
						switch s {
						case "Monday":
							return "monday", nil
						case "Tuesday":
							return "tuesday", nil
						default:
							return "", fmt.Errorf("wrong day")
						}
					},
				},
			},
			nil,
			"SELECT * FROM users WHERE favorite_day IN (?,?)",
			[]any{"monday", "tuesday"},
		},
		{
			"unknown field",
			"name:Beau age:30",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"age": {
					ColumnName: "age",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
			},
			unknownFieldErr,
			"",
			nil,
		},
		{
			"wrong value type",
			"age:Beau",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"age": {
					ColumnName: "age",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
			},
			valueConvertErr,
			"",
			nil,
		},
		{
			"wrong values number",
			"age: (1 OR 2)",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"age": {
					ColumnName: "age",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
			},
			valuesNumError,
			"",
			nil,
		},
		{
			"custom parser",
			"age: (1 OR 2)",
			true,
			map[string]FilterToSquirrelSqlFieldConfig{
				"age": {
					ColumnName: "age",
					CustomBuilder: func(stmt sq.SelectBuilder, operator string, values []string) (sq.SelectBuilder, error) {
						for i := range values {
							vInt64, err := strconv.ParseInt(values[i], 10, 64)
							if err != nil {
								return stmt, err
							}
							stmt = stmt.Where(sq.Gt{"age": vInt64})
						}
						return stmt, nil
					},
				},
			},
			nil,
			"SELECT * FROM users WHERE age > ? AND age > ?",
			[]any{int64(1), int64(2)},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			f, errParse := Parse(test.input, test.withRanges)
			require.NoError(t, errParse)
			stmt, err := f.ToSquirrelSql(sq.Select("*").From("users"), test.columnMap)
			require.ErrorIs(t, err, test.expectedError)
			if test.expectedError == nil {
				sql, args, err := stmt.ToSql()
				require.NoError(t, err)
				require.Equal(t, test.expectedSQL, sql)
				require.Equal(t, test.expectedArgs, args)
			}
		})
	}
}
