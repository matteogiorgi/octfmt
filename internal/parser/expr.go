package parser

import (
	"github.com/matteo/octave-formatter/internal/ast"
	"github.com/matteo/octave-formatter/internal/lexer"
)

// Operator precedence, low to high, per Octave's documented table:
// || < && < | < & < relational < : (range) < + - < * / \ .* ./ .\ < unary < ^ .^ < postfix

func (p *Parser) parseExpr() ast.Expr { return p.parseOr() }

func (p *Parser) parseOr() ast.Expr {
	x := p.parseAnd()
	for p.cur().Type == lexer.OR {
		op := p.advance().Type
		y := p.parseAnd()
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func (p *Parser) parseAnd() ast.Expr {
	x := p.parseBitOr()
	for p.cur().Type == lexer.AND {
		op := p.advance().Type
		y := p.parseBitOr()
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func (p *Parser) parseBitOr() ast.Expr {
	x := p.parseBitAnd()
	for p.cur().Type == lexer.BITOR {
		op := p.advance().Type
		y := p.parseBitAnd()
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func (p *Parser) parseBitAnd() ast.Expr {
	x := p.parseRel()
	for p.cur().Type == lexer.BITAND {
		op := p.advance().Type
		y := p.parseRel()
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func isRelOp(t lexer.TokenType) bool {
	switch t {
	case lexer.EQ, lexer.NE, lexer.LT, lexer.GT, lexer.LE, lexer.GE:
		return true
	}
	return false
}

func (p *Parser) parseRel() ast.Expr {
	x := p.parseRange()
	for isRelOp(p.cur().Type) {
		op := p.advance().Type
		y := p.parseRange()
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func (p *Parser) parseRange() ast.Expr {
	start := p.parseAdd()
	if p.cur().Type != lexer.COLON {
		return start
	}
	p.advance()
	mid := p.parseAdd()
	if p.cur().Type == lexer.COLON {
		p.advance()
		stop := p.parseAdd()
		return &ast.RangeExpr{Start: start, Step: mid, Stop: stop}
	}
	return &ast.RangeExpr{Start: start, Stop: mid}
}

// startsNewMatrixElement reports whether, given we're directly inside a
// matrix/cell row, the +/- operator token op should be treated as the start
// of a new element (space before it, no space after) rather than a binary
// operator continuing the current element.
func (p *Parser) startsNewMatrixElement(op lexer.Token) bool {
	if !p.inMatrixRow() {
		return false
	}
	if op.Type != lexer.PLUS && op.Type != lexer.MINUS {
		return false
	}
	if !op.PrecededBySpace {
		return false
	}
	return !p.peek(1).PrecededBySpace
}

func (p *Parser) parseAdd() ast.Expr {
	x := p.parseMul()
	for p.cur().Type == lexer.PLUS || p.cur().Type == lexer.MINUS {
		if p.startsNewMatrixElement(p.cur()) {
			break
		}
		op := p.advance().Type
		y := p.parseMul()
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func isMulOp(t lexer.TokenType) bool {
	switch t {
	case lexer.STAR, lexer.SLASH, lexer.BACKSLASH, lexer.DOTSTAR, lexer.DOTSLASH, lexer.DOTBACKSLASH:
		return true
	}
	return false
}

func (p *Parser) parseMul() ast.Expr {
	x := p.parseUnary()
	for isMulOp(p.cur().Type) {
		op := p.advance().Type
		y := p.parseUnary()
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func (p *Parser) parseUnary() ast.Expr {
	switch p.cur().Type {
	case lexer.PLUS, lexer.MINUS, lexer.NOT:
		if p.cur().Type == lexer.NOT && p.isBareTildePlaceholder() {
			p.advance()
			return &ast.TildeExpr{}
		}
		op := p.advance().Type
		x := p.parseUnary()
		return &ast.UnaryExpr{Op: op, X: x}
	default:
		return p.parsePow()
	}
}

// isBareTildePlaceholder reports whether a '~'/'!' token is being used as
// the ignored-output placeholder in a multi-assign target list, e.g.
// `[~, b] = f(x)`, rather than as the logical-not operator.
func (p *Parser) isBareTildePlaceholder() bool {
	switch p.peek(1).Type {
	case lexer.COMMA, lexer.RBRACKET:
		return true
	}
	return false
}

func (p *Parser) parsePow() ast.Expr {
	x := p.parsePostfix()
	for p.cur().Type == lexer.CARET || p.cur().Type == lexer.DOTCARET {
		op := p.advance().Type
		y := p.parseUnary() // allows `2^-1`
		x = &ast.BinaryExpr{Op: op, X: x, Y: y}
	}
	return x
}

func (p *Parser) parsePostfix() ast.Expr {
	x := p.parsePrimary()
	for {
		switch p.cur().Type {
		case lexer.LPAREN:
			p.advance()
			p.pushCtx('P')
			args := p.parseArgList(lexer.RPAREN)
			p.popCtx()
			p.expect(lexer.RPAREN)
			x = &ast.IndexExpr{Fn: x, Args: args}
		case lexer.LBRACE:
			p.advance()
			p.pushCtx('P')
			args := p.parseArgList(lexer.RBRACE)
			p.popCtx()
			p.expect(lexer.RBRACE)
			x = &ast.CellIndexExpr{Fn: x, Args: args}
		case lexer.DOT:
			p.advance()
			if p.cur().Type == lexer.LPAREN {
				p.advance()
				p.pushCtx('P')
				name := p.parseExpr()
				p.popCtx()
				p.expect(lexer.RPAREN)
				x = &ast.DynFieldExpr{X: x, Name: name}
			} else {
				name := p.expect(lexer.IDENT).Literal
				x = &ast.FieldExpr{X: x, Name: name}
			}
		case lexer.TRANSPOSE:
			p.advance()
			x = &ast.UnaryExpr{Op: lexer.TRANSPOSE, X: x, Postfix: true}
		case lexer.DOTTRANSPOSE:
			p.advance()
			x = &ast.UnaryExpr{Op: lexer.DOTTRANSPOSE, X: x, Postfix: true}
		default:
			return x
		}
	}
}

func (p *Parser) parseArgList(close lexer.TokenType) []ast.Expr {
	var args []ast.Expr
	if p.cur().Type == close {
		return args
	}
	for {
		args = append(args, p.parseExpr())
		if p.cur().Type == lexer.COMMA {
			p.advance()
			continue
		}
		break
	}
	return args
}

func (p *Parser) parsePrimary() ast.Expr {
	tok := p.cur()
	switch tok.Type {
	case lexer.IDENT:
		p.advance()
		return &ast.Ident{Name: tok.Literal}
	case lexer.NUMBER:
		p.advance()
		return &ast.NumberLit{Value: tok.Literal}
	case lexer.STRING:
		p.advance()
		return &ast.StringLit{Value: tok.Literal, Quote: tok.Literal[0]}
	case lexer.END:
		p.advance()
		return &ast.EndExpr{}
	case lexer.COLON:
		p.advance()
		return &ast.ColonExpr{}
	case lexer.LPAREN:
		p.advance()
		p.pushCtx('P')
		x := p.parseExpr()
		p.popCtx()
		p.expect(lexer.RPAREN)
		return &ast.ParenExpr{X: x}
	case lexer.LBRACKET:
		rows := p.parseMatrixBody(lexer.RBRACKET)
		return &ast.MatrixExpr{Rows: rows}
	case lexer.LBRACE:
		rows := p.parseMatrixBody(lexer.RBRACE)
		return &ast.CellExpr{Rows: rows}
	case lexer.AT:
		p.advance()
		if p.cur().Type == lexer.LPAREN {
			p.advance()
			params := p.parseIdentListUntil(lexer.RPAREN)
			p.expect(lexer.RPAREN)
			body := p.parseExpr()
			return &ast.AnonFuncExpr{Params: params, Body: body}
		}
		name := p.expect(lexer.IDENT).Literal
		return &ast.FuncHandleExpr{Name: name}
	default:
		p.errorf("unexpected token %v %q in expression", tok.Type, tok.Literal)
		p.advance()
		return &ast.Ident{Name: tok.Literal}
	}
}

// parseMatrixBody parses the row-structured body of a `[...]` or `{...}`
// literal, up to and including the closing token.
func (p *Parser) parseMatrixBody(close lexer.TokenType) [][]ast.Expr {
	p.advance() // opening bracket
	p.pushCtx('M')
	defer p.popCtx()

	p.skipBlankNewlines()
	var rows [][]ast.Expr
	for p.cur().Type != close && p.cur().Type != lexer.EOF {
		row := p.parseMatrixRow(close)
		rows = append(rows, row)
		for p.cur().Type == lexer.SEMICOLON || p.cur().Type == lexer.NEWLINE {
			p.advance()
		}
	}
	p.expect(close)
	return rows
}

func (p *Parser) parseMatrixRow(close lexer.TokenType) []ast.Expr {
	var elems []ast.Expr
	for {
		elems = append(elems, p.parseExpr())
		switch p.cur().Type {
		case lexer.COMMA:
			p.advance()
			if p.cur().Type == close || p.cur().Type == lexer.SEMICOLON || p.cur().Type == lexer.NEWLINE {
				return elems
			}
			continue
		case close, lexer.SEMICOLON, lexer.NEWLINE, lexer.EOF:
			return elems
		default:
			// adjacency: another element follows, separated only by whitespace
			continue
		}
	}
}
