package kqlfilter

import (
	"fmt"
	"strings"
)

type Filter struct {
	Clauses []Clause
}

type Clause struct {
	Field string
	// One of the following: `=`, `<`, `<=`, `>`, `>=`
	Operator string
	Value    string
}

// Parse parses a filter string into a Filter struct.
// The filter string must not contain any boolean operators, parentheses or nested queries.
// The filter string must contain only simple clauses of the form "field:value", where all clauses are AND'ed.
// Optionally, range operators can be enabled, e.g. for expressions involving date ranges.
// If you need to parse a more complex filter string, use ParseAST instead.
func Parse(input string, enableRangeOperator bool) (Filter, error) {
	if strings.TrimSpace(input) == "" {
		return Filter{}, nil
	}
	ast, err := ParseAST(input, DisableComplexExpressions())
	if err != nil {
		return Filter{}, err
	}
	return convertToFilter(ast, enableRangeOperator)
}

// ParseAST parses a filter string into an AST.
// The filter string must be a valid Kibana query language filter string.
func ParseAST(input string, options ...ParserOption) (n Node, err error) {
	p := &parser{
		maxDepth:      20,
		maxComplexity: 20,
	}
	for _, option := range options {
		option(p)
	}
	p.text = input

	defer p.recover(&err)
	p.lex = lex(input)
	p.parse()
	p.lex = nil // release lexer for garbage collection

	return p.Root, err
}

// ParserOption is a function that configures a parser.
type ParserOption func(*parser)

// DisableComplexExpressions disables complex expressions.
func DisableComplexExpressions() ParserOption {
	return func(p *parser) {
		p.disableComplexExpressions = true
	}
}

// WithMaxDepth sets limit to maximum number of nesting.
func WithMaxDepth(depth int) ParserOption {
	return func(p *parser) {
		p.maxDepth = depth
	}
}

// WithMaxComplexity sets limit to maximum number of individual clauses separated by boolean operators.
func WithMaxComplexity(complexity int) ParserOption {
	return func(p *parser) {
		p.maxComplexity = complexity
	}
}

func convertToFilter(ast Node, enableRangeOperator bool) (Filter, error) {
	if ast == nil {
		return Filter{}, nil
	}
	switch n := ast.(type) {
	case *AndNode:
		return convertAndNode(n, enableRangeOperator)
	case *IsNode:
		return convertIsNode(n)
	case *RangeNode:
		if enableRangeOperator {
			return convertRangeNode(n)
		}
		return Filter{}, fmt.Errorf("unsupported node type %T", ast)
	default:
		return Filter{}, fmt.Errorf("unsupported node type %T", ast)
	}
}

func convertAndNode(ast *AndNode, enableRangeOperator bool) (Filter, error) {
	var filter Filter
	fieldCounts := make(map[string]int)
	for _, node := range ast.Nodes {
		var f Filter
		var err error
		switch n := node.(type) {
		case *IsNode:
			f, err = convertIsNode(n)
		case *RangeNode:
			if !enableRangeOperator {
				return Filter{}, fmt.Errorf("unsupported node type %T", ast)
			}
			f, err = convertRangeNode(n)
		default:
			return Filter{}, fmt.Errorf("unsupported node type %T", ast)
		}
		if err != nil {
			return Filter{}, err
		}
		filter.Clauses = append(filter.Clauses, f.Clauses...)
	}
	for _, clause := range filter.Clauses {
		fieldCounts[clause.Field]++
		if fieldCounts[clause.Field] > 2 {
			return Filter{}, fmt.Errorf("field count maximum in filter exceeded")
		}
	}
	return filter, nil
}

func convertIsNode(ast *IsNode) (Filter, error) {
	var value string
	switch n := ast.Value.(type) {
	case *LiteralNode:
		value = n.Value
	default:
		return Filter{}, fmt.Errorf("unsupported node type %T", ast.Value)
	}
	return Filter{
		Clauses: []Clause{
			{
				Field:    ast.Identifier,
				Operator: "=",
				Value:    value,
			},
		},
	}, nil
}

func convertRangeNode(ast *RangeNode) (Filter, error) {
	var value string
	switch n := ast.Value.(type) {
	case *LiteralNode:
		value = n.Value
	default:
		return Filter{}, fmt.Errorf("unsupported node type %T", ast.Value)
	}
	operator := ast.Operator.String()
	if operator == "???" {
		return Filter{}, fmt.Errorf("unsupported operator %s", operator)
	}
	return Filter{
		Clauses: []Clause{
			{
				Field:    ast.Identifier,
				Operator: operator,
				Value:    value,
			},
		},
	}, nil
}
