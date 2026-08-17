// Package parser builds an ast.File from Octave source using a
// recursive-descent parser over the lexer's token stream.
package parser

import (
	"fmt"

	"github.com/matteo/octfmt/internal/ast"
	"github.com/matteo/octfmt/internal/lexer"
)

type Parser struct {
	toks []lexer.Token
	pos  int

	// ctxStack tracks whether we're directly inside a matrix/cell row
	// ('M') where bare whitespace separates elements and a space-before/
	// tight-after +/- starts a new element, or inside parens/args ('P')
	// where that rule is suspended.
	ctxStack []byte

	errors []error
}

// Parse tokenizes and parses src into a File. Parse errors are collected
// and returned rather than aborting; the returned File is a best-effort
// tree usable for formatting even when errors are present.
func Parse(src string) (*ast.File, []error) {
	lx := lexer.New(src)
	toks, err := lx.Tokenize()
	if err != nil {
		return nil, []error{err}
	}
	p := &Parser{toks: toks}
	f := &ast.File{}
	f.Stmts, f.TrailingBlankLines = p.parseBlock(stopSet())
	if p.cur().Type != lexer.EOF {
		p.errorf("unexpected token %v %q at end of parse", p.cur().Type, p.cur().Literal)
	}
	return f, p.errors
}

func (p *Parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Errorf("line %d: %s", p.cur().Pos.Line, fmt.Sprintf(format, args...)))
}

func (p *Parser) cur() lexer.Token { return p.toks[p.pos] }

func (p *Parser) peek(n int) lexer.Token {
	i := p.pos + n
	if i >= len(p.toks) {
		return p.toks[len(p.toks)-1] // EOF
	}
	return p.toks[i]
}

func (p *Parser) advance() lexer.Token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *Parser) expect(t lexer.TokenType) lexer.Token {
	if p.cur().Type != t {
		p.errorf("expected token %v, got %v %q", t, p.cur().Type, p.cur().Literal)
		return p.cur()
	}
	return p.advance()
}

func (p *Parser) pushCtx(c byte) { p.ctxStack = append(p.ctxStack, c) }
func (p *Parser) popCtx()        { p.ctxStack = p.ctxStack[:len(p.ctxStack)-1] }
func (p *Parser) inMatrixRow() bool {
	return len(p.ctxStack) > 0 && p.ctxStack[len(p.ctxStack)-1] == 'M'
}

func (p *Parser) makeComment(t lexer.Token) ast.Comment {
	return ast.Comment{Text: t.Literal, Block: t.Type == lexer.BLOCKCOMMENT}
}

// stopSet builds a membership predicate over the given token types.
func stopSet(types ...lexer.TokenType) func(lexer.TokenType) bool {
	set := make(map[lexer.TokenType]bool, len(types))
	for _, t := range types {
		set[t] = true
	}
	return func(t lexer.TokenType) bool { return set[t] }
}

func (p *Parser) skipBlankNewlines() int {
	n := 0
	for p.cur().Type == lexer.NEWLINE {
		p.advance()
		n++
	}
	return n
}

// headerTail consumes the optional comma/semicolon separator, an optional
// trailing comment, and the line-ending newline that follow a compound
// statement's header (e.g. after `if cond`, `for i = 1:10`, `else`).
func (p *Parser) headerTail() *ast.Comment {
	for p.cur().Type == lexer.COMMA || p.cur().Type == lexer.SEMICOLON {
		p.advance()
	}
	var c *ast.Comment
	if p.cur().Type == lexer.COMMENT || p.cur().Type == lexer.BLOCKCOMMENT {
		cc := p.makeComment(p.cur())
		c = &cc
		p.advance()
	}
	if p.cur().Type == lexer.NEWLINE {
		p.advance()
	}
	return c
}

