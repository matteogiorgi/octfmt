package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "octfmt")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func TestCLIStdin(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader("x=1+2;\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	want := "x = 1 + 2;\n"
	if out.String() != want {
		t.Errorf("got %q want %q", out.String(), want)
	}
}

func TestCLIWriteAndList(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.m")
	if err := os.WriteFile(path, []byte("x=1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// -l should report the file since it's unformatted
	cmd := exec.Command(bin, "-l", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if strings.TrimSpace(out.String()) != path {
		t.Errorf("expected -l to list %s, got %q", path, out.String())
	}

	// -w should rewrite in place
	cmd = exec.Command(bin, "-w", path)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "x = 1;\n" {
		t.Errorf("got %q", string(data))
	}

	// -l on an already-formatted file should print nothing
	cmd = exec.Command(bin, "-l", path)
	out.Reset()
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if out.String() != "" {
		t.Errorf("expected no output, got %q", out.String())
	}
}

func TestCLIParseErrorExitCode(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader("if x\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for malformed input")
	}
	if stderr.Len() == 0 {
		t.Error("expected an error message on stderr")
	}
}
