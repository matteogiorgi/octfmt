// Package diff produces a minimal unified-style line diff, used by the
// octfmt CLI's -d flag.
package diff

import (
	"fmt"
	"strings"
)

type op int

const (
	opEqual op = iota
	opDelete
	opInsert
)

type edit struct {
	op   op
	line string
}

// lines splits s into lines without losing a trailing partial line.
func lines(s string) []string {
	if s == "" {
		return nil
	}
	l := strings.Split(s, "\n")
	if l[len(l)-1] == "" {
		l = l[:len(l)-1]
	}
	return l
}

// Unified returns a unified-diff-formatted comparison of a and b, labeled
// with aName/bName. It returns "" if a == b.
func Unified(aName, bName, a, b string) string {
	al, bl := lines(a), lines(b)
	edits := myers(al, bl)
	if len(edits) == 0 {
		return ""
	}
	hunks := groupHunks(edits, 3)
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", aName, bName)
	for _, h := range hunks {
		writeHunk(&sb, h)
	}
	return sb.String()
}

// myers computes a simple LCS-based edit script (adequate for typical
// source-file sizes; not optimized for very large inputs).
func myers(a, b []string) []edit {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []edit
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, edit{opEqual, a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			out = append(out, edit{opDelete, a[i]})
			i++
		default:
			out = append(out, edit{opInsert, b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, edit{opDelete, a[i]})
	}
	for ; j < m; j++ {
		out = append(out, edit{opInsert, b[j]})
	}
	return out
}

type hunk struct {
	edits          []edit
	aStart, bStart int
}

func groupHunks(edits []edit, context int) []hunk {
	var hunks []hunk
	aLine, bLine := 1, 1
	i := 0
	for i < len(edits) {
		if edits[i].op == opEqual {
			aLine++
			bLine++
			i++
			continue
		}
		// found a change; back up to include leading context
		start := i
		ctxBefore := 0
		for start > 0 && edits[start-1].op == opEqual && ctxBefore < context {
			start--
			ctxBefore++
		}
		end := i
		for end < len(edits) {
			// extend through this change and any changes separated by <=2*context equal lines
			for end < len(edits) && edits[end].op != opEqual {
				end++
			}
			// count following equal run
			run := 0
			k := end
			for k < len(edits) && edits[k].op == opEqual && run < 2*context {
				k++
				run++
			}
			if k < len(edits) && edits[k].op != opEqual {
				end = k
				continue
			}
			break
		}
		trailing := context
		hEnd := end
		for hEnd < len(edits) && trailing > 0 && edits[hEnd].op == opEqual {
			hEnd++
			trailing--
		}
		hAStart := aLine - ctxBefore
		hBStart := bLine - ctxBefore
		h := hunk{edits: edits[start:hEnd], aStart: hAStart, bStart: hBStart}
		hunks = append(hunks, h)

		for k := i; k < hEnd; k++ {
			switch edits[k].op {
			case opEqual:
				aLine++
				bLine++
			case opDelete:
				aLine++
			case opInsert:
				bLine++
			}
		}
		i = hEnd
	}
	return hunks
}

func writeHunk(sb *strings.Builder, h hunk) {
	aCount, bCount := 0, 0
	for _, e := range h.edits {
		switch e.op {
		case opEqual:
			aCount++
			bCount++
		case opDelete:
			aCount++
		case opInsert:
			bCount++
		}
	}
	fmt.Fprintf(sb, "@@ -%d,%d +%d,%d @@\n", h.aStart, aCount, h.bStart, bCount)
	for _, e := range h.edits {
		switch e.op {
		case opEqual:
			sb.WriteString(" " + e.line + "\n")
		case opDelete:
			sb.WriteString("-" + e.line + "\n")
		case opInsert:
			sb.WriteString("+" + e.line + "\n")
		}
	}
}
