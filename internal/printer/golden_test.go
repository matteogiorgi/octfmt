package printer

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/matteo/octfmt/internal/parser"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
	matches, err := filepath.Glob("../../testdata/golden/*.m.in")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no golden fixtures found")
	}
	for _, in := range matches {
		in := in
		name := filepath.Base(in)
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			f, errs := parser.Parse(string(src))
			for _, e := range errs {
				t.Fatalf("parse error: %v", e)
			}
			got := Print(f, DefaultOptions)

			goldenPath := in[:len(in)-len(".in")] + ".golden"
			if *update {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden (run with -update to create): %v", err)
			}
			if got != string(want) {
				t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}

			// idempotency: reformatting the output must be a no-op
			f2, errs2 := parser.Parse(got)
			for _, e := range errs2 {
				t.Fatalf("parse error on reformat: %v", e)
			}
			got2 := Print(f2, DefaultOptions)
			if got2 != got {
				t.Errorf("formatting %s is not idempotent", name)
			}
		})
	}
}
