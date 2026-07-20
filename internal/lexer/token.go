package lexer

// TokenType identifies the category of a lexical token.
type TokenType int

const (
	EOF TokenType = iota
	ILLEGAL
	NEWLINE

	IDENT
	NUMBER
	STRING // single- or double-quoted string literal, Value includes resolved content

	COMMENT      // line comment, Value is the comment text including % or #
	BLOCKCOMMENT // %{ ... %} or #{ ... #}, Value is the full block including delimiters

	ELLIPSIS       // ... line continuation
	BACKSLASH_CONT // line continuation using trailing backslash (rare, not std) - unused placeholder

	// operators
	PLUS
	MINUS
	STAR
	SLASH
	BACKSLASH
	CARET
	DOTSTAR
	DOTSLASH
	DOTBACKSLASH
	DOTCARET
	TRANSPOSE    // '
	DOTTRANSPOSE // .'

	ASSIGN  // =
	PLUSEQ  // +=
	MINUSEQ // -=
	STAREQ  // *=
	SLASHEQ // /=
	CARETEQ // ^=

	EQ // ==
	NE // ~= or !=
	LT // <
	GT // >
	LE // <=
	GE // >=

	AND    // &&
	OR     // ||
	BITAND // &
	BITOR  // |
	NOT    // ~ or !

	COLON     // :
	SEMICOLON // ;
	COMMA     // ,
	DOT       // .
	AT        // @

	LPAREN
	RPAREN
	LBRACKET
	RBRACKET
	LBRACE
	RBRACE

	// keywords
	IF
	ELSEIF
	ELSE
	ENDIF
	END
	FOR
	ENDFOR
	PARFOR
	ENDPARFOR
	WHILE
	ENDWHILE
	DO
	UNTIL
	SWITCH
	CASE
	OTHERWISE
	ENDSWITCH
	FUNCTION
	ENDFUNCTION
	RETURN
	BREAK
	CONTINUE
	GLOBAL
	PERSISTENT
	TRY
	CATCH
	END_TRY_CATCH
	UNWIND_PROTECT
	UNWIND_PROTECT_CLEANUP
	END_UNWIND_PROTECT
)

var keywords = map[string]TokenType{
	"if":                     IF,
	"elseif":                 ELSEIF,
	"else":                   ELSE,
	"endif":                  ENDIF,
	"end":                    END,
	"for":                    FOR,
	"endfor":                 ENDFOR,
	"parfor":                 PARFOR,
	"endparfor":              ENDPARFOR,
	"while":                  WHILE,
	"endwhile":               ENDWHILE,
	"do":                     DO,
	"until":                  UNTIL,
	"switch":                 SWITCH,
	"case":                   CASE,
	"otherwise":              OTHERWISE,
	"endswitch":              ENDSWITCH,
	"function":               FUNCTION,
	"endfunction":            ENDFUNCTION,
	"return":                 RETURN,
	"break":                  BREAK,
	"continue":               CONTINUE,
	"global":                 GLOBAL,
	"persistent":             PERSISTENT,
	"try":                    TRY,
	"catch":                  CATCH,
	"end_try_catch":          END_TRY_CATCH,
	"unwind_protect":         UNWIND_PROTECT,
	"unwind_protect_cleanup": UNWIND_PROTECT_CLEANUP,
	"end_unwind_protect":     END_UNWIND_PROTECT,
}

// blockEnders is the set of keywords that close a block started by one of
// if/for/while/switch/function/try/unwind_protect/do. The generic "end" also
// closes any of them.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

func IsBlockEnd(t TokenType) bool {
	switch t {
	case END, ENDIF, ENDFOR, ENDPARFOR, ENDWHILE, ENDSWITCH, ENDFUNCTION,
		END_TRY_CATCH, END_UNWIND_PROTECT:
		return true
	}
	return false
}

// Position records where a token was found in the source.
type Position struct {
	Line   int
	Column int
	Offset int
}

// Token is a single lexical token with its source text and position.
type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
	// PrecededBySpace records whether whitespace (not newline) appeared
	// immediately before this token. Used by the parser to disambiguate
	// matrix-row element separation and unary vs binary +/-.
	PrecededBySpace bool
	// PrecededByNewline records whether this is the first token on its line
	// (ignoring the NEWLINE token itself).
}
