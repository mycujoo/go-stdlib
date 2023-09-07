package kqlfilter

import (
	"fmt"
	"runtime"
)

// parser is the representation of a single parsed filter.
type parser struct {
	Root Node   // top-level root of the tree.
	text string // text parsed to create the filter
	// Parsing only; cleared after parse.
	lex       *lexer
	token     [3]item // three-token lookahead for parser.
	peekCount int
	// Disallow complex expressions:
	// OR, AND, NOT, grouping parentheses or nested queries.
	disableComplexExpressions bool
}

// next returns the next token.
func (p *parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lex.nextItem()
	}
	return p.token[p.peekCount]
}

// backup backs the input stream up one token.
func (p *parser) backup() {
	p.peekCount++
}

// peek returns but does not consume the next token.
func (p *parser) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1]
	}
	p.peekCount = 1
	p.token[0] = p.lex.nextItem()
	return p.token[0]
}

// Parsing.

// errorf formats the error and terminates processing.
func (p *parser) errorf(format string, args ...any) {
	p.Root = nil
	format = fmt.Sprintf("parser error: %s at pos %d", format, p.token[0].pos)
	panic(fmt.Errorf(format, args...))
}

// expect consumes the next token and guarantees it has the required type.
func (p *parser) expect(expected itemType, context string) item {
	token := p.next()
	if token.typ != expected {
		p.unexpected(token, context)
	}
	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (p *parser) expectOneOf(expected []itemType, context string) item {
	token := p.next()
	for _, expectedType := range expected {
		if token.typ == expectedType {
			return token
		}
	}
	p.unexpected(token, context)
	return token
}

// unexpected complains about the token and terminates processing.
func (p *parser) unexpected(token item, context string) {
	if token.typ == itemError {
		extra := ""
		p.errorf("%s%s", token, extra)
	}
	p.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (p *parser) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*errp = e.(error)
	}
}

// parse is the top-level parser for the KQL query
func (p *parser) parse() {
	head := p.parseOr()
	// Handle implicit AND
	if p.peek().typ != itemEOF {
		andN := p.newAndNode(0)
		andN.append(head)
		for p.peek().typ != itemEOF {
			andN.append(p.parseOr())
		}
		p.Root = andN
		return
	}
	p.Root = head
}

func (p *parser) parseOr() Node {
	n := p.newOrNode(p.peek().pos)
	and := p.parseAnd()
	n.append(and)
	for p.peek().typ == itemOr {
		if p.disableComplexExpressions {
			p.errorf("complex expressions are not allowed")
		}
		p.next()
		and = p.parseAnd()
		n.append(and)
	}
	// simplify if only one node
	if len(n.Nodes) == 1 {
		return n.Nodes[0]
	}
	return n
}

func (p *parser) parseAnd() Node {
	n := p.newAndNode(p.peek().pos)
	not := p.parseNot()
	n.append(not)
	for p.peek().typ == itemAnd {
		if p.disableComplexExpressions {
			p.errorf("complex expressions are not allowed")
		}
		p.next()
		not = p.parseNot()
		n.append(not)
	}
	// simplify if only one node
	if len(n.Nodes) == 1 {
		return n.Nodes[0]
	}
	return n
}

func (p *parser) parseNot() Node {
	if p.peek().typ == itemNot {
		pos := p.peek().pos
		p.next()
		expr := p.parseSubQuery()
		return p.newNotNode(pos, expr)
	}
	return p.parseSubQuery()
}

func (p *parser) parseSubQuery() Node {
	if p.peek().typ == itemLeftParen {
		p.next()
		n := p.parseOr()
		p.expect(itemRightParen, "subquery")
		return n
	}
	return p.parseExpression()
}

func (p *parser) parseExpression() Node {
	switch p.peek().typ {
	case itemIdentifier:
		idItem := p.next()

		op := p.next()

		switch op.typ {
		case itemColon:
			value := p.parseListOfValues()
			return p.newIsNode(idItem.pos, idItem.val, value)
		case itemRangeOperator:
			value := p.parseValue()
			var rop RangeOperator
			switch op.val {
			case "<":
				rop = RangeOperatorLt
			case "<=":
				rop = RangeOperatorLte
			case ">":
				rop = RangeOperatorGt
			case ">=":
				rop = RangeOperatorGte
			}
			return p.newRangeNode(idItem.pos, idItem.val, rop, value)
		default:
			p.backup()
			return p.newLiteralNode(idItem.pos, idItem.val)
		}

	default:
		p.unexpected(p.peek(), "expression")
		return nil
	}
}

func (p *parser) parseListOfValues() Node {
	peeked := p.peek()
	if peeked.typ == itemLeftBrace {
		if p.disableComplexExpressions {
			p.errorf("complex expressions are not allowed")
		}
		p.next()
		n := p.parseOr()
		p.expect(itemRightBrace, "list of values")
		return p.newNestedNode(peeked.pos, n)
	}
	if peeked.typ == itemLeftParen {
		if p.disableComplexExpressions {
			p.errorf("complex expressions are not allowed")
		}
		p.next()
		n := p.parseOr()
		p.expect(itemRightParen, "list of values")
		return n
	}
	return p.parseValue()
}

func (p *parser) parseValue() Node {
	var value string

	// Allow prefix wildcard
	if p.peek().typ == itemWildcard {
		p.next()
		value = "*"
	}
	item := p.expectOneOf([]itemType{
		itemString,
		itemNumber,
		itemBool,
		itemIdentifier,
	}, "value")
	value += item.val
	if item.typ == itemString {
		// Strip the quotes
		value = value[1 : len(value)-1]
	}
	// Check for suffix wildcard
	if p.peek().typ == itemWildcard {
		p.next()
		value += "*"
	}

	return p.newLiteralNode(item.pos, value)
}
