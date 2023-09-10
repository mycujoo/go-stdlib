package kqlfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAST(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedError bool
		expected      string
	}{
		{
			"simple filter",
			"field:value",
			false,
			"field=value",
		},
		{
			"quoted",
			`field:"value"`,
			false,
			"field=value",
		},
		{
			"quoted 2",
			`field:"value AND x"`,
			false,
			"field=value AND x",
		},
		{
			"wildcard",
			"field:value*",
			false,
			"field=value*",
		},
		{
			"wildcard prefix",
			"field:*value",
			false,
			"field=*value",
		},
		{
			"multi filter",
			"field:value second:filter",
			false,
			"(field=value AND second=filter)",
		},
		{
			"boolean OR",
			"field:value OR second: filter",
			false,
			"(field=value OR second=filter)",
		},
		{
			"boolean mixed",
			"first:x OR second:y AND third:z",
			false,
			"(first=x OR (second=y AND third=z))",
		},
		{
			"boolean mixed 2",
			"first:x OR second:y and NOT third:z",
			false,
			"(first=x OR (second=y AND NOT third=z))",
		},
		{
			"boolean mixed 3",
			"(first:x OR second:y) and NOT third:z",
			false,
			"((first=x OR second=y) AND NOT third=z)",
		},
		{
			"or values",
			"field:(x OR y AND z)",
			false,
			"field=(x OR (y AND z))",
		},
		{
			"nested values",
			"field:{nested:x or y:z}",
			false,
			"field={(nested=x OR y=z)}",
		},
		{
			"ranges",
			`start_time >= "2022-02-02T10:30:00.000Z" start_time < "2022-02-03T10:30:00.000Z"`,
			false,
			"(start_time>=2022-02-02T10:30:00.000Z AND start_time<2022-02-03T10:30:00.000Z)",
		},
		{
			"ranges2",
			`start_time>"2022-02-02T10:30:00.000Z" OR start_time<="2022-02-03T10:30:00.000Z"`,
			false,
			"(start_time>2022-02-02T10:30:00.000Z OR start_time<=2022-02-03T10:30:00.000Z)",
		},
		{
			"escapes",
			"field\\(x\\):separated\\:value",
			false,
			"field(x)=separated:value",
		},
		{
			"escapes 2",
			"field:slashed\\\\value",
			false,
			"field=slashed\\value",
		},
		{
			"escapes 3",
			"field:\\and",
			false,
			"field=and",
		},
		{
			"invalid wildcard",
			"value*",
			true,
			"",
		},
		{
			"invalid parenthesis",
			"field:(x OR y",
			true,
			"",
		},
		{
			"invalid syntax",
			"field>(x OR y)",
			true,
			"",
		},
		{
			"invalid syntax 2",
			"field :: value",
			true,
			"",
		},
		{
			"invalid syntax 3",
			"field < :value",
			true,
			"",
		},
		{
			"invalid syntax 4",
			"some:field AND OR another:field",
			true,
			"",
		},
		{
			"invalid syntax 5",
			"some:field,another:field",
			true,
			"",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			n, err := ParseAST(test.input)
			if test.expectedError {
				require.Error(t, err, "expected error, got none")
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, n.String())
			}
		})
	}
}

func TestParseSimple(t *testing.T) {
	// All of those should return an error.
	testCases := []struct {
		name  string
		input string
	}{
		{
			"boolean 2",
			"field:value OR second:filter",
		},
		{
			"list of values",
			"field:(x OR y OR z)",
		},
		{
			"nested",
			"field:{nested:x}",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseAST(test.input, DisableComplexExpressions())
			require.Error(t, err)
		})
	}

}