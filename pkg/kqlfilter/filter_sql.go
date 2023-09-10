package kqlfilter

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	FilterSQLColumnMapItemString = iota
	FilterSQLColumnMapItemInt
	FilterSQLColumnMapItemDouble
	FilterSQLColumnMapItemBool
)

type FilterSQLColumnMapItem struct {
	// SQL table column name. Can be omitted if the column name is equal to the key in the column map.
	ColumnName string
	// SQL column type
	ColumnType int
	// Allow prefix and/or suffix matching when a wildcard (`*`) is present. Only works for FilterColumnMapValueString
	AllowPartialMatch bool
}

// toSQL turns a Filter into a partial SQL statement. It takes a map of columns that are allowed to be queried via this
// filter. It returns a SQL clause that can be added to a WHERE clause, along with associated params.
// An example follows.
//
// Given a Filter that looks like this:
//
//	[(Field: "userId", Value: "12345"), (Field: "email", Value: "*@example.com")]
//
// and a columnMap that looks like this:
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
func (f Filter) toSQL(columnMap map[string]FilterSQLColumnMapItem) (string, map[string]any) {
	condAnds := []string{}
	params := map[string]any{}

	for i, clause := range f.Clauses {
		if cmv, ok := columnMap[clause.Field]; ok {
			columnName := cmv.ColumnName
			if columnName == "" {
				columnName = clause.Field
			}
			placeholderName := fmt.Sprintf("%s%d", "GeneratedPlaceholder", i)
			switch cmv.ColumnType {
			case FilterSQLColumnMapItemString:
				if !cmv.AllowPartialMatch {
					condAnds = append(condAnds, columnName+" = @"+placeholderName)
					params[placeholderName] = clause.Value
				} else {
					// TODO: Handle for escaped asterisk (*) characters that should not serve as wildcards
					prefixMatch := strings.HasPrefix(clause.Value, "*")
					suffixMatch := strings.HasSuffix(clause.Value, "*")
					if prefixMatch {
						if suffixMatch {
							condAnds = append(condAnds, columnName+" LIKE %@"+placeholderName+"%")
							params[placeholderName] = clause.Value[1 : len(clause.Value)-1]
						} else {
							condAnds = append(condAnds, columnName+" LIKE %@"+placeholderName)
							params[placeholderName] = clause.Value[1:len(clause.Value)]
						}
					} else if suffixMatch {
						condAnds = append(condAnds, columnName+" LIKE @"+placeholderName+"%")
						params[placeholderName] = clause.Value[0 : len(clause.Value)-1]
					} else {
						condAnds = append(condAnds, columnName+" = @"+placeholderName)
						params[placeholderName] = clause.Value
					}
				}
			case FilterSQLColumnMapItemInt:
				intVal, err := strconv.Atoi(clause.Value)
				if err != nil {
					continue
				}
				condAnds = append(condAnds, columnName+" = @"+placeholderName)
				params[placeholderName] = intVal
			case FilterSQLColumnMapItemDouble:
				doubleVal, err := strconv.ParseFloat(clause.Value, 64)
				if err != nil {
					continue
				}
				condAnds = append(condAnds, columnName+" = @"+placeholderName)
				params[placeholderName] = doubleVal
			case FilterSQLColumnMapItemBool:
				boolVal, _ := strconv.ParseBool(clause.Value)
				condAnds = append(condAnds, columnName+" IS @"+placeholderName)
				params[placeholderName] = boolVal
			}
		}
	}

	if len(condAnds) == 0 {
		return "", params
	}
	sql := "(" + strings.Join(condAnds, " AND ") + ")"
	return sql, params
}
