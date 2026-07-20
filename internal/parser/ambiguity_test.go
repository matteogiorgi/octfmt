package parser

import (
	"github.com/matteo/octave-formatter/internal/ast"
	"testing"
)

func TestMatrixSpacing(t *testing.T) {
	f, errs := Parse("a = [1 -1 2, 3; 4 5 6]';\n")
	for _, e := range errs {
		t.Fatal(e)
	}
	as := f.Stmts[0].(*ast.AssignStmt)
	un := as.Rhs.(*ast.UnaryExpr)
	if !un.Postfix {
		t.Fatal("expected postfix transpose")
	}
	mat := un.X.(*ast.MatrixExpr)
	if len(mat.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(mat.Rows))
	}
	if len(mat.Rows[0]) != 4 {
		t.Fatalf("expected 4 elements in row0, got %d", len(mat.Rows[0]))
	}
	if len(mat.Rows[1]) != 3 {
		t.Fatalf("expected 3 elements in row1, got %d", len(mat.Rows[1]))
	}
	if _, ok := mat.Rows[0][1].(*ast.UnaryExpr); !ok {
		t.Fatalf("expected element1 to be unary minus, got %T", mat.Rows[0][1])
	}
}

func TestBinaryVsUnaryInMatrix(t *testing.T) {
	f, errs := Parse("a = [1 - 1, 2-2, 3 -3];\n")
	for _, e := range errs {
		t.Fatal(e)
	}
	as := f.Stmts[0].(*ast.AssignStmt)
	mat := as.Rhs.(*ast.MatrixExpr)
	if len(mat.Rows[0]) != 4 {
		t.Fatalf("expected 4 elements (binary,binary,unary-split), got %d: %#v", len(mat.Rows[0]), mat.Rows[0])
	}
	if _, ok := mat.Rows[0][0].(*ast.BinaryExpr); !ok {
		t.Fatalf("elem0 'a - a' should be binary, got %T", mat.Rows[0][0])
	}
	if _, ok := mat.Rows[0][1].(*ast.BinaryExpr); !ok {
		t.Fatalf("elem1 '2-2' should be binary, got %T", mat.Rows[0][1])
	}
	if _, ok := mat.Rows[0][2].(*ast.NumberLit); !ok {
		t.Fatalf("elem2 should be plain 3, got %T", mat.Rows[0][2])
	}
	if _, ok := mat.Rows[0][3].(*ast.UnaryExpr); !ok {
		t.Fatalf("elem3 should be unary -3, got %T", mat.Rows[0][3])
	}
}

func TestCommandSyntax(t *testing.T) {
	f, errs := Parse("hold on\nformat long g\ndisp hello\nx = 1;\n")
	for _, e := range errs {
		t.Fatal(e)
	}
	if len(f.Stmts) != 4 {
		t.Fatalf("expected 4 stmts, got %d", len(f.Stmts))
	}
	if _, ok := f.Stmts[0].(*ast.CommandStmt); !ok {
		t.Fatalf("stmt0 should be CommandStmt, got %T", f.Stmts[0])
	}
	if _, ok := f.Stmts[3].(*ast.AssignStmt); !ok {
		t.Fatalf("stmt3 should be AssignStmt, got %T", f.Stmts[3])
	}
}