// parseBlock parses statements (and standalone comments) until a token
// satisfying stop is reached, or EOF. It returns the statements and the
// number of blank lines immediately preceding the stop token.
func (p *Parser) parseBlock(stop func(lexer.TokenType) bool) ([]ast.Stmt, int) {
	var out []ast.Stmt
	for {
		blanks := p.skipBlankNewlines()
		if p.cur().Type == lexer.EOF || stop(p.cur().Type) {
			return out, blanks
		}
		if p.cur().Type == lexer.COMMENT || p.cur().Type == lexer.BLOCKCOMMENT {
			c := p.makeComment(p.cur())
			p.advance()
			if p.cur().Type == lexer.NEWLINE {
				p.advance()
			}
			cs := &ast.CommentStmt{C: c}
			cs.BlankLinesBefore = blanks
			out = append(out, cs)
			continue
		}

		first := true
		for {
			stmt := p.parseStatement()
			if stmt == nil {
				// parse error recovery: skip the offending token to avoid looping forever
				p.advance()
				break
			}
			base := stmt.Base()
			if first {
				base.BlankLinesBefore = blanks
				first = false
			}
			out = append(out, stmt)

			switch p.cur().Type {
			case lexer.SEMICOLON:
				base.Semicolon = true
				p.advance()
			case lexer.COMMA:
				p.advance()
			default:
				goto endline
			}
			if p.cur().Type == lexer.NEWLINE || p.cur().Type == lexer.EOF ||
				p.cur().Type == lexer.COMMENT || p.cur().Type == lexer.BLOCKCOMMENT ||
				stop(p.cur().Type) {
				goto endline
			}
			continue
		endline:
			if p.cur().Type == lexer.COMMENT || p.cur().Type == lexer.BLOCKCOMMENT {
				c := p.makeComment(p.cur())
				base.TrailingComment = &c
				p.advance()
			}
			if p.cur().Type == lexer.NEWLINE {
				p.advance()
			}
			break
		}
	}
}

func (p *Parser) parseStatement() ast.Stmt {
	switch p.cur().Type {
	case lexer.IF:
		return p.parseIf()
	case lexer.FOR, lexer.PARFOR:
		return p.parseFor()
	case lexer.WHILE:
		return p.parseWhile()
	case lexer.DO:
		return p.parseDoUntil()
	case lexer.SWITCH:
		return p.parseSwitch()
	case lexer.FUNCTION:
		return p.parseFunction()
	case lexer.RETURN:
		p.advance()
		return &ast.ReturnStmt{}
	case lexer.BREAK:
		p.advance()
		return &ast.BreakStmt{}
	case lexer.CONTINUE:
		p.advance()
		return &ast.ContinueStmt{}
	case lexer.GLOBAL:
		return p.parseGlobal()
	case lexer.PERSISTENT:
		return p.parsePersistent()
	case lexer.TRY:
		return p.parseTry()
	case lexer.UNWIND_PROTECT:
		return p.parseUnwindProtect()
	case lexer.LBRACKET:
		return p.parseBracketStmt()
	case lexer.IDENT:
		return p.parseIdentStmt()
	case lexer.ILLEGAL:
		p.errorf("illegal token %q", p.cur().Literal)
		p.advance()
		return nil
	default:
		x := p.parseExpr()
		return &ast.ExprStmt{X: x}
	}
}

func (p *Parser) parseBracketStmt() ast.Stmt {
	x := p.parseExpr()
	if p.cur().Type == lexer.ASSIGN {
		if mat, ok := x.(*ast.MatrixExpr); ok && len(mat.Rows) == 1 {
			p.advance()
			rhs := p.parseExpr()
			return &ast.MultiAssignStmt{Targets: mat.Rows[0], Rhs: rhs}
		}
		p.advance()
		rhs := p.parseExpr()
		return &ast.AssignStmt{Lhs: x, Op: lexer.ASSIGN, Rhs: rhs}
	}
	return &ast.ExprStmt{X: x}
}

func (p *Parser) parseIdentStmt() ast.Stmt {
	if cs, ok := p.tryParseCommand(); ok {
		return cs
	}
	x := p.parseExpr()
	switch p.cur().Type {
	case lexer.ASSIGN, lexer.PLUSEQ, lexer.MINUSEQ, lexer.STAREQ, lexer.SLASHEQ, lexer.CARETEQ:
		op := p.cur().Type
		p.advance()
		rhs := p.parseExpr()
		return &ast.AssignStmt{Lhs: x, Op: op, Rhs: rhs}
	default:
		return &ast.ExprStmt{X: x}
	}
}

