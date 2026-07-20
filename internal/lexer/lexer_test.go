package lexer

import "testing"

func typesOf(t *testing.T, src string) []TokenType {
	t.Helper()
	l := New(src)
	toks, err := l.Tokenize()
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	var types []TokenType
	for _, tok := range toks {
		if tok.Type == NEWLINE {
			continue
		}
		types = append(types, tok.Type)
	}
	return types
}

func TestTransposeVsString(t *testing.T) {
	// no space before ' after an identifier => transpose
	types := typesOf(t, "a'")
	want := []TokenType{IDENT, TRANSPOSE, EOF}
	assertTypes(t, types, want)

	// space before ' => string literal start
	types = typesOf(t, "a 'hello'")
	want = []TokenType{IDENT, STRING, EOF}
	assertTypes(t, types, want)

	// ' at start of expression is always a string
	types = typesOf(t, "'hello'")
	want = []TokenType{STRING, EOF}
	assertTypes(t, types, want)

	// after closing paren, tight ' is transpose
	types = typesOf(t, "f(x)'")
	want = []TokenType{IDENT, LPAREN, IDENT, RPAREN, TRANSPOSE, EOF}
	assertTypes(t, types, want)
}

func TestNumbers(t *testing.T) {
	cases := []string{"1", "1.5", "1e10", "1.5e-3", "0x1F", "0b101", "2i", "3.5j"}
	for _, c := range cases {
		toks, err := New(c).Tokenize()
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		if toks[0].Type != NUMBER || toks[0].Literal != c {
			t.Errorf("%s: got %v %q", c, toks[0].Type, toks[0].Literal)
		}
	}
}

func TestDotOperatorsNotDecimal(t *testing.T) {
	types := typesOf(t, "a.*b")
	assertTypes(t, types, []TokenType{IDENT, DOTSTAR, IDENT, EOF})

	types = typesOf(t, "1.5")
	assertTypes(t, types, []TokenType{NUMBER, EOF})
}

func TestComments(t *testing.T) {
	toks, err := New("x = 1; % trailing\n% standalone\ny = 2;").Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	var comments []string
	for _, tok := range toks {
		if tok.Type == COMMENT {
			comments = append(comments, tok.Literal)
		}
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d: %v", len(comments), comments)
	}
}

func TestBlockComment(t *testing.T) {
	src := "%{\nblock\nbody\n%}\nx = 1;"
	toks, err := New(src).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if toks[0].Type != BLOCKCOMMENT {
		t.Fatalf("expected BLOCKCOMMENT, got %v", toks[0].Type)
	}
}

func TestLineContinuation(t *testing.T) {
	types := typesOf(t, "a = 1 + ...\n  2;")
	assertTypes(t, types, []TokenType{IDENT, ASSIGN, NUMBER, PLUS, NUMBER, SEMICOLON, EOF})
}

func assertTypes(t *testing.T, got, want []TokenType) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %v want %v (full got=%v)", i, got[i], want[i], got)
		}
	}
}
