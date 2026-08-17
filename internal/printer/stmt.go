package printer

import (
	"github.com/matteo/octfmt/internal/ast"
)

func (p *Printer) printStmt(s ast.Stmt) {
	switch v := s.(type) {
	case *ast.CommentStmt:
		p.printCommentLines(v.C)
	case *ast.ExprStmt:
		p.writeIndent()
		p.write(p.exprStr(v.X))
		p.finishSimple(&v.StmtBase)
	case *ast.AssignStmt:
		p.writeIndent()
		p.write(p.exprStr(v.Lhs))
		p.write(" " + paddedOpText[v.Op] + " ")
		p.write(p.exprStr(v.Rhs))
		p.finishSimple(&v.StmtBase)
	case *ast.MultiAssignStmt:
		p.writeIndent()
		p.write("[" + p.argListStr(v.Targets, p.ind) + "] = ")
		p.write(p.exprStr(v.Rhs))
		p.finishSimple(&v.StmtBase)
	case *ast.CommandStmt:
		p.writeIndent()
		p.write(v.Name)
		for _, a := range v.Args {
			p.write(" " + a)
		}
		p.finishSimple(&v.StmtBase)
	case *ast.ReturnStmt:
		p.writeIndent()
		p.write("return")
		p.finishSimple(&v.StmtBase)
	case *ast.BreakStmt:
		p.writeIndent()
		p.write("break")
		p.finishSimple(&v.StmtBase)
	case *ast.ContinueStmt:
		p.writeIndent()
		p.write("continue")
		p.finishSimple(&v.StmtBase)
	case *ast.GlobalStmt:
		p.writeIndent()
		p.write("global" + p.globalVarListStr(v.Names))
		p.finishSimple(&v.StmtBase)
	case *ast.PersistentStmt:
		p.writeIndent()
		p.write("persistent" + p.globalVarListStr(v.Names))
		p.finishSimple(&v.StmtBase)
	case *ast.IfStmt:
		p.printIf(v)
	case *ast.ForStmt:
		p.printFor(v)
	case *ast.WhileStmt:
		p.printWhile(v)
	case *ast.DoUntilStmt:
		p.printDoUntil(v)
	case *ast.SwitchStmt:
		p.printSwitch(v)
	case *ast.FunctionStmt:
		p.printFunction(v)
	case *ast.TryStmt:
		p.printTry(v)
	case *ast.UnwindProtectStmt:
		p.printUnwindProtect(v)
	}
}

func (p *Printer) globalVarListStr(names []ast.GlobalVar) string {
	s := ""
	for _, n := range names {
		s += " " + n.Name
		if n.Init != nil {
			s += " = " + p.exprStr(n.Init)
		}
	}
	return s
}

func (p *Printer) printIf(v *ast.IfStmt) {
	p.writeIndent()
	p.write("if " + p.exprStr(v.Cond))
	p.writeTrailingComment(v.CondComment)
	p.newline()
	p.printBlock(v.Body)

	for _, ei := range v.ElseIfs {
		p.writeIndent()
		p.write("elseif " + p.exprStr(ei.Cond))
		p.writeTrailingComment(ei.TrailingComment)
		p.newline()
		p.printBlock(ei.Body)
	}

	if v.HasElse {
		p.writeIndent()
		p.write("else")
		p.writeTrailingComment(v.ElseComment)
		p.newline()
		p.printBlock(v.Else)
	}

	p.writeIndent()
	p.write("end")
	p.finishSimple(&v.StmtBase)
}

func (p *Printer) printFor(v *ast.ForStmt) {
	p.writeIndent()
	if v.Parfor {
		p.write("parfor ")
	} else {
		p.write("for ")
	}
	p.write(p.exprStr(v.Var) + " = " + p.exprStr(v.Range))
	if v.MaxProc != nil {
		p.write(", " + p.exprStr(v.MaxProc))
	}
	p.writeTrailingComment(v.HeaderComment)
	p.newline()
	p.printBlock(v.Body)
	p.writeIndent()
	p.write("end")
	p.finishSimple(&v.StmtBase)
}