var commandArgStopSet = map[lexer.TokenType]bool{
	lexer.NEWLINE: true, lexer.SEMICOLON: true, lexer.COMMA: true,
	lexer.EOF: true, lexer.COMMENT: true, lexer.BLOCKCOMMENT: true,
}

// tryParseCommand recognizes Octave "command syntax" like `hold on`,
// `format long g`, `pkg load io`. It only fires when doing so cannot be
// confused with a normal expression or assignment statement.
func (p *Parser) tryParseCommand() (ast.Stmt, bool) {
	next := p.peek(1)
	if !next.PrecededBySpace {
		return nil, false
	}
	switch next.Type {
	case lexer.IDENT, lexer.STRING:
	default:
		return nil, false
	}
	// bail if a top-level assignment operator appears later on the line
	depth := 0
	for i := p.pos + 1; i < len(p.toks); i++ {
		t := p.toks[i].Type
		if t == lexer.NEWLINE || t == lexer.SEMICOLON || t == lexer.EOF ||
			t == lexer.COMMENT || t == lexer.BLOCKCOMMENT {
			break
		}
		switch t {
		case lexer.LPAREN, lexer.LBRACKET, lexer.LBRACE:
			depth++
		case lexer.RPAREN, lexer.RBRACKET, lexer.RBRACE:
			depth--
		case lexer.ASSIGN, lexer.PLUSEQ, lexer.MINUSEQ, lexer.STAREQ, lexer.SLASHEQ, lexer.CARETEQ:
			if depth == 0 {
				return nil, false
			}
		}
	}

	name := p.cur().Literal
	p.advance()
	var args []string
	for !commandArgStopSet[p.cur().Type] {
		args = append(args, p.cur().Literal)
		p.advance()
	}
	return &ast.CommandStmt{Name: name, Args: args}, true
}

// ---- compound statements ----

func (p *Parser) parseIf() ast.Stmt {
	p.advance() // IF
	cond := p.parseExpr()
	condComment := p.headerTail()
	body, _ := p.parseBlock(stopSet(lexer.ELSEIF, lexer.ELSE, lexer.ENDIF, lexer.END))

	var elseifs []ast.ElseIfClause
	for p.cur().Type == lexer.ELSEIF {
		p.advance()
		c := p.parseExpr()
		tc := p.headerTail()
		b, _ := p.parseBlock(stopSet(lexer.ELSEIF, lexer.ELSE, lexer.ENDIF, lexer.END))
		elseifs = append(elseifs, ast.ElseIfClause{Cond: c, Body: b, TrailingComment: tc})
	}

	var elseBody []ast.Stmt
	var elseComment *ast.Comment
	hasElse := false
	if p.cur().Type == lexer.ELSE {
		hasElse = true
		p.advance()
		elseComment = p.headerTail()
		elseBody, _ = p.parseBlock(stopSet(lexer.ENDIF, lexer.END))
	}

	if p.cur().Type == lexer.ENDIF || p.cur().Type == lexer.END {
		p.advance()
	} else {
		p.errorf("expected 'end' or 'endif' to close if-statement")
	}

	return &ast.IfStmt{
		Cond: cond, CondComment: condComment, Body: body,
		ElseIfs: elseifs, HasElse: hasElse, Else: elseBody, ElseComment: elseComment,
	}
}

func (p *Parser) parseFor() ast.Stmt {
	parfor := p.cur().Type == lexer.PARFOR
	p.advance()
	varExpr := p.parseExpr()
	p.expect(lexer.ASSIGN)
	rangeExpr := p.parseExpr()

	var maxProc ast.Expr
	if parfor && p.cur().Type == lexer.COMMA {
		save := p.pos
		p.advance()
		t := p.cur().Type
		if t == lexer.COMMENT || t == lexer.BLOCKCOMMENT || t == lexer.NEWLINE ||
			t == lexer.SEMICOLON || t == lexer.EOF {
			p.pos = save
		} else {
			maxProc = p.parseExpr()
		}
	}

	headerComment := p.headerTail()
	stops := stopSet(lexer.ENDFOR, lexer.ENDPARFOR, lexer.END)
	body, _ := p.parseBlock(stops)
	if lexer.IsBlockEnd(p.cur().Type) {
		p.advance()
	} else {
		p.errorf("expected 'end'/'endfor' to close for-statement")
	}
	return &ast.ForStmt{
		Parfor: parfor, Var: varExpr, Range: rangeExpr, MaxProc: maxProc,
		HeaderComment: headerComment, Body: body,
	}
}

