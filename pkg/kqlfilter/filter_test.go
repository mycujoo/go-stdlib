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
		expectedError bool
		expected      Filter
	}{
		{
			"one field",
			"field:value",
			false,
			Filter{
				Clauses: []Clause{
					{
						Field: "field",
						Value: "value",
					},
				},
			},
		},
		{
			"two fields",
			"field:value another:second",
			false,
			Filter{
				Clauses: []Clause{
					{
						Field: "field",
						Value: "value",
					},
					{
						Field: "another",
						Value: "second",
					},
				},
			},
		},
		{
			"or is not supported",
			"field:value OR another:second",
			true,
			Filter{},
		},
		{
			"or values not supported",
			"field:(value OR second)",
			true,
			Filter{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			f, err := Parse(test.input)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, f)
		})
	}
}
