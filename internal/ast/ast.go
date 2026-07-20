// Package ast defines the abstract syntax tree for Octave source produced
// by the parser and consumed by the printer.
package ast

import "github.com/matteo/octave-formatter/internal/lexer"

// Comment is a single line or block comment.
type Comment struct {
	Text  string // raw text including leading % or # (and for block, the fences)
	Block bool
}

// Node is implemented by all AST nodes.
type Node interface {
	node()
}

// Stmt is implemented by all statement nodes.
type Stmt interface {
	Node
	stmtNode()
	Base() *StmtBase
}

// Expr is implemented by all expression nodes.
type Expr interface {
	Node
	exprNode()
}

// StmtBase carries formatting metadata common to every statement:
// how many blank lines preceded it in the source, and any comment trailing
// it on the same line.
type StmtBase struct {
	BlankLinesBefore int
	TrailingComment  *Comment
	Semicolon        bool // statement suppresses output with a trailing ';'
}

func (b *StmtBase) Base() *StmtBase { return b }

// File is the root node: a sequence of top-level statements. Standalone
// comments appear as CommentStmt entries within Stmts.
type File struct {
	Stmts              []Stmt
	TrailingBlankLines int // blank lines after the final statement
}

func (*File) node() {}

// ---- Statements ----

type ExprStmt struct {
	StmtBase
	X Expr
}

type AssignStmt struct {
	StmtBase
	Lhs Expr
	Op  lexer.TokenType // ASSIGN, PLUSEQ, MINUSEQ, STAREQ, SLASHEQ, CARETEQ
	Rhs Expr
}

// MultiAssignStmt handles `[a, b] = f(x)` and `[a, ~, c] = f(x)`.
type MultiAssignStmt struct {
	StmtBase
	Targets []Expr // Ident, FieldExpr, IndexExpr, or nil-ish TildeExpr placeholder
	Rhs     Expr
}

type ElseIfClause struct {
	Cond             Expr
	Body             []Stmt
	BlankLinesBefore int
	TrailingComment  *Comment // comment after "elseif <cond>"
}

type IfStmt struct {
	StmtBase
	Cond           Expr
	CondComment    *Comment // comment trailing "if <cond>"
	Body           []Stmt
	ElseIfs        []ElseIfClause
	HasElse        bool
	Else           []Stmt
	ElseBlankLines int
	ElseComment    *Comment
}

type ForStmt struct {
	StmtBase
	Parfor        bool
	Var           Expr
	Range         Expr
	MaxProc       Expr // optional parfor(...,N) argument
	HeaderComment *Comment
	Body          []Stmt
}

type WhileStmt struct {
	StmtBase
	Cond          Expr
	HeaderComment *Comment
	Body          []Stmt
}

type DoUntilStmt struct {
	StmtBase
	Body          []Stmt
	Cond          Expr
	HeaderComment *Comment
}

type CaseClause struct {
	Expr             Expr
	Body             []Stmt
	BlankLinesBefore int
	TrailingComment  *Comment
}

type SwitchStmt struct {
	StmtBase
	Tag                 Expr
	HeaderComment       *Comment
	Cases               []CaseClause
	HasOtherwise        bool
	Otherwise           []Stmt
	OtherwiseBlankLines int
	OtherwiseComment    *Comment
}

type FunctionStmt struct {
	StmtBase
	Outputs       []Expr // Ident list (possibly empty, possibly one without brackets)
	Name          string
	Params        []Expr // Ident list
	HeaderComment *Comment
	Body          []Stmt
	HasEnd        bool // whether source used endfunction/end explicitly (script-style functions may omit it)
}

type ReturnStmt struct{ StmtBase }
type BreakStmt struct{ StmtBase }
type ContinueStmt struct{ StmtBase }

// GlobalVar is one `name` or `name = init` entry in a global/persistent
// declaration.
type GlobalVar struct {
	Name string
	Init Expr // optional
}

type GlobalStmt struct {
	StmtBase
	Names []GlobalVar
}

type PersistentStmt struct {
	StmtBase
	Names []GlobalVar
}

type TryStmt struct {
	StmtBase
	Body            []Stmt
	HasCatch        bool
	CatchVar        Expr // optional
	CatchComment    *Comment
	CatchBody       []Stmt
	CatchBlankLines int
}

type UnwindProtectStmt struct {
	StmtBase
	HeaderComment     *Comment
	Body              []Stmt
	HasCleanup        bool
	Cleanup           []Stmt
	CleanupBlankLines int
}

// CommandStmt models Octave "command syntax", e.g. `hold on`, `format long g`,
// `pkg load signal`. Args are kept as raw source words.
type CommandStmt struct {
	StmtBase
	Name string
	Args []string
}

// CommentStmt is a standalone comment occupying its own statement position
// (used for comments that appear inside a block, not just before a statement,
// e.g. a comment as the very last thing in a function body).
type CommentStmt struct {
	StmtBase
	C Comment
}

