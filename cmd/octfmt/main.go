// Command octfmt formats GNU Octave (.m) source files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/matteo/octave-formatter/internal/diff"
	"github.com/matteo/octave-formatter/internal/parser"
	"github.com/matteo/octave-formatter/internal/printer"
)

var (
	writeFlag  = flag.Bool("w", false, "write result to (source) file instead of stdout")
	listFlag   = flag.Bool("l", false, "list files whose formatting differs from octfmt's")
	diffFlag   = flag.Bool("d", false, "display diffs instead of rewriting files")
	indentFlag = flag.Int("indent", printer.DefaultOptions.IndentWidth, "indent width in spaces")
	blankFlag  = flag.Int("max-blank-lines", printer.DefaultOptions.MaxBlankLines, "maximum consecutive blank lines to keep")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: octfmt [flags] [path ...]\n\nFormats GNU Octave source. With no path, reads from stdin and writes to stdout.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	opts := printer.Options{IndentWidth: *indentFlag, MaxBlankLines: *blankFlag}

	args := flag.Args()
	exitCode := 0

	if len(args) == 0 {
		if *writeFlag {
			fmt.Fprintln(os.Stderr, "octfmt: -w requires file arguments")
			os.Exit(2)
		}
		if err := processStdin(opts); err != nil {
			fmt.Fprintln(os.Stderr, "octfmt:", err)
			exitCode = 1
		}
		os.Exit(exitCode)
	}

	for _, path := range args {
		if err := processFile(path, opts); err != nil {
			fmt.Fprintln(os.Stderr, "octfmt:", err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func formatSource(src string, opts printer.Options) (string, error) {
	f, errs := parser.Parse(src)
	if len(errs) > 0 {
		var sb strings.Builder
		for _, e := range errs {
			sb.WriteString(e.Error())
			sb.WriteByte('\n')
		}
		return "", errors.New(strings.TrimRight(sb.String(), "\n"))
	}
	return printer.Print(f, opts), nil
}

func processStdin(opts printer.Options) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	src := string(data)
	formatted, err := formatSource(src, opts)
	if err != nil {
		return fmt.Errorf("<standard input>: %w", err)
	}
	switch {
	case *listFlag:
		if formatted != src {
			fmt.Println("<standard input>")
		}
	case *diffFlag:
		if d := diff.Unified("<standard input>", "<standard input>", src, formatted); d != "" {
			fmt.Print(d)
		}
	default:
		fmt.Print(formatted)
	}
	return nil
}

func processFile(path string, opts printer.Options) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	src := string(data)
	formatted, err := formatSource(src, opts)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	switch {
	case *listFlag:
		if formatted != src {
			fmt.Println(path)
		}
	case *diffFlag:
		if d := diff.Unified(path+".orig", path, src, formatted); d != "" {
			fmt.Print(d)
		}
	case *writeFlag:
		if formatted != src {
			if err := os.WriteFile(path, []byte(formatted), info.Mode().Perm()); err != nil {
				return err
			}
		}
	default:
		fmt.Print(formatted)
	}
	return nil
}