func (p *Parser) parseWhile() ast.Stmt {
	p.advance() // WHILE
	cond := p.parseExpr()
	headerComment := p.headerTail()
	body, _ := p.parseBlock(stopSet(lexer.ENDWHILE, lexer.END))
	if lexer.IsBlockEnd(p.cur().Type) {
		p.advance()
	} else {
		p.errorf("expected 'end'/'endwhile' to close while-statement")
	}
	return &ast.WhileStmt{Cond: cond, HeaderComment: headerComment, Body: body}
}

func (p *Parser) parseDoUntil() ast.Stmt {
	p.advance() // DO
	headerComment := p.headerTail()
	body, _ := p.parseBlock(stopSet(lexer.UNTIL))
	if p.cur().Type == lexer.UNTIL {
		p.advance()
	} else {
		p.errorf("expected 'until' to close do-statement")
	}
	cond := p.parseExpr()
	return &ast.DoUntilStmt{Body: body, Cond: cond, HeaderComment: headerComment}
}

func (p *Parser) parseSwitch() ast.Stmt {
	p.advance() // SWITCH
	tag := p.parseExpr()
	headerComment := p.headerTail()
	stops := stopSet(lexer.CASE, lexer.OTHERWISE, lexer.ENDSWITCH, lexer.END)
	_, blanks := p.parseBlock(stops) // any content before first `case` is discarded (comments only, permissively)

	var cases []ast.CaseClause
	nextBlanks := blanks
	for p.cur().Type == lexer.CASE {
		p.advance()
		ce := p.parseExpr()
		tc := p.headerTail()
		body, b2 := p.parseBlock(stops)
		cases = append(cases, ast.CaseClause{Expr: ce, Body: body, TrailingComment: tc, BlankLinesBefore: nextBlanks})
		nextBlanks = b2
	}

	var otherwiseBody []ast.Stmt
	var otherwiseComment *ast.Comment
	otherwiseBlanks := nextBlanks
	hasOtherwise := false
	if p.cur().Type == lexer.OTHERWISE {
		hasOtherwise = true
		p.advance()
		otherwiseComment = p.headerTail()
		otherwiseBody, _ = p.parseBlock(stopSet(lexer.ENDSWITCH, lexer.END))
	}

	if lexer.IsBlockEnd(p.cur().Type) {
		p.advance()
	} else {
		p.errorf("expected 'end'/'endswitch' to close switch-statement")
	}

	return &ast.SwitchStmt{
		Tag: tag, HeaderComment: headerComment, Cases: cases,
		HasOtherwise: hasOtherwise, Otherwise: otherwiseBody, OtherwiseComment: otherwiseComment,
		OtherwiseBlankLines: otherwiseBlanks,
	}
}

func (p *Parser) parseFunction() ast.Stmt {
	p.advance() // FUNCTION
	var outputs []ast.Expr
	var name string

	if p.cur().Type == lexer.LBRACKET {
		p.advance()
		outputs = p.parseIdentListUntil(lexer.RBRACKET)
		p.expect(lexer.RBRACKET)
		p.expect(lexer.ASSIGN)
		name = p.expect(lexer.IDENT).Literal
	} else {
		first := p.expect(lexer.IDENT).Literal
		if p.cur().Type == lexer.ASSIGN {
			outputs = []ast.Expr{&ast.Ident{Name: first}}
			p.advance()
			name = p.expect(lexer.IDENT).Literal
		} else {
			name = first
		}
	}

	var params []ast.Expr
	if p.cur().Type == lexer.LPAREN {
		p.advance()
		params = p.parseIdentListUntil(lexer.RPAREN)
		p.expect(lexer.RPAREN)
	}

	headerComment := p.headerTail()
	body, _ := p.parseBlock(stopSet(lexer.ENDFUNCTION, lexer.END, lexer.FUNCTION))
	hasEnd := false
	if p.cur().Type == lexer.ENDFUNCTION || p.cur().Type == lexer.END {
		hasEnd = true
		p.advance()
	}

	return &ast.FunctionStmt{
		Outputs: outputs, Name: name, Params: params,
		HeaderComment: headerComment, Body: body, HasEnd: hasEnd,
	}
}

