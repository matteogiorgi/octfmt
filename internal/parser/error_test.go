package parser

import "testing"

func TestParseErrorsDoNotPanic(t *testing.T) {
	cases := []string{
		"if x\n",             // missing end
		"for i = 1:10\n",     // missing end
		"x = [1, 2;\n",       // unterminated matrix
		"function y = f(x\n", // unterminated params
		"a = 'unterminated",  // unterminated string
		"1 + + + ;",
		"switch\ncase\nend",
	}
	for _, src := range cases {
		f, errs := Parse(src)
		if f == nil && len(errs) == 0 {
			t.Errorf("%q: expected either a file or errors", src)
		}
		if len(errs) == 0 {
			t.Errorf("%q: expected parse errors, got none", src)
		}
	}
}
