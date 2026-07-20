// Package lexer tokenizes GNU Octave source code.
package lexer

import (
	"fmt"
	"strings"
)

// Lexer converts Octave source text into a stream of Tokens.
type Lexer struct {
	src []rune

	pos     int // index of ch in src
	readPos int // index of next rune to read
	ch      rune

	line   int
	column int

	// lastSignificant is the type of the last non-whitespace, non-comment
	// token produced, used to disambiguate ' (transpose) from a string start.
	lastSignificant TokenType
	haveLast        bool

	// blockCommentDepth tracks nested %{ %} / #{ #} comments.
	blockCommentDepth int
}

// New creates a Lexer over src.
func New(src string) *Lexer {
	l := &Lexer{src: []rune(src), line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.src) {
		l.ch = 0
	} else {
		l.ch = l.src[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.src) {
		return 0
	}
	return l.src[l.readPos]
}

func (l *Lexer) peekAt(offset int) rune {
	idx := l.pos + offset
	if idx < 0 || idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

func (l *Lexer) curPos() Position {
	return Position{Line: l.line, Column: l.column, Offset: l.pos}
}

// Tokenize returns all tokens in src, ending with an EOF token.
func (l *Lexer) Tokenize() ([]Token, error) {
	var toks []Token
	for {
		tok, err := l.Next()
		if err != nil {
			return toks, err
		}
		toks = append(toks, tok)
		if tok.Type == EOF {
			break
		}
	}
	return toks, nil
}

// Next scans and returns the next token.
func (l *Lexer) Next() (Token, error) {
	precededBySpace := l.skipSpaceTrackingSpace()

	pos := l.curPos()

	if l.ch == 0 {
		return l.emit(EOF, "", pos, precededBySpace), nil
	}

	switch {
	case l.ch == '\n':
		l.readChar()
		return l.emit(NEWLINE, "\n", pos, precededBySpace), nil
	case l.ch == '\r':
		l.readChar()
		if l.ch == '\n' {
			l.readChar()
		}
		return l.emit(NEWLINE, "\n", pos, precededBySpace), nil
	case l.ch == '%' || l.ch == '#':
		return l.readCommentOrBlock(pos, precededBySpace)
	case l.ch == '\'':
		if l.transposeContext() {
			l.readChar()
			return l.emit(TRANSPOSE, "'", pos, precededBySpace), nil
		}
		return l.readString('\'', pos, precededBySpace)
	case l.ch == '"':
		return l.readString('"', pos, precededBySpace)
	case isDigit(l.ch) || (l.ch == '.' && isDigit(l.peekChar())):
		return l.readNumber(pos, precededBySpace)
	case isIdentStart(l.ch):
		return l.readIdent(pos, precededBySpace)
	default:
		return l.readOperator(pos, precededBySpace)
	}
}

// skipSpaceTrackingSpace consumes spaces and tabs (not newlines) and
// reports whether any were consumed.
func (l *Lexer) skipSpaceTrackingSpace() bool {
	saw := false
	for l.ch == ' ' || l.ch == '\t' {
		saw = true
		l.readChar()
	}
	return saw
}

func (l *Lexer) emit(t TokenType, lit string, pos Position, space bool) Token {
	tok := Token{Type: t, Literal: lit, Pos: pos, PrecededBySpace: space}
	switch t {
	case NEWLINE, COMMENT, BLOCKCOMMENT:
		// not significant for transpose-context tracking
	default:
		l.lastSignificant = t
		l.haveLast = true
	}
	return tok
}

// transposeContext reports whether a following ' should be lexed as the
// transpose operator (true) rather than the start of a string literal.
// It is transpose only when immediately preceded (no space) by an
// identifier, number, closing bracket/paren/brace, string, or another
// transpose - matching Octave/MATLAB's own disambiguation rule.
func (l *Lexer) transposeContext() bool {
	if !l.haveLast {
		return false
	}
	// If whitespace or newline immediately precedes this ', it can't be transpose.
	if l.pos > 0 {
		prevCh := l.src[l.pos-1]
		if prevCh == ' ' || prevCh == '\t' || prevCh == '\n' || prevCh == '\r' {
			return false
		}
	}
	switch l.lastSignificant {
	case IDENT, NUMBER, RPAREN, RBRACKET, RBRACE, TRANSPOSE, DOTTRANSPOSE, STRING:
		return true
	}
	return false
}

func (l *Lexer) readCommentOrBlock(pos Position, space bool) (Token, error) {
	marker := l.ch // % or #
	// Check for block comment: marker immediately followed by '{' and the
	// rest of the line (after trimming) is empty.
	if l.peekChar() == '{' && l.restOfLineIsOnly(marker, '{') {
		return l.readBlockComment(marker, pos, space)
	}
	// line comment: consume to end of line
	var sb strings.Builder
	for l.ch != '\n' && l.ch != 0 && l.ch != '\r' {
		sb.WriteRune(l.ch)
		l.readChar()
	}
	return l.emit(COMMENT, sb.String(), pos, space), nil
}

// restOfLineIsOnly checks (without consuming) that starting at the current
// position the line consists of marker, delim, then only whitespace until
// newline/EOF - i.e. this is a standalone "%{" style block-comment fence.
func (l *Lexer) restOfLineIsOnly(marker, delim rune) bool {
	i := l.pos + 2 // skip marker and delim
	for i < len(l.src) {
		c := l.src[i]
		if c == '\n' {
			return true
		}
		if c != ' ' && c != '\t' && c != '\r' {
			return false
		}
		i++
	}
	return true
}

func (l *Lexer) readBlockComment(marker rune, pos Position, space bool) (Token, error) {
	var sb strings.Builder
	depth := 0
	for {
		if l.ch == 0 {
			return Token{}, fmt.Errorf("unterminated block comment starting at line %d", pos.Line)
		}
		lineStart := l.pos
		// read whole line
		for l.ch != '\n' && l.ch != 0 {
			sb.WriteRune(l.ch)
			l.readChar()
		}
		line := strings.TrimSpace(string(l.src[lineStart:l.pos]))
		if line == string(marker)+"{" {
			depth++
		} else if line == string(marker)+"}" {
			depth--
		}
		if l.ch == '\n' {
			sb.WriteRune('\n')
			l.readChar()
		}
		if depth == 0 {
			break
		}
	}
	return l.emit(BLOCKCOMMENT, sb.String(), pos, space), nil
}

func (l *Lexer) readString(quote rune, pos Position, space bool) (Token, error) {
	var sb strings.Builder
	sb.WriteRune(quote)
	l.readChar()
	for {
		if l.ch == 0 || l.ch == '\n' {
			return Token{}, fmt.Errorf("unterminated string literal at line %d", pos.Line)
		}
		if l.ch == quote {
			if l.peekChar() == quote {
				// escaped quote via doubling
				sb.WriteRune(quote)
				sb.WriteRune(quote)
				l.readChar()
				l.readChar()
				continue
			}
			sb.WriteRune(quote)
			l.readChar()
			break
		}
		if quote == '"' && l.ch == '\\' {
			// double-quoted strings support backslash escapes; keep raw.
			sb.WriteRune(l.ch)
			l.readChar()
			if l.ch != 0 {
				sb.WriteRune(l.ch)
				l.readChar()
			}
			continue
		}
		sb.WriteRune(l.ch)
		l.readChar()
	}
	return l.emit(STRING, sb.String(), pos, space), nil
}

func (l *Lexer) readNumber(pos Position, space bool) (Token, error) {
	var sb strings.Builder
	if l.ch == '0' && (l.peekChar() == 'x' || l.peekChar() == 'X') {
		sb.WriteRune(l.ch)
		l.readChar()
		sb.WriteRune(l.ch)
		l.readChar()
		for isHexDigit(l.ch) {
			sb.WriteRune(l.ch)
			l.readChar()
		}
		return l.emit(NUMBER, sb.String(), pos, space), nil
	}
	if l.ch == '0' && (l.peekChar() == 'b' || l.peekChar() == 'B') {
		sb.WriteRune(l.ch)
		l.readChar()
		sb.WriteRune(l.ch)
		l.readChar()
		for l.ch == '0' || l.ch == '1' {
			sb.WriteRune(l.ch)
			l.readChar()
		}
		return l.emit(NUMBER, sb.String(), pos, space), nil
	}
	for isDigit(l.ch) {
		sb.WriteRune(l.ch)
		l.readChar()
	}
	if l.ch == '.' && l.peekChar() != '\'' { // avoid gobbling into transpose/dot-op ambiguity handled elsewhere
		// don't treat ".*" ".^" etc following digits as decimal point
		next := l.peekChar()
		if isDigit(next) || !isOperatorStartAfterDot(next) {
			sb.WriteRune(l.ch)
			l.readChar()
			for isDigit(l.ch) {
				sb.WriteRune(l.ch)
				l.readChar()
			}
		}
	}
	if l.ch == 'e' || l.ch == 'E' {
		save := l.pos
		exp := string(l.ch)
		saveCh := l.ch
		saveReadPos := l.readPos
		saveLine, saveCol := l.line, l.column
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			exp += string(l.ch)
			l.readChar()
		}
		if isDigit(l.ch) {
			for isDigit(l.ch) {
				exp += string(l.ch)
				l.readChar()
			}
			sb.WriteString(exp)
		} else {
			// not a real exponent; rewind
			l.pos, l.readPos, l.line, l.column, l.ch = save, saveReadPos, saveLine, saveCol, saveCh
		}
	}
	if l.ch == 'i' || l.ch == 'j' || l.ch == 'I' || l.ch == 'J' {
		sb.WriteRune(l.ch)
		l.readChar()
	}
	return l.emit(NUMBER, sb.String(), pos, space), nil
}

// isOperatorStartAfterDot reports whether c following a '.' after digits
// indicates the '.' belongs to a dotted operator (.*, ./, .^, .\, .') rather
// than a decimal point.
func isOperatorStartAfterDot(c rune) bool {
	switch c {
	case '*', '/', '^', '\\', '\'':
		return true
	}
	return false
}

func (l *Lexer) readIdent(pos Position, space bool) (Token, error) {
	var sb strings.Builder
	for isIdentPart(l.ch) {
		sb.WriteRune(l.ch)
		l.readChar()
	}
	lit := sb.String()
	tt := LookupIdent(lit)
	return l.emit(tt, lit, pos, space), nil
}

func (l *Lexer) readOperator(pos Position, space bool) (Token, error) {
	ch := l.ch
	two := string(ch) + string(l.peekChar())
	switch two {
	case "==":
		l.readChar()
		l.readChar()
		return l.emit(EQ, "==", pos, space), nil
	case "~=", "!=":
		l.readChar()
		l.readChar()
		return l.emit(NE, two, pos, space), nil
	case "<=":
		l.readChar()
		l.readChar()
		return l.emit(LE, "<=", pos, space), nil
	case ">=":
		l.readChar()
		l.readChar()
		return l.emit(GE, ">=", pos, space), nil
	case "&&":
		l.readChar()
		l.readChar()
		return l.emit(AND, "&&", pos, space), nil
	case "||":
		l.readChar()
		l.readChar()
		return l.emit(OR, "||", pos, space), nil
	case "+=":
		l.readChar()
		l.readChar()
		return l.emit(PLUSEQ, "+=", pos, space), nil
	case "-=":
		l.readChar()
		l.readChar()
		return l.emit(MINUSEQ, "-=", pos, space), nil
	case "*=":
		l.readChar()
		l.readChar()
		return l.emit(STAREQ, "*=", pos, space), nil
	case "/=":
		l.readChar()
		l.readChar()
		return l.emit(SLASHEQ, "/=", pos, space), nil
	case "^=":
		l.readChar()
		l.readChar()
		return l.emit(CARETEQ, "^=", pos, space), nil
	case ".*":
		l.readChar()
		l.readChar()
		return l.emit(DOTSTAR, ".*", pos, space), nil
	case "./":
		l.readChar()
		l.readChar()
		return l.emit(DOTSLASH, "./", pos, space), nil
	case ".^":
		l.readChar()
		l.readChar()
		return l.emit(DOTCARET, ".^", pos, space), nil
	case ".\\":
		l.readChar()
		l.readChar()
		return l.emit(DOTBACKSLASH, ".\\", pos, space), nil
	case ".'":
		l.readChar()
		l.readChar()
		return l.emit(DOTTRANSPOSE, ".'", pos, space), nil
	}
	if ch == '.' && l.peekChar() == '.' && l.peekAt(2) == '.' {
		l.readChar()
		l.readChar()
		l.readChar()
		// consume rest of line as continuation (may include trailing comment)
		for l.ch != '\n' && l.ch != 0 && l.ch != '\r' {
			l.readChar()
		}
		if l.ch == '\r' {
			l.readChar()
		}
		if l.ch == '\n' {
			l.readChar()
		}
		// line continuation is invisible to the parser: recurse for next token,
		// but preserve "preceded by space" from before the ellipsis.
		return l.Next2WithSpace(space)
	}

	switch ch {
	case '+':
		l.readChar()
		return l.emit(PLUS, "+", pos, space), nil
	case '-':
		l.readChar()
		return l.emit(MINUS, "-", pos, space), nil
	case '*':
		l.readChar()
		return l.emit(STAR, "*", pos, space), nil
	case '/':
		l.readChar()
		return l.emit(SLASH, "/", pos, space), nil
	case '\\':
		l.readChar()
		return l.emit(BACKSLASH, "\\", pos, space), nil
	case '^':
		l.readChar()
		return l.emit(CARET, "^", pos, space), nil
	case '=':
		l.readChar()
		return l.emit(ASSIGN, "=", pos, space), nil
	case '<':
		l.readChar()
		return l.emit(LT, "<", pos, space), nil
	case '>':
		l.readChar()
		return l.emit(GT, ">", pos, space), nil
	case '&':
		l.readChar()
		return l.emit(BITAND, "&", pos, space), nil
	case '|':
		l.readChar()
		return l.emit(BITOR, "|", pos, space), nil
	case '~':
		l.readChar()
		return l.emit(NOT, "~", pos, space), nil
	case '!':
		l.readChar()
		return l.emit(NOT, "!", pos, space), nil
	case ':':
		l.readChar()
		return l.emit(COLON, ":", pos, space), nil
	case ';':
		l.readChar()
		return l.emit(SEMICOLON, ";", pos, space), nil
	case ',':
		l.readChar()
		return l.emit(COMMA, ",", pos, space), nil
	case '.':
		l.readChar()
		return l.emit(DOT, ".", pos, space), nil
	case '@':
		l.readChar()
		return l.emit(AT, "@", pos, space), nil
	case '(':
		l.readChar()
		return l.emit(LPAREN, "(", pos, space), nil
	case ')':
		l.readChar()
		return l.emit(RPAREN, ")", pos, space), nil
	case '[':
		l.readChar()
		return l.emit(LBRACKET, "[", pos, space), nil
	case ']':
		l.readChar()
		return l.emit(RBRACKET, "]", pos, space), nil
	case '{':
		l.readChar()
		return l.emit(LBRACE, "{", pos, space), nil
	case '}':
		l.readChar()
		return l.emit(RBRACE, "}", pos, space), nil
	}
	l.readChar()
	return l.emit(ILLEGAL, string(ch), pos, space), nil
}

// Next2WithSpace continues tokenizing after a swallowed "..." continuation,
// forcing the PrecededBySpace flag of the following token to reflect that a
// continuation (which always acts like whitespace) preceded it.
func (l *Lexer) Next2WithSpace(space bool) (Token, error) {
	l.skipSpaceTrackingSpace()
	tok, err := l.Next()
	if err != nil {
		return tok, err
	}
	tok.PrecededBySpace = true
	return tok, nil
}

func isDigit(ch rune) bool { return ch >= '0' && ch <= '9' }
func isHexDigit(ch rune) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
func isIdentStart(ch rune) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch)
}
