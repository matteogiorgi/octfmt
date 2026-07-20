package printer

import (
	"strings"

	"github.com/matteo/octave-formatter/internal/ast"
	"github.com/matteo/octave-formatter/internal/lexer"
)

// paddedOpText maps binary operator tokens that are printed with a
// surrounding space on each side to their canonical text. Operators not
// listed here (^, .^) are printed tight with no surrounding space.
var paddedOpText = map[lexer.TokenType]string{
	lexer.OR:           "||",
	lexer.AND:          "&&",
	lexer.BITOR:        "|",
	lexer.BITAND:       "&",
	lexer.EQ:           "==",
	lexer.NE:           "~=",
	lexer.LT:           "<",
	lexer.GT:           ">",
	lexer.LE:           "<=",
	lexer.GE:           ">=",
	lexer.PLUS:         "+",
	lexer.MINUS:        "-",
	lexer.STAR:         "*",
	lexer.SLASH:        "/",
	lexer.BACKSLASH:    "\\",
	lexer.DOTSTAR:      ".*",
	lexer.DOTSLASH:     "./",
	lexer.DOTBACKSLASH: ".\\",
	lexer.ASSIGN:       "=",
	lexer.PLUSEQ:       "+=",
	lexer.MINUSEQ:      "-=",
	lexer.STAREQ:       "*=",
	lexer.SLASHEQ:      "/=",
	lexer.CARETEQ:      "^=",
}

var tightOpText = map[lexer.TokenType]string{
	lexer.CARET:    "^",
	lexer.DOTCARET: ".^",
}

var unaryOpText = map[lexer.TokenType]string{
	lexer.PLUS:  "+",
	lexer.MINUS: "-",
	lexer.NOT:   "~",
}

func (p *Printer) exprStr(e ast.Expr) string {
	return p.exprStrIndent(e, p.ind)
}

// exprStrIndent renders e as it would appear starting at indent level ind
// (used so nested multi-row matrix/cell literals align correctly even when
// they appear inside a call argument rather than directly in a statement).
func (p *Printer) exprStrIndent(e ast.Expr, ind int) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.NumberLit:
		return v.Value
	case *ast.StringLit:
		return v.Value
	case *ast.EndExpr:
		return "end"
	case *ast.ColonExpr:
		return ":"
	case *ast.TildeExpr:
		return "~"
	case *ast.RangeExpr:
		if v.Step != nil {
			return p.exprStrIndent(v.Start, ind) + ":" + p.exprStrIndent(v.Step, ind) + ":" + p.exprStrIndent(v.Stop, ind)
		}
		return p.exprStrIndent(v.Start, ind) + ":" + p.exprStrIndent(v.Stop, ind)
	case *ast.UnaryExpr:
		x := p.exprStrIndent(v.X, ind)
		if v.Postfix {
			if v.Op == lexer.DOTTRANSPOSE {
				return x + ".'"
			}
			return x + "'"
		}
		return unaryOpText[v.Op] + x
	case *ast.BinaryExpr:
		x := p.exprStrIndent(v.X, ind)
		y := p.exprStrIndent(v.Y, ind)
		if txt, ok := tightOpText[v.Op]; ok {
			return x + txt + y
		}
		return x + " " + paddedOpText[v.Op] + " " + y
	case *ast.ParenExpr:
		return "(" + p.exprStrIndent(v.X, ind) + ")"
	case *ast.MatrixExpr:
		return p.matrixStr(v.Rows, "[", "]", ind)
	case *ast.CellExpr:
		return p.matrixStr(v.Rows, "{", "}", ind)
	case *ast.IndexExpr:
		return p.exprStrIndent(v.Fn, ind) + "(" + p.argListStr(v.Args, ind) + ")"
	case *ast.CellIndexExpr:
		return p.exprStrIndent(v.Fn, ind) + "{" + p.argListStr(v.Args, ind) + "}"
	case *ast.FieldExpr:
		return p.exprStrIndent(v.X, ind) + "." + v.Name
	case *ast.DynFieldExpr:
		return p.exprStrIndent(v.X, ind) + ".(" + p.exprStrIndent(v.Name, ind) + ")"
	case *ast.AnonFuncExpr:
		return "@(" + identListStr(v.Params) + ") " + p.exprStrIndent(v.Body, ind)
	case *ast.FuncHandleExpr:
		return "@" + v.Name
	default:
		return ""
	}
}

func (p *Printer) argListStr(args []ast.Expr, ind int) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = p.exprStrIndent(a, ind)
	}
	return strings.Join(parts, ", ")
}

func identListStr(idents []ast.Expr) string {
	parts := make([]string, len(idents))
	for i, id := range idents {
		if t, ok := id.(*ast.TildeExpr); ok {
			_ = t
			parts[i] = "~"
			continue
		}
		if n, ok := id.(*ast.Ident); ok {
			parts[i] = n.Name
		}
	}
	return strings.Join(parts, ", ")
}

// matrixStr renders a `[...]`/`{...}` literal. A single row is kept on one
// line; multiple rows are broken one-per-line, indented one level deeper
// than ind, matching how the literal would be re-indented if reformatted
// wherever it appears.
func (p *Printer) matrixStr(rows [][]ast.Expr, open, close string, ind int) string {
	if len(rows) <= 1 {
		var row []ast.Expr
		if len(rows) == 1 {
			row = rows[0]
		}
		return open + p.argListStr(row, ind) + close
	}
	innerPrefix := strings.Repeat(" ", (ind+1)*p.opts.IndentWidth)
	outerPrefix := strings.Repeat(" ", ind*p.opts.IndentWidth)
	var sb strings.Builder
	sb.WriteString(open)
	sb.WriteByte('\n')
	for _, row := range rows {
		sb.WriteString(innerPrefix)
		sb.WriteString(p.argListStr(row, ind+1))
		sb.WriteByte('\n')
	}
	sb.WriteString(outerPrefix)
	sb.WriteString(close)
	return sb.String()
}
