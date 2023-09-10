package kqlfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSQL(t *testing.T) {
	// All of those should return an error.
	testCases := []struct {
		name           string
		input          string
		columnMap      map[string]FilterSQLColumnMapItem
		expectedClause string
		expectedParams map[string]any
	}{
		{
			"one integer field",
			"userId:12345",
			map[string]FilterSQLColumnMapItem{
				"userId": FilterSQLColumnMapItem{
					ColumnName: "user_id",
					ColumnType: FilterSQLColumnMapItemInt,
				},
			},
			"(user_id = @GeneratedPlaceholder0)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
			},
		},
		{
			"one integer field and one string field",
			"userId:12345 email:johnexamplecom",
			map[string]FilterSQLColumnMapItem{
				"userId": FilterSQLColumnMapItem{
					ColumnName: "u.user_id",
					ColumnType: FilterSQLColumnMapItemInt,
				},
				"email": FilterSQLColumnMapItem{
					ColumnType: FilterSQLColumnMapItemString,
				},
			},
			"(u.user_id = @GeneratedPlaceholder0 AND email = @GeneratedPlaceholder1)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": "johnexamplecom",
			},
		},
		{
			"one integer field and one string field with no partial matching allowed",
			"userId:12345 email:*examplecom",
			map[string]FilterSQLColumnMapItem{
				"userId": FilterSQLColumnMapItem{
					ColumnName: "u.user_id",
					ColumnType: FilterSQLColumnMapItemInt,
				},
				"email": FilterSQLColumnMapItem{
					ColumnType: FilterSQLColumnMapItemString,
				},
			},
			"(u.user_id = @GeneratedPlaceholder0 AND email = @GeneratedPlaceholder1)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": "*examplecom",
			},
		},
		{
			"one integer field and one string field with wildcards allowed",
			"userId:12345 email:*examplecom",
			map[string]FilterSQLColumnMapItem{
				"userId": FilterSQLColumnMapItem{
					ColumnName: "u.user_id",
					ColumnType: FilterSQLColumnMapItemInt,
				},
				"email": FilterSQLColumnMapItem{
					ColumnType:        FilterSQLColumnMapItemString,
					AllowPartialMatch: true,
				},
			},
			"(u.user_id = @GeneratedPlaceholder0 AND email LIKE %@GeneratedPlaceholder1)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": "examplecom",
			},
		},
		{
			"one integer field and one string field with wildcards allowed, with suffix",
			"userId:12345 email:*examplecom*",
			map[string]FilterSQLColumnMapItem{
				"userId": FilterSQLColumnMapItem{
					ColumnName: "u.user_id",
					ColumnType: FilterSQLColumnMapItemInt,
				},
				"email": FilterSQLColumnMapItem{
					ColumnType:        FilterSQLColumnMapItemString,
					AllowPartialMatch: true,
				},
			},
			"(u.user_id = @GeneratedPlaceholder0 AND email LIKE %@GeneratedPlaceholder1%)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": "examplecom",
			},
		},
		// Disabled test, parser breaks
		//{
		//	"one integer field and one string field with wildcards allowed, illegal wildcard in middle",
		//	"userId:12345 email:*example*com",
		//	map[string]FilterSQLColumnMapItem{
		//		"userId": FilterSQLColumnMapItem{
		//			ColumnName: "u.user_id",
		//			ColumnType: FilterSQLColumnMapItemInt,
		//		},
		//		"email": FilterSQLColumnMapItem{
		//			ColumnType:        FilterSQLColumnMapItemString,
		//			AllowPartialMatch: true,
		//		},
		//	},
		//	"(u.user_id = @GeneratedPlaceholder0)",
		//	map[string]any{
		//		"GeneratedPlaceholder0": 12345,
		//	},
		//},
		{
			"disallowed column",
			"userId:12345 password:qwertyuiop",
			map[string]FilterSQLColumnMapItem{
				"userId": FilterSQLColumnMapItem{
					ColumnName: "u.user_id",
					ColumnType: FilterSQLColumnMapItemInt,
				},
			},
			"(u.user_id = @GeneratedPlaceholder0)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
			},
		},
		{
			"double columns and bool",
			"lat:52.4052963 lon:4.8856547 exact:false",
			map[string]FilterSQLColumnMapItem{
				"lat":   FilterSQLColumnMapItem{ColumnType: FilterSQLColumnMapItemDouble},
				"lon":   FilterSQLColumnMapItem{ColumnType: FilterSQLColumnMapItemDouble},
				"exact": FilterSQLColumnMapItem{ColumnType: FilterSQLColumnMapItemBool},
			},
			"(lat = @GeneratedPlaceholder0 AND lon = @GeneratedPlaceholder1 AND exact IS @GeneratedPlaceholder2)",
			map[string]any{
				"GeneratedPlaceholder0": 52.4052963,
				"GeneratedPlaceholder1": 4.8856547,
				"GeneratedPlaceholder2": false,
			},
		},
		{
			"fuzzy booleans",
			"truthy:1 falsey:0 also_truthy:t",
			map[string]FilterSQLColumnMapItem{
				"truthy": FilterSQLColumnMapItem{ColumnType: FilterSQLColumnMapItemBool},
				"falsey": FilterSQLColumnMapItem{ColumnType: FilterSQLColumnMapItemBool},
				"also_truthy": FilterSQLColumnMapItem{
					ColumnName: "alsoTruthy",
					ColumnType: FilterSQLColumnMapItemBool,
				},
			},
			"(truthy IS @GeneratedPlaceholder0 AND falsey IS @GeneratedPlaceholder1 AND alsoTruthy IS @GeneratedPlaceholder2)",
			map[string]any{
				"GeneratedPlaceholder0": true,
				"GeneratedPlaceholder1": false,
				"GeneratedPlaceholder2": true,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			f, err := Parse(test.input)
			require.NoError(t, err)
			clause, params := f.toSQL(test.columnMap)
			assert.Equal(t, test.expectedClause, clause)
			assert.Equal(t, test.expectedParams, params)
		})
	}
}