func (*ExprStmt) stmtNode()          {}
func (*AssignStmt) stmtNode()        {}
func (*MultiAssignStmt) stmtNode()   {}
func (*IfStmt) stmtNode()            {}
func (*ForStmt) stmtNode()           {}
func (*WhileStmt) stmtNode()         {}
func (*DoUntilStmt) stmtNode()       {}
func (*SwitchStmt) stmtNode()        {}
func (*FunctionStmt) stmtNode()      {}
func (*ReturnStmt) stmtNode()        {}
func (*BreakStmt) stmtNode()         {}
func (*ContinueStmt) stmtNode()      {}
func (*GlobalStmt) stmtNode()        {}
func (*PersistentStmt) stmtNode()    {}
func (*TryStmt) stmtNode()           {}
func (*UnwindProtectStmt) stmtNode() {}
func (*CommandStmt) stmtNode()       {}
func (*CommentStmt) stmtNode()       {}

func (*ExprStmt) node()          {}
func (*AssignStmt) node()        {}
func (*MultiAssignStmt) node()   {}
func (*IfStmt) node()            {}
func (*ForStmt) node()           {}
func (*WhileStmt) node()         {}
func (*DoUntilStmt) node()       {}
func (*SwitchStmt) node()        {}
func (*FunctionStmt) node()      {}
func (*ReturnStmt) node()        {}
func (*BreakStmt) node()         {}
func (*ContinueStmt) node()      {}
func (*GlobalStmt) node()        {}
func (*PersistentStmt) node()    {}
func (*TryStmt) node()           {}
func (*UnwindProtectStmt) node() {}
func (*CommandStmt) node()       {}
func (*CommentStmt) node()       {}

// ---- Expressions ----

type Ident struct{ Name string }
type NumberLit struct{ Value string }
type StringLit struct {
	Value string // raw literal including quotes
	Quote byte   // '\'' or '"'
}
type EndExpr struct{}   // bare `end` used as "last index" inside indexing
type ColonExpr struct{} // bare `:` used as "all elements along dimension"
type TildeExpr struct{} // `~` used as ignored output in multi-assign targets

type RangeExpr struct {
	Start Expr
	Step  Expr // optional
	Stop  Expr
}

type UnaryExpr struct {
	Op      lexer.TokenType
	X       Expr
	Postfix bool // true for postfix ' and .'
}

type BinaryExpr struct {
	Op   lexer.TokenType
	X, Y Expr
}

type ParenExpr struct{ X Expr }

// MatrixExpr is a `[ ... ]` literal; Rows holds each semicolon/newline
// separated row.
type MatrixExpr struct{ Rows [][]Expr }

// CellExpr is a `{ ... }` literal.
type CellExpr struct{ Rows [][]Expr }

// IndexExpr covers both function calls and array indexing: `f(x, y)`.
type IndexExpr struct {
	Fn   Expr
	Args []Expr
}

// CellIndexExpr covers `a{i, j}` cell-content indexing.
type CellIndexExpr struct {
	Fn   Expr
	Args []Expr
}

type FieldExpr struct {
	X    Expr
	Name string
}

// DynFieldExpr covers `a.(expr)`.
type DynFieldExpr struct {
	X    Expr
	Name Expr
}

// AnonFuncExpr covers `@(x, y) x + y`.
type AnonFuncExpr struct {
	Params []Expr
	Body   Expr
}

// FuncHandleExpr covers `@name`.
type FuncHandleExpr struct{ Name string }

func (*Ident) exprNode()          {}
func (*NumberLit) exprNode()      {}
func (*StringLit) exprNode()      {}
func (*EndExpr) exprNode()        {}
func (*ColonExpr) exprNode()      {}
func (*TildeExpr) exprNode()      {}
func (*RangeExpr) exprNode()      {}
func (*UnaryExpr) exprNode()      {}
func (*BinaryExpr) exprNode()     {}
func (*ParenExpr) exprNode()      {}
func (*MatrixExpr) exprNode()     {}
func (*CellExpr) exprNode()       {}
func (*IndexExpr) exprNode()      {}
func (*CellIndexExpr) exprNode()  {}
func (*FieldExpr) exprNode()      {}
func (*DynFieldExpr) exprNode()   {}
func (*AnonFuncExpr) exprNode()   {}
func (*FuncHandleExpr) exprNode() {}

func (*Ident) node()          {}
func (*NumberLit) node()      {}
func (*StringLit) node()      {}
func (*EndExpr) node()        {}
func (*ColonExpr) node()      {}
func (*TildeExpr) node()      {}
func (*RangeExpr) node()      {}
func (*UnaryExpr) node()      {}
func (*BinaryExpr) node()     {}
func (*ParenExpr) node()      {}
func (*MatrixExpr) node()     {}
func (*CellExpr) node()       {}
func (*IndexExpr) node()      {}
func (*CellIndexExpr) node()  {}
func (*FieldExpr) node()      {}
func (*DynFieldExpr) node()   {}
func (*AnonFuncExpr) node()   {}
func (*FuncHandleExpr) node() {}
