package kqlfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newItem(typ itemType, text string) item {
	return item{
		typ: typ,
		val: text,
	}
}

var (
	tEOF      = newItem(itemEOF, "")
	tLparen   = newItem(itemLeftParen, "(")
	tRparen   = newItem(itemRightParen, ")")
	tLbrace   = newItem(itemLeftBrace, "{")
	tRbrace   = newItem(itemRightBrace, "}")
	tColon    = newItem(itemColon, ":")
	tWildcard = newItem(itemWildcard, "*")
)

func iterate(l *lexer) []item {
	var items []item
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return items
}

func compareItems(t *testing.T, items []item, expected []item, comparePos bool) {
	assert.Len(t, items, len(expected), "number of items should match")
	for i, item := range items {
		assert.Equalf(t, expected[i].typ, item.typ, "item type should match %s!=%s", expected[i].typ, item.typ)
		assert.Equal(t, expected[i].val, item.val, "item value should match")
		if comparePos {
			assert.Equal(t, expected[i].pos, item.pos, "item position should match")
		}
	}
}

func TestLexer(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []item
	}{
		{
			"empty input",
			"",
			[]item{tEOF},
		},
		{
			"white space",
			" \n\t\r",
			[]item{
				tEOF,
			},
		},
		{
			"simple filter",
			"field: value*",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				newItem(itemIdentifier, "value"),
				tWildcard,
				tEOF,
			},
		},
		{
			"number range",
			"price>125.25",
			[]item{
				newItem(itemIdentifier, "price"),
				newItem(itemRangeOperator, ">"),
				newItem(itemNumber, "125.25"),
				tEOF,
			},
		},
		{
			"number range 2",
			"temp<=-20",
			[]item{
				newItem(itemIdentifier, "temp"),
				newItem(itemRangeOperator, "<="),
				newItem(itemNumber, "-20"),
				tEOF,
			},
		},
		{
			"quoted filter",
			"field: \"value two\"*",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				newItem(itemString, `"value two"`),
				tWildcard,
				tEOF,
			},
		},
		{
			"unicode filter",
			"field: \"Lūgēte, ō Venerēs Cupīdinēsque\"",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				newItem(itemString, `"Lūgēte, ō Venerēs Cupīdinēsque"`),
				tEOF,
			},
		},
		{
			"unicode filter 2",
			"field:Lūgēte",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				newItem(itemIdentifier, `Lūgēte`),
				tEOF,
			},
		},
		{
			"escapes",
			"field\\(x\\):separated\\:value",
			[]item{
				newItem(itemIdentifier, "field(x)"),
				tColon,
				newItem(itemIdentifier, "separated:value"),
				tEOF,
			},
		},
		{
			"parenthesis",
			"field: (one  OR two)",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				tLparen,
				newItem(itemIdentifier, "one"),
				newItem(itemOr, "OR"),
				newItem(itemIdentifier, "two"),
				tRparen,
				tEOF,
			},
		},
		{
			"unclosed parenthesis",
			"field: (",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				tLparen,
				newItem(itemError, "unclosed left parenthesis"),
			},
		},
		{
			"unbalanced parenthesis",
			"field: (one OR two))",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				tLparen,
				newItem(itemIdentifier, "one"),
				newItem(itemOr, "OR"),
				newItem(itemIdentifier, "two"),
				tRparen,
				newItem(itemError, "unexpected right parenthesis"),
			},
		},
		{
			"unbalanced braces",
			"field: {one:x}}",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				tLbrace,
				newItem(itemIdentifier, "one"),
				tColon,
				newItem(itemIdentifier, "x"),
				tRbrace,
				newItem(itemError, "unexpected right brace"),
			},
		},
		{
			"braces",
			"field: {nested:{ field:\"value\"}}",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				tLbrace,
				newItem(itemIdentifier, "nested"),
				tColon,
				tLbrace,
				newItem(itemIdentifier, "field"),
				tColon,
				newItem(itemString, `"value"`),
				tRbrace,
				tRbrace,
				tEOF,
			},
		},
		{
			"dotted",
			"some.nested.field:value",
			[]item{
				newItem(itemIdentifier, "some.nested.field"),
				tColon,
				newItem(itemIdentifier, `value`),
				tEOF,
			},
		},
		{
			"bad number",
			"field: -23d",
			[]item{
				newItem(itemIdentifier, "field"),
				tColon,
				newItem(itemError, `bad number syntax: "-23d"`),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			l := lex(test.input)
			items := iterate(l)
			compareItems(t, items, test.expected, false)
		})
	}
}
