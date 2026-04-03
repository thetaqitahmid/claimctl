// Package utils provides utility functions.
//
// Label Filter Implementation Overview:
// -------------------------------------
// The label filter evaluates boolean expressions against a list of resource labels.
// It consists of two main phases: Lexing and Parsing.
//
// 1. Lexing (Input String -> Tokens)
//    Input:  "gpu AND (linux OR windows)"
//    Tokens: [LABEL("gpu")] [AND] [LPAREN] [LABEL("linux")] [OR] [LABEL("windows")] [RPAREN]
//
// 2. Parsing (Tokens -> Abstract Syntax Tree)
//    Builds an evaluatable AST using a Recursive Descent Parser.
//    Precedence: NOT > AND > OR (overridden by parentheses)
//
//           [andExpr]
//           /       \
//  [labelExpr("gpu")] [orExpr]
//                     /      \
//       [labelExpr("linux")] [labelExpr("windows")]
//
// 3. Evaluation (AST.Matches(labels) -> bool)
//    filter.Matches([]string{"gpu", "linux"}) -> true
package utils

import (
	"fmt"
	"strings"
	"unicode"
)

// FilterExpr represents a boolean label expression.
type FilterExpr interface {
	Matches(labels []string) bool
}

type andExpr struct {
	left, right FilterExpr
}

func (e *andExpr) Matches(labels []string) bool {
	return e.left.Matches(labels) && e.right.Matches(labels)
}

type orExpr struct {
	left, right FilterExpr
}

func (e *orExpr) Matches(labels []string) bool {
	return e.left.Matches(labels) || e.right.Matches(labels)
}

type notExpr struct {
	expr FilterExpr
}

func (e *notExpr) Matches(labels []string) bool {
	return !e.expr.Matches(labels)
}

type labelExpr struct {
	label string
}

func (e *labelExpr) Matches(labels []string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, e.label) {
			return true
		}
	}
	return false
}

// Token types
type tokenType int

const (
	tokAND tokenType = iota
	tokOR
	tokNOT
	tokLPAREN
	tokRPAREN
	tokLABEL
	tokEOF
)

type token struct {
	t   tokenType
	val string
}

// lexer converts an expression string into tokens.
func lex(input string) ([]token, error) {
	var tokens []token
	runes := []rune(input)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if r == '(' {
			tokens = append(tokens, token{t: tokLPAREN})
			i++
			continue
		}
		if r == ')' {
			tokens = append(tokens, token{t: tokRPAREN})
			i++
			continue
		}

		// Read a word or quoted string
		var valBuilder strings.Builder
		if r == '"' || r == '\'' {
			quote := r
			i++
			for i < len(runes) && runes[i] != quote {
				valBuilder.WriteRune(runes[i])
				i++
			}
			if i == len(runes) {
				return nil, fmt.Errorf("unclosed quote")
			}
			i++
			val := valBuilder.String()
			tokens = append(tokens, token{t: tokLABEL, val: val})
		} else {
			for i < len(runes) && !unicode.IsSpace(runes[i]) && runes[i] != '(' && runes[i] != ')' {
				valBuilder.WriteRune(runes[i])
				i++
			}
			val := valBuilder.String()
			upper := strings.ToUpper(val)
			switch upper {
			case "AND":
				tokens = append(tokens, token{t: tokAND})
			case "OR":
				tokens = append(tokens, token{t: tokOR})
			case "NOT":
				tokens = append(tokens, token{t: tokNOT})
			default:
				tokens = append(tokens, token{t: tokLABEL, val: val})
			}
		}
	}
	tokens = append(tokens, token{t: tokEOF})
	return tokens, nil
}

// parser builds an AST from tokens.
type parser struct {
	tokens []token
	pos    int
}

func newParser(tokens []token) *parser {
	return &parser{tokens: tokens, pos: 0}
}

func (p *parser) next() token {
	if p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		p.pos++
		return t
	}
	return token{t: tokEOF}
}

func (p *parser) peek() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return token{t: tokEOF}
}

// expression -> term (OR term)*
func (p *parser) parseExpression() (FilterExpr, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.peek().t == tokOR {
		p.next() // consume OR
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &orExpr{left: left, right: right}
	}
	return left, nil
}

// term -> factor (AND factor)*
func (p *parser) parseTerm() (FilterExpr, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for p.peek().t == tokAND {
		p.next() // consume AND
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &andExpr{left: left, right: right}
	}
	return left, nil
}

// factor -> NOT factor | LPAREN expression RPAREN | LABEL
func (p *parser) parseFactor() (FilterExpr, error) {
	tok := p.next()
	if tok.t == tokNOT {
		expr, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &notExpr{expr: expr}, nil
	}

	if tok.t == tokLPAREN {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.next().t != tokRPAREN {
			return nil, fmt.Errorf("expected ')'")
		}
		return expr, nil
	}

	if tok.t == tokLABEL {
		return &labelExpr{label: tok.val}, nil
	}

	return nil, fmt.Errorf("unexpected token: %v", tok.val)
}

// ParseLabelFilter parses a boolean expression string into a FilterExpr.
// If the expression is empty, it returns nil and no error.
func ParseLabelFilter(input string) (FilterExpr, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	tokens, err := lex(input)
	if err != nil {
		return nil, err
	}
	p := newParser(tokens)
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek().t != tokEOF {
		return nil, fmt.Errorf("unexpected tokens after end of expression")
	}
	return expr, nil
}