func (p *Parser) parseIdentListUntil(close lexer.TokenType) []ast.Expr {
	var out []ast.Expr
	if p.cur().Type == close {
		return out
	}
	for {
		if p.cur().Type == lexer.NOT {
			out = append(out, &ast.TildeExpr{})
			p.advance()
		} else {
			name := p.expect(lexer.IDENT).Literal
			out = append(out, &ast.Ident{Name: name})
		}
		if p.cur().Type == lexer.COMMA {
			p.advance()
			continue
		}
		break
	}
	return out
}

func (p *Parser) parseGlobal() ast.Stmt {
	p.advance()
	names := p.parseGlobalVarList()
	return &ast.GlobalStmt{Names: names}
}

func (p *Parser) parsePersistent() ast.Stmt {
	p.advance()
	names := p.parseGlobalVarList()
	return &ast.PersistentStmt{Names: names}
}

func (p *Parser) parseGlobalVarList() []ast.GlobalVar {
	var names []ast.GlobalVar
	for p.cur().Type == lexer.IDENT {
		n := p.cur().Literal
		p.advance()
		var init ast.Expr
		if p.cur().Type == lexer.ASSIGN {
			p.advance()
			init = p.parseExpr()
		}
		names = append(names, ast.GlobalVar{Name: n, Init: init})
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}
	return names
}

func (p *Parser) parseTry() ast.Stmt {
	p.advance() // TRY
	headerTailComment := p.headerTail()
	_ = headerTailComment // try's own header rarely carries a comment worth keeping separately; folded into body via blank handling
	body, blanks := p.parseBlock(stopSet(lexer.CATCH, lexer.END_TRY_CATCH, lexer.END))

	var catchVar ast.Expr
	var catchComment *ast.Comment
	var catchBody []ast.Stmt
	catchBlanks := blanks
	hasCatch := false
	if p.cur().Type == lexer.CATCH {
		hasCatch = true
		p.advance()
		if p.cur().Type == lexer.IDENT {
			nt := p.peek(1).Type
			if nt == lexer.NEWLINE || nt == lexer.SEMICOLON || nt == lexer.COMMENT ||
				nt == lexer.BLOCKCOMMENT || nt == lexer.COMMA || nt == lexer.EOF {
				catchVar = &ast.Ident{Name: p.cur().Literal}
				p.advance()
			}
		}
		catchComment = p.headerTail()
		catchBody, _ = p.parseBlock(stopSet(lexer.END_TRY_CATCH, lexer.END))
	}

	if lexer.IsBlockEnd(p.cur().Type) {
		p.advance()
	} else {
		p.errorf("expected 'end'/'end_try_catch' to close try-statement")
	}

	return &ast.TryStmt{
		Body: body, HasCatch: hasCatch, CatchVar: catchVar, CatchComment: catchComment,
		CatchBody: catchBody, CatchBlankLines: catchBlanks,
	}
}

func (p *Parser) parseUnwindProtect() ast.Stmt {
	p.advance() // UNWIND_PROTECT
	headerComment := p.headerTail()
	body, blanks := p.parseBlock(stopSet(lexer.UNWIND_PROTECT_CLEANUP, lexer.END_UNWIND_PROTECT, lexer.END))

	var cleanup []ast.Stmt
	cleanupBlanks := blanks
	hasCleanup := false
	if p.cur().Type == lexer.UNWIND_PROTECT_CLEANUP {
		hasCleanup = true
		p.advance()
		p.headerTail()
		cleanup, _ = p.parseBlock(stopSet(lexer.END_UNWIND_PROTECT, lexer.END))
	}

	if lexer.IsBlockEnd(p.cur().Type) {
		p.advance()
	} else {
		p.errorf("expected 'end'/'end_unwind_protect' to close unwind_protect-statement")
	}

	return &ast.UnwindProtectStmt{
		HeaderComment: headerComment, Body: body, HasCleanup: hasCleanup,
		Cleanup: cleanup, CleanupBlankLines: cleanupBlanks,
	}
}
