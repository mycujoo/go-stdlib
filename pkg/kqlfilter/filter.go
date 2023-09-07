package kqlfilter

import "fmt"

type SimpleFilter struct {
	Clauses []SimpleClause
}

type SimpleClause struct {
	Field string
	Value string
}

// ParseSimpleFilter parses a simple filter string into a SimpleFilter struct.
// The filter string must not contain any boolean operators, parentheses or nested queries.
// The filter string must contain only simple clauses of the form "field:value".
func ParseSimpleFilter(input string) (SimpleFilter, error) {
	ast, err := Parse(input, DisableComplexExpressions())
	if err != nil {
		return SimpleFilter{}, err
	}
	return convertToSimpleFilter(ast)
}

// Parse parses a filter string into an AST.
// The filter string must be a valid Kibana query language filter string.
func Parse(input string, options ...ParserOption) (n Node, err error) {
	p := &parser{}
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

func convertToSimpleFilter(ast Node) (SimpleFilter, error) {
	if ast == nil {
		return SimpleFilter{}, nil
	}
	switch n := ast.(type) {
	case *AndNode:
		return convertAndNode(n)
	case *IsNode:
		return convertIsNode(n)
	default:
		return SimpleFilter{}, fmt.Errorf("unsupported node type %T", ast)
	}
}

func convertAndNode(ast *AndNode) (SimpleFilter, error) {
	var filter SimpleFilter
	for _, node := range ast.Nodes {
		isNode, ok := node.(*IsNode)
		if !ok {
			return SimpleFilter{}, fmt.Errorf("unsupported node type %T", node)
		}
		f, err := convertIsNode(isNode)
		if err != nil {
			return SimpleFilter{}, err
		}
		filter.Clauses = append(filter.Clauses, f.Clauses...)
	}
	return filter, nil
}

func convertIsNode(ast *IsNode) (SimpleFilter, error) {
	var value string
	switch n := ast.Value.(type) {
	case *LiteralNode:
		value = n.Value
	default:
		return SimpleFilter{}, fmt.Errorf("unsupported node type %T", ast.Value)
	}
	return SimpleFilter{
		Clauses: []SimpleClause{
			{
				Field: ast.Identifier,
				Value: value,
			},
		},
	}, nil
}
