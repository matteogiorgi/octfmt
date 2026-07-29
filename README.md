# Octave formatter

A source-code formatter for GNU Octave (`.m` files), written in Go. It
parses Octave into an AST and re-prints it with consistent indentation,
operator spacing, blank-line normalization, and comment formatting —
similar in spirit to `gofmt`.

## Install / build

```sh
go build -o octfmt ./cmd/octfmt
```

## Usage

```sh
octfmt file.m              # print formatted result to stdout
octfmt -w file.m ...       # rewrite files in place
octfmt -l file.m ...       # list files whose formatting would change
octfmt -d file.m ...       # print a unified diff instead of rewriting
cat file.m | octfmt        # read from stdin, write to stdout
```

Flags:

- `-w` write result to the source file instead of stdout
- `-l` list files that differ from octfmt's formatting
- `-d` show a diff instead of rewriting
- `-indent N` indent width in spaces (default 4)
- `-max-blank-lines N` max consecutive blank lines kept between statements (default 1)

Exit code is non-zero if any input file fails to parse; the offending
file(s) are left untouched and the error is printed to stderr.

## What it formats

- **Indentation**: consistent per-block indentation for
  `if/elseif/else`, `for`/`parfor`, `while`, `do/until`, `switch/case/otherwise`,
  `function`, `try/catch`, `unwind_protect`, matching the indentation style
  shown in the Octave manual (`case`/`otherwise` indented one level under
  `switch`).
- **Operator spacing**: binary operators get a single space on each side
  (`a + b`, `a == b`, `a && b`); `^`/`.^` and range `:` are kept tight
  (`a^2`, `1:10`) since that's the prevailing math-notation convention;
  unary `+`/`-`/`~` and postfix transpose (`'`, `.'`) are tight against
  their operand.
- **Blank lines**: runs of blank lines between statements are capped
  (default: at most 1), and the file always ends with exactly one newline.
- **Comments**: exactly one space after `%`/`#` is enforced, except for
  `%%`/`##` section-divider comments and `%{ ... %}`/`#{ ... #}` block
  comments, which are preserved verbatim. Trailing same-line comments get a
  2-space gap before the marker.
- **Block terminators** are normalized to the generic `end` (rather than
  `endif`/`endfor`/`endwhile`/`endfunction`/...) for brevity and MATLAB
  portability, except when a function had no explicit end at all in the
  source (script-style multi-function files that rely on the next
  `function` keyword or EOF to terminate) — that style is preserved as-is.
- **Multi-row matrix/cell literals** (`[...]`/`{...}` with more than one
  `;`- or newline-separated row) are printed one row per line, indented;
  single-row literals stay on one line regardless of length.

What it deliberately does **not** do: reflow/wrap long single-row
expressions or argument lists to a maximum line width (no line-width-based
prettifying), reorder code, or change string quote style (`'`/`"` are
semantically different in Octave — double-quoted strings support escapes —
so quote style is always preserved).

## Known limitations

- Octave's "command syntax" (e.g. `hold on`, `format long g`, `pkg load io`)
  is recognized heuristically. It only fires when the token right after an
  identifier is space-separated and looks like a bare word/string, and no
  assignment operator appears later on the line. Edge cases involving
  leading `-`/`+` command arguments (e.g. `disp -ascii`) are treated as
  arithmetic expressions rather than command arguments.
- `classdef`-based object definitions are not supported.
- No line-width-based reflowing of long expressions/argument lists.

## Project layout

```
internal/lexer    tokenizer (handles Octave's ' transpose-vs-string
                   ambiguity and matrix-row whitespace significance)
internal/ast      AST node definitions
internal/parser   recursive-descent parser -> ast.File
internal/printer  ast.File -> formatted source
internal/diff     minimal unified-diff generator, used by -d
cmd/octfmt        CLI
testdata/golden   golden-file fixtures (regenerate with `go test ./internal/printer/... -run TestGolden -update`)
```

## Tests

```sh
go test ./...
```

Includes lexer unit tests (transpose/string disambiguation, number
formats, comments), parser tests (matrix element-splitting ambiguity,
command-syntax detection, malformed-input error recovery), golden-file
formatting tests (exact expected output, plus an automatic idempotency
check that reformatting the output is a no-op) covering both everyday
constructs and edge cases (anonymous functions, dynamic fields, `do/until`,
`parfor` with a worker count, `global`/`persistent` with initializers,
`unwind_protect`), and CLI integration tests.
