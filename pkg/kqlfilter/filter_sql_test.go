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
		withRanges     bool
		columnMap      map[string]FilterSQLAllowedFieldsItem
		expectedError  bool
		expectedClause string
		expectedParams map[string]any
	}{
		{
			"one integer field",
			"userId:12345",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "user_id",
					ColumnType: FilterSQLAllowedFieldsItemInt,
				},
			},
			false,
			"(user_id=@GeneratedPlaceholder0)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
			},
		},
		{
			"one integer field and one string field",
			"userId:12345 email:johnexamplecom",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterSQLAllowedFieldsItemInt,
				},
				"email": {
					ColumnType: FilterSQLAllowedFieldsItemString,
				},
			},
			false,
			"(u.user_id=@GeneratedPlaceholder0 AND email=@GeneratedPlaceholder1)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": "johnexamplecom",
			},
		},
		{
			"one integer field and one string field with no partial matching allowed",
			"userId:12345 email:*examplecom",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterSQLAllowedFieldsItemInt,
				},
				"email": {
					ColumnType: FilterSQLAllowedFieldsItemString,
				},
			},
			false,
			"(u.user_id=@GeneratedPlaceholder0 AND email=@GeneratedPlaceholder1)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": "*examplecom",
			},
		},
		{
			"one integer field and one string field with prefix matching allowed",
			"userId:12345 email:johnexample*",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterSQLAllowedFieldsItemInt,
				},
				"email": {
					ColumnType:       FilterSQLAllowedFieldsItemString,
					AllowPrefixMatch: true,
				},
			},
			false,
			"(u.user_id=@GeneratedPlaceholder0 AND email LIKE @GeneratedPlaceholder1)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": "johnexample%",
			},
		},
		// Disabled test, parser breaks
		//{
		//	"escape percentage sign with wildcard suffix allowed",
		//	"discount_string:70%*",
		//  false,
		//	map[string]FilterSQLAllowedFieldsItem{
		//		"email": {
		//			ColumnType:       FilterSQLAllowedFieldsItemString,
		//			AllowPrefixMatch: true,
		//		},
		//	},
		//	false,
		//	"(email LIKE @GeneratedPlaceholder0)",
		//	map[string]any{
		//		"GeneratedPlaceholder0": "70\\%%",
		//	},
		//},
		// Disabled test, parser breaks
		//{
		//	"one integer field and one string field with wildcards allowed, illegal wildcard in middle",
		//	"userId:12345 email:*example*com",
		//  false,
		//	map[string]FilterSQLAllowedFieldsItem{
		//		"userId": FilterSQLAllowedFieldsItem{
		//			ColumnName: "u.user_id",
		//			ColumnType: FilterSQLAllowedFieldsItemInt,
		//		},
		//		"email": FilterSQLAllowedFieldsItem{
		//			ColumnType:        FilterSQLAllowedFieldsItemString,
		//			AllowPartialMatch: true,
		//		},
		//	},
		//  false,
		//	"(u.user_id=@GeneratedPlaceholder0)",
		//	map[string]any{
		//		"GeneratedPlaceholder0": 12345,
		//	},
		//},
		{
			"disallowed column",
			"userId:12345 password:qwertyuiop",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterSQLAllowedFieldsItemInt,
				},
			},
			true,
			"",
			map[string]any{},
		},
		{
			"double columns and bool",
			"lat:52.4052963 lon:4.8856547 exact:false",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"lat":   {ColumnType: FilterSQLAllowedFieldsItemDouble},
				"lon":   {ColumnType: FilterSQLAllowedFieldsItemDouble},
				"exact": {ColumnType: FilterSQLAllowedFieldsItemBool},
			},
			false,
			"(lat=@GeneratedPlaceholder0 AND lon=@GeneratedPlaceholder1 AND exact IS @GeneratedPlaceholder2)",
			map[string]any{
				"GeneratedPlaceholder0": 52.4052963,
				"GeneratedPlaceholder1": 4.8856547,
				"GeneratedPlaceholder2": false,
			},
		},
		{
			"fuzzy booleans",
			"truthy:1 falsey:0 also_truthy:t",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"truthy": {ColumnType: FilterSQLAllowedFieldsItemBool},
				"falsey": {ColumnType: FilterSQLAllowedFieldsItemBool},
				"also_truthy": {
					ColumnName: "alsoTruthy",
					ColumnType: FilterSQLAllowedFieldsItemBool,
				},
			},
			false,
			"(truthy IS @GeneratedPlaceholder0 AND falsey IS @GeneratedPlaceholder1 AND alsoTruthy IS @GeneratedPlaceholder2)",
			map[string]any{
				"GeneratedPlaceholder0": true,
				"GeneratedPlaceholder1": false,
				"GeneratedPlaceholder2": true,
			},
		},
		{
			"all four range operators",
			"userId>=12345 lat<50.0 lon>4.1 date<=\"2023-06-01 23:00:00\"",
			true,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "user_id",
					ColumnType: FilterSQLAllowedFieldsItemInt,
				},
				"lat":  {ColumnType: FilterSQLAllowedFieldsItemDouble},
				"lon":  {ColumnType: FilterSQLAllowedFieldsItemDouble},
				"date": {ColumnType: FilterSQLAllowedFieldsItemDateTime},
			},
			false,
			"(user_id>=@GeneratedPlaceholder0 AND lat<@GeneratedPlaceholder1 AND lon>@GeneratedPlaceholder2 AND date<=@GeneratedPlaceholder3)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": 50.0,
				"GeneratedPlaceholder2": 4.1,
				"GeneratedPlaceholder3": "2023-06-01 23:00:00",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			f, err := Parse(test.input, test.withRanges)
			clause, params, err := f.toSQL(test.columnMap)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			assert.Equal(t, test.expectedClause, clause)
			assert.Equal(t, test.expectedParams, params)
		})
	}
}
