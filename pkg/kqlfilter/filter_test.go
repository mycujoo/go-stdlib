package kqlfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	// All of those should return an error.
	testCases := []struct {
		name          string
		input         string
		withRanges    bool
		expectedError bool
		expected      Filter
	}{
		{
			"empty",
			"  ",
			false,
			false,
			Filter{},
		},
		{
			"one field",
			"field:value",
			false,
			false,
			Filter{
				Clauses: []Clause{
					{
						Field:    "field",
						Operator: "=",
						Value:    "value",
					},
				},
			},
		},
		{
			"two fields",
			"field:value another:second",
			false,
			false,
			Filter{
				Clauses: []Clause{
					{
						Field:    "field",
						Operator: "=",
						Value:    "value",
					},
					{
						Field:    "another",
						Operator: "=",
						Value:    "second",
					},
				},
			},
		},
		{
			"two fields with and",
			"field:value and another:second",
			false,
			false,
			Filter{
				Clauses: []Clause{
					{
						Field:    "field",
						Operator: "=",
						Value:    "value",
					},
					{
						Field:    "another",
						Operator: "=",
						Value:    "second",
					},
				},
			},
		},
		{
			"or is not supported",
			"field:value OR another:second",
			false,
			true,
			Filter{},
		},
		{
			"or values not supported",
			"field:(value OR second)",
			false,
			true,
			Filter{},
		},
		{
			"one field with range operator",
			"field>=value",
			true,
			false,
			Filter{
				Clauses: []Clause{
					{
						Field:    "field",
						Operator: ">=",
						Value:    "value",
					},
				},
			},
		},
		{
			"one field with illegal range operator",
			"field>=value",
			false,
			true,
			Filter{},
		},
		{
			"one field repeated to create a range",
			"amount>=1 and amount<5",
			true,
			false,
			Filter{
				Clauses: []Clause{
					{
						Field:    "amount",
						Operator: ">=",
						Value:    "1",
					},
					{
						Field:    "amount",
						Operator: "<",
						Value:    "5",
					},
				},
			},
		},
		{
			"3 or more and in a sequence",
			"a:1 and b:2 and c:3 and d:4 and e:6",
			false,
			false,
			Filter{
				Clauses: []Clause{
					{
						Field:    "a",
						Operator: "=",
						Value:    "1",
					},
					{
						Field:    "b",
						Operator: "=",
						Value:    "2",
					},
					{
						Field:    "c",
						Operator: "=",
						Value:    "3",
					},
					{
						Field:    "d",
						Operator: "=",
						Value:    "4",
					},
					{
						Field:    "e",
						Operator: "=",
						Value:    "6",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			f, err := Parse(test.input, test.withRanges)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, f)
		})
	}
}