func (p *Printer) printWhile(v *ast.WhileStmt) {
	p.writeIndent()
	p.write("while " + p.exprStr(v.Cond))
	p.writeTrailingComment(v.HeaderComment)
	p.newline()
	p.printBlock(v.Body)
	p.writeIndent()
	p.write("end")
	p.finishSimple(&v.StmtBase)
}

func (p *Printer) printDoUntil(v *ast.DoUntilStmt) {
	p.writeIndent()
	p.write("do")
	p.writeTrailingComment(v.HeaderComment)
	p.newline()
	p.printBlock(v.Body)
	p.writeIndent()
	p.write("until " + p.exprStr(v.Cond))
	p.finishSimple(&v.StmtBase)
}

func (p *Printer) printSwitch(v *ast.SwitchStmt) {
	p.writeIndent()
	p.write("switch " + p.exprStr(v.Tag))
	p.writeTrailingComment(v.HeaderComment)
	p.newline()

	p.ind++
	for i, c := range v.Cases {
		if i > 0 {
			n := min(c.BlankLinesBefore, p.opts.MaxBlankLines)
			for j := 0; j < n; j++ {
				p.blankLine()
			}
		}
		p.writeIndent()
		p.write("case " + p.exprStr(c.Expr))
		p.writeTrailingComment(c.TrailingComment)
		p.newline()
		p.printBlock(c.Body)
	}

	if v.HasOtherwise {
		if len(v.Cases) > 0 {
			n := min(v.OtherwiseBlankLines, p.opts.MaxBlankLines)
			for j := 0; j < n; j++ {
				p.blankLine()
			}
		}
		p.writeIndent()
		p.write("otherwise")
		p.writeTrailingComment(v.OtherwiseComment)
		p.newline()
		p.printBlock(v.Otherwise)
	}
	p.ind--

	p.writeIndent()
	p.write("end")
	p.finishSimple(&v.StmtBase)
}

func (p *Printer) printFunction(v *ast.FunctionStmt) {
	p.writeIndent()
	p.write("function ")
	switch len(v.Outputs) {
	case 0:
	case 1:
		p.write(p.exprStr(v.Outputs[0]) + " = ")
	default:
		p.write("[" + p.argListStr(v.Outputs, p.ind) + "] = ")
	}
	p.write(v.Name)
	p.write("(" + p.argListStr(v.Params, p.ind) + ")")
	p.writeTrailingComment(v.HeaderComment)
	p.newline()
	p.printBlock(v.Body)
	if v.HasEnd {
		p.writeIndent()
		p.write("end")
		p.finishSimple(&v.StmtBase)
	}
}

func (p *Printer) printTry(v *ast.TryStmt) {
	p.writeIndent()
	p.write("try")
	p.newline()
	p.printBlock(v.Body)
	if v.HasCatch {
		p.writeIndent()
		p.write("catch")
		if id, ok := v.CatchVar.(*ast.Ident); ok {
			p.write(" " + id.Name)
		}
		p.writeTrailingComment(v.CatchComment)
		p.newline()
		p.printBlock(v.CatchBody)
	}
	p.writeIndent()
	p.write("end")
	p.finishSimple(&v.StmtBase)
}

func (p *Printer) printUnwindProtect(v *ast.UnwindProtectStmt) {
	p.writeIndent()
	p.write("unwind_protect")
	p.writeTrailingComment(v.HeaderComment)
	p.newline()
	p.printBlock(v.Body)
	if v.HasCleanup {
		p.writeIndent()
		p.write("unwind_protect_cleanup")
		p.newline()
		p.printBlock(v.Cleanup)
	}
	p.writeIndent()
	p.write("end")
	p.finishSimple(&v.StmtBase)
}
