// Package printer renders an ast.File back into formatted Octave source.
package printer

import (
	"strings"

	"github.com/matteo/octave-formatter/internal/ast"
)

// Options controls formatting style.
type Options struct {
	IndentWidth   int // spaces per indent level
	MaxBlankLines int // max consecutive blank lines kept between statements
}

// DefaultOptions is the formatter's default style.
var DefaultOptions = Options{IndentWidth: 2, MaxBlankLines: 1}

type Printer struct {
	opts Options
	sb   strings.Builder
	ind  int
}

func New(opts Options) *Printer {
	return &Printer{opts: opts}
}

// Print renders f and returns the formatted source, always ending in
// exactly one trailing newline.
func Print(f *ast.File, opts Options) string {
	p := New(opts)
	p.printStmtList(f.Stmts)
	out := strings.TrimRight(p.sb.String(), "\n")
	if out == "" {
		return ""
	}
	return out + "\n"
}

func (p *Printer) indentPrefix() string {
	return strings.Repeat(" ", p.ind*p.opts.IndentWidth)
}

func (p *Printer) writeIndent()   { p.sb.WriteString(p.indentPrefix()) }
func (p *Printer) write(s string) { p.sb.WriteString(s) }
func (p *Printer) newline()       { p.sb.WriteByte('\n') }
func (p *Printer) blankLine()     { p.sb.WriteByte('\n') }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// printStmtList prints a sequence of statements, honoring blank-line
// separation (capped) between them but never before the first statement in
// the list.
func (p *Printer) printStmtList(stmts []ast.Stmt) {
	for i, s := range stmts {
		if i > 0 {
			n := min(s.Base().BlankLinesBefore, p.opts.MaxBlankLines)
			for j := 0; j < n; j++ {
				p.blankLine()
			}
		}
		p.printStmt(s)
	}
}

func (p *Printer) printBlock(stmts []ast.Stmt) {
	p.ind++
	p.printStmtList(stmts)
	p.ind--
}

// finishSimple writes the optional semicolon and trailing comment that end
// a simple (single-line) statement, then the newline.
func (p *Printer) finishSimple(base *ast.StmtBase) {
	if base.Semicolon {
		p.write(";")
	}
	p.writeTrailingComment(base.TrailingComment)
	p.newline()
}

func (p *Printer) writeTrailingComment(c *ast.Comment) {
	if c == nil {
		return
	}
	p.write("  " + normalizeComment(c.Text))
}

func (p *Printer) printCommentLines(c ast.Comment) {
	text := c.Text
	if c.Block {
		text = strings.TrimRight(text, "\n")
		lines := strings.Split(text, "\n")
		p.writeIndent()
		p.write(lines[0])
		p.newline()
		for _, ln := range lines[1:] {
			p.write(ln)
			p.newline()
		}
		return
	}
	p.writeIndent()
	p.write(normalizeComment(text))
	p.newline()
}

// normalizeComment ensures exactly one space between the comment marker and
// its text, while leaving special double-marker section dividers (%%, ##)
// and empty markers untouched.
func normalizeComment(text string) string {
	if text == "" {
		return text
	}
	marker := text[0]
	if marker != '%' && marker != '#' {
		return text
	}
	rest := text[1:]
	if strings.HasPrefix(rest, string(marker)) {
		return strings.TrimRight(text, " \t")
	}
	trimmed := strings.TrimLeft(rest, " \t")
	trimmed = strings.TrimRight(trimmed, " \t")
	if trimmed == "" {
		return string(marker)
	}
	return string(marker) + " " + trimmed
}
