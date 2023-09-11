package kqlfilter

import (
	"strings"
	"testing"
	"time"

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
		expectedSQL    string
		expectedParams map[string]any
	}{
		{
			"one integer field",
			"userId:12345",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "user_id",
					ColumnType: FilterSQLAllowedFieldsColumnTypeInt,
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
					ColumnType: FilterSQLAllowedFieldsColumnTypeInt,
				},
				"email": {
					ColumnType: FilterSQLAllowedFieldsColumnTypeString,
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
					ColumnType: FilterSQLAllowedFieldsColumnTypeInt,
				},
				"email": {
					ColumnType: FilterSQLAllowedFieldsColumnTypeString,
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
					ColumnType: FilterSQLAllowedFieldsColumnTypeInt,
				},
				"email": {
					ColumnType:       FilterSQLAllowedFieldsColumnTypeString,
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
		//			ColumnType:       FilterSQLAllowedFieldsColumnTypeString,
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
		//			ColumnType: FilterSQLAllowedFieldsColumnTypeInt,
		//		},
		//		"email": FilterSQLAllowedFieldsItem{
		//			ColumnType:        FilterSQLAllowedFieldsColumnTypeString,
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
					ColumnType: FilterSQLAllowedFieldsColumnTypeInt,
				},
			},
			true,
			"",
			map[string]any{},
		},
		{
			"disallowed field value",
			"state:deleted",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"state": {
					AllowedValues: map[string]string{"active": "active", "canceled": "canceled", "expired": "expired"},
				},
			},
			true,
			"",
			map[string]any{},
		},
		{
			"allowed field value with implicit column value",
			"state:active",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"state": {
					AllowedValues: map[string]string{"active": "", "canceled": "", "expired": ""},
				},
			},
			false,
			"(state=@GeneratedPlaceholder0)",
			map[string]any{
				"GeneratedPlaceholder0": "active",
			},
		},
		{
			"allowed field value with input and column values differing",
			"state:payment_state_active",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"state": {
					AllowedValues: map[string]string{
						"payment_state_active":   "active",
						"payment_state_canceled": "canceled",
						"payment_state_expired":  "expired",
					},
				},
			},
			false,
			"(state=@GeneratedPlaceholder0)",
			map[string]any{
				"GeneratedPlaceholder0": "active",
			},
		},
		{
			"double columns and bool",
			"lat:52.4052963 lon:4.8856547 exact:false",
			false,
			map[string]FilterSQLAllowedFieldsItem{
				"lat":   {ColumnType: FilterSQLAllowedFieldsColumnTypeDouble},
				"lon":   {ColumnType: FilterSQLAllowedFieldsColumnTypeDouble},
				"exact": {ColumnType: FilterSQLAllowedFieldsColumnTypeBool},
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
				"truthy": {ColumnType: FilterSQLAllowedFieldsColumnTypeBool},
				"falsey": {ColumnType: FilterSQLAllowedFieldsColumnTypeBool},
				"also_truthy": {
					ColumnName: "alsoTruthy",
					ColumnType: FilterSQLAllowedFieldsColumnTypeBool,
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
			"userId>=12345 lat<50.0 lon>4.1 date<=\"2023-06-01T23:00:00.20Z\"",
			true,
			map[string]FilterSQLAllowedFieldsItem{
				"userId": {
					ColumnName: "user_id",
					ColumnType: FilterSQLAllowedFieldsColumnTypeInt,
				},
				"lat":  {ColumnType: FilterSQLAllowedFieldsColumnTypeDouble},
				"lon":  {ColumnType: FilterSQLAllowedFieldsColumnTypeDouble},
				"date": {ColumnType: FilterSQLAllowedFieldsColumnTypeDateTime},
			},
			false,
			"(user_id>=@GeneratedPlaceholder0 AND lat<@GeneratedPlaceholder1 AND lon>@GeneratedPlaceholder2 AND date<=@GeneratedPlaceholder3)",
			map[string]any{
				"GeneratedPlaceholder0": 12345,
				"GeneratedPlaceholder1": 50.0,
				"GeneratedPlaceholder2": 4.1,
				"GeneratedPlaceholder3": time.Date(2023, time.June, 1, 23, 0, 0, 200000000, time.UTC),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			f, err := Parse(test.input, test.withRanges)
			condAnds, params, err := f.ToSQL(test.columnMap)
			if test.expectedError {
				require.Error(t, err)
				return
			}

			sql := ""
			if len(condAnds) > 0 {
				sql = "(" + strings.Join(condAnds, " AND ") + ")"
			}
			assert.Equal(t, test.expectedSQL, sql)
			assert.Equal(t, test.expectedParams, params)
		})
	}
}
