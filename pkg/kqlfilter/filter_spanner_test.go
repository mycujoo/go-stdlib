package kqlfilter

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSpannerSQL(t *testing.T) {
	// All of those should return an error.
	testCases := []struct {
		name           string
		input          string
		withRanges     bool
		columnMap      map[string]FilterToSpannerFieldConfig
		expectedError  bool
		expectedSQL    string
		expectedParams map[string]any
	}{
		{
			"one integer field",
			"userId:12345",
			false,
			map[string]FilterToSpannerFieldConfig{
				"userId": {
					ColumnName: "user_id",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
			},
			false,
			"(user_id=@KQL0)",
			map[string]any{
				"KQL0": int64(12345),
			},
		},
		{
			"one integer field and one string field",
			"userId:12345 email:johnexamplecom",
			false,
			map[string]FilterToSpannerFieldConfig{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
				"email": {
					ColumnType: FilterToSpannerFieldColumnTypeString,
				},
			},
			false,
			"(u.user_id=@KQL0 AND email=@KQL1)",
			map[string]any{
				"KQL0": int64(12345),
				"KQL1": "johnexamplecom",
			},
		},
		{
			"one integer field and one string field with no partial matching allowed",
			"userId:12345 email:*examplecom",
			false,
			map[string]FilterToSpannerFieldConfig{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
				"email": {
					ColumnType: FilterToSpannerFieldColumnTypeString,
				},
			},
			false,
			"(u.user_id=@KQL0 AND email=@KQL1)",
			map[string]any{
				"KQL0": int64(12345),
				"KQL1": "*examplecom",
			},
		},
		{
			"one integer field and one string field with prefix matching allowed",
			"userId:12345 email:johnexample*",
			false,
			map[string]FilterToSpannerFieldConfig{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
				"email": {
					ColumnType:       FilterToSpannerFieldColumnTypeString,
					AllowPrefixMatch: true,
				},
			},
			false,
			"(u.user_id=@KQL0 AND email LIKE @KQL1)",
			map[string]any{
				"KQL0": int64(12345),
				"KQL1": "johnexample%",
			},
		},
		{
			"escape percentage sign with wildcard suffix allowed",
			"discount_string:70%*",
			false,
			map[string]FilterToSpannerFieldConfig{
				"discount_string": {
					ColumnType:       FilterToSpannerFieldColumnTypeString,
					AllowPrefixMatch: true,
				},
			},
			false,
			"(discount_string LIKE @KQL0)",
			map[string]any{
				"KQL0": "70\\%%",
			},
		},
		{
			"one integer field and one string field with wildcards allowed, illegal wildcard in middle",
			"userId:12345 email:*example*com",
			false,
			map[string]FilterToSpannerFieldConfig{
				"userId": FilterToSpannerFieldConfig{
					ColumnName: "u.user_id",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
				"email": FilterToSpannerFieldConfig{
					ColumnType:       FilterToSpannerFieldColumnTypeString,
					AllowPrefixMatch: true,
				},
			},
			false,
			"(u.user_id=@KQL0 AND email=@KQL1)",
			map[string]any{
				"KQL0": int64(12345),
				"KQL1": "*example*com",
			},
		},
		{
			"email prefix",
			"email:john@*",
			false,
			map[string]FilterToSpannerFieldConfig{
				"email": FilterToSpannerFieldConfig{
					ColumnType:       FilterToSpannerFieldColumnTypeString,
					AllowPrefixMatch: true,
				},
			},
			false,
			"(email LIKE @KQL0)",
			map[string]any{
				"KQL0": "john@%",
			},
		},
		{
			"email match",
			"email:john@example.com",
			false,
			map[string]FilterToSpannerFieldConfig{
				"email": FilterToSpannerFieldConfig{
					ColumnType:       FilterToSpannerFieldColumnTypeString,
					AllowPrefixMatch: true,
				},
			},
			false,
			"(email=@KQL0)",
			map[string]any{
				"KQL0": "john@example.com",
			},
		},
		{
			"disallowed column",
			"userId:12345 password:qwertyuiop",
			false,
			map[string]FilterToSpannerFieldConfig{
				"userId": {
					ColumnName: "u.user_id",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
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
			map[string]FilterToSpannerFieldConfig{
				"state": {
					MapValue: func(inputValue string) (any, error) {
						switch inputValue {
						case "active":
							return "active", nil
						case "canceled":
							return "canceled", nil
						case "expired":
							return "expired", nil
						}
						return nil, errors.New("illegal value provided")
					},
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
			map[string]FilterToSpannerFieldConfig{
				"state": {
					MapValue: func(inputValue string) (any, error) {
						switch inputValue {
						case "active":
							return "active", nil
						case "canceled":
							return "canceled", nil
						case "expired":
							return "expired", nil
						}
						return nil, errors.New("illegal value provided")
					},
				},
			},
			false,
			"(state=@KQL0)",
			map[string]any{
				"KQL0": "active",
			},
		},
		{
			"allowed field value with input and column values differing",
			"state:payment_state_active",
			false,
			map[string]FilterToSpannerFieldConfig{
				"state": {
					MapValue: func(inputValue string) (any, error) {
						switch inputValue {
						case "payment_state_active":
							return "active", nil
						case "payment_state_canceled":
							return "canceled", nil
						case "payment_state_expired":
							return "expired", nil
						}
						return nil, errors.New("illegal value provided")
					},
				},
			},
			false,
			"(state=@KQL0)",
			map[string]any{
				"KQL0": "active",
			},
		},
		{
			"double columns and bool",
			"lat:52.4052963 lon:4.8856547 exact:false",
			false,
			map[string]FilterToSpannerFieldConfig{
				"lat":   {ColumnType: FilterToSpannerFieldColumnTypeFloat64},
				"lon":   {ColumnType: FilterToSpannerFieldColumnTypeFloat64},
				"exact": {ColumnType: FilterToSpannerFieldColumnTypeBool},
			},
			false,
			"(lat=@KQL0 AND lon=@KQL1 AND exact IS @KQL2)",
			map[string]any{
				"KQL0": 52.4052963,
				"KQL1": 4.8856547,
				"KQL2": false,
			},
		},
		{
			"fuzzy booleans",
			"truthy:1 falsey:0 also_truthy:t",
			false,
			map[string]FilterToSpannerFieldConfig{
				"truthy": {ColumnType: FilterToSpannerFieldColumnTypeBool},
				"falsey": {ColumnType: FilterToSpannerFieldColumnTypeBool},
				"also_truthy": {
					ColumnName: "alsoTruthy",
					ColumnType: FilterToSpannerFieldColumnTypeBool,
				},
			},
			false,
			"(truthy IS @KQL0 AND falsey IS @KQL1 AND alsoTruthy IS @KQL2)",
			map[string]any{
				"KQL0": true,
				"KQL1": false,
				"KQL2": true,
			},
		},
		{
			"all four range operators",
			"userId>=12345 lat<50.0 lon>4.1 date<=\"2023-06-01T23:00:00.20Z\"",
			true,
			map[string]FilterToSpannerFieldConfig{
				"userId": {
					ColumnName: "user_id",
					ColumnType: FilterToSpannerFieldColumnTypeInt64,
				},
				"lat":  {ColumnType: FilterToSpannerFieldColumnTypeFloat64},
				"lon":  {ColumnType: FilterToSpannerFieldColumnTypeFloat64},
				"date": {ColumnType: FilterToSpannerFieldColumnTypeTimestamp},
			},
			false,
			"(user_id>=@KQL0 AND lat<@KQL1 AND lon>@KQL2 AND date<=@KQL3)",
			map[string]any{
				"KQL0": int64(12345),
				"KQL1": 50.0,
				"KQL2": 4.1,
				"KQL3": time.Date(2023, time.June, 1, 23, 0, 0, 200000000, time.UTC),
			},
		},
		{
			"repeat query on same field more than allowed",
			"count>=1 and count<5 and count>3",
			true,
			map[string]FilterToSpannerFieldConfig{
				"count": {},
			},
			true,
			"",
			map[string]any{},
		},
		{
			"in query",
			"state:(state_active OR state_canceled)",
			false,
			map[string]FilterToSpannerFieldConfig{
				"state": {
					AllowMultipleValues: true,
					MapValue: func(inputValue string) (any, error) {
						switch inputValue {
						case "state_active":
							return "active", nil
						case "state_canceled":
							return "canceled", nil
						case "state_expired":
							return "expired", nil
						}
						return nil, errors.New("illegal value provided")
					},
				},
			},
			false,
			"(state IN (?,?))",
			map[string]any{
				"KQL0": "active",
				"KQL1": "canceled",
			},
		},
		{
			"in query - disabled",
			"state:(active OR canceled)",
			false,
			map[string]FilterToSpannerFieldConfig{
				"state": {
					AllowMultipleValues: false,
					MapValue: func(inputValue string) (any, error) {
						switch inputValue {
						case "active":
							return "active", nil
						case "canceled":
							return "canceled", nil
						case "expired":
							return "expired", nil
						}
						return nil, errors.New("illegal value provided")
					},
				},
			},
			true,
			"",
			map[string]any{},
		},
		{
			"in query - int",
			"user_id:(123 OR 321)",
			false,
			map[string]FilterToSpannerFieldConfig{
				"user_id": {
					ColumnName:          "UserID",
					ColumnType:          FilterToSpannerFieldColumnTypeInt64,
					AllowMultipleValues: true,
				},
			},
			false,
			"(UserID IN (?,?))",
			map[string]any{
				"KQL0": int64(123),
				"KQL1": int64(321),
			},
		},
		{
			"in query - bool",
			"user_id:(true OR false)",
			false,
			map[string]FilterToSpannerFieldConfig{
				"user_id": {
					ColumnName:          "UserID",
					ColumnType:          FilterToSpannerFieldColumnTypeBool,
					AllowMultipleValues: true,
				},
			},
			true, // operator IN not supported for field type BOOL
			"",
			map[string]any{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			f, errParse := Parse(test.input, test.withRanges)
			condAnds, params, err := f.ToSpannerSQL(test.columnMap)
			if test.expectedError {
				if errParse == nil && err == nil {
					t.Errorf("expected error, but got none")
				}
				return
			} else {
				require.NoError(t, errParse)
				require.NoError(t, err)
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
