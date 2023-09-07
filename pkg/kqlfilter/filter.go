package kqlfilter

import "fmt"

type Filter struct {
	Clauses []Clause
}

type Clause struct {
	Field string
	Value string
}

// Parse parses a filter string into a Filter struct.
// The filter string must not contain any boolean operators, parentheses or nested queries.
// The filter string must contain only simple clauses of the form "field:value".
// If you need to parse a more complex filter string, use ParseAST instead.
func Parse(input string) (Filter, error) {
	ast, err := ParseAST(input, DisableComplexExpressions())
	if err != nil {
		return Filter{}, err
	}
	return convertToSimpleFilter(ast)
}

// ParseAST parses a filter string into an AST.
// The filter string must be a valid Kibana query language filter string.
func ParseAST(input string, options ...ParserOption) (n Node, err error) {
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

func convertToSimpleFilter(ast Node) (Filter, error) {
	if ast == nil {
		return Filter{}, nil
	}
	switch n := ast.(type) {
	case *AndNode:
		return convertAndNode(n)
	case *IsNode:
		return convertIsNode(n)
	default:
		return Filter{}, fmt.Errorf("unsupported node type %T", ast)
	}
}

func convertAndNode(ast *AndNode) (Filter, error) {
	var filter Filter
	for _, node := range ast.Nodes {
		isNode, ok := node.(*IsNode)
		if !ok {
			return Filter{}, fmt.Errorf("unsupported node type %T", node)
		}
		f, err := convertIsNode(isNode)
		if err != nil {
			return Filter{}, err
		}
		filter.Clauses = append(filter.Clauses, f.Clauses...)
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
				Field: ast.Identifier,
				Value: value,
			},
		},
	}, nil
}
