// shape-scan reports layer-2 function bodies that are IDENTICAL MODULO NAMES
// across languages/lib/*.metajs.
//
// Why this exists, and why the obvious view does not work: for months the only
// view anyone had of layer-2 duplication was the table of `js_*` extern names
// defined in more than one file. That table both UNDERSTATES the duplication
// (python and ruby wrote the same exact-decimal bignum under different names for
// months, and the name view is blind to it by construction) and OVERSTATES it
// (24 duplicated names, of which only two have bodies that group at any
// threshold; `js_pylen`'s five copies are 7, 9, 11, 16 and 7 lines of genuinely
// different code). It is measuring names. The thing being merged is bodies.
//
// So: strip comments, tokenise, rewrite every non-keyword identifier to a
// positional token (#1, #2, ... in order of first appearance), and group by the
// resulting string. Two bodies land in the same group exactly when one is the
// other with the identifiers renamed - which is the property that makes a merge
// mechanical and a difference list meaningful.
//
//	go run ./tools/shape-scan               # the 140-char threshold, the number to quote
//	go run ./tools/shape-scan -min 60       # permissive; sees more, most of it noise
//	go run ./tools/shape-scan -v            # list every member with its file:line
//	go run ./tools/shape-scan -max 8        # exit 1 above N groups: the CI ratchet
//
// The threshold is on the NORMALISED body, not the source, so it is a measure of
// structure and not of naming or indentation.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The dialect's reserved words. These are structure, not names, so they survive
// normalisation - otherwise `if` and `while` bodies would group together.
var keywords = map[string]bool{
	"function": true, "return": true, "var": true, "let": true, "const": true,
	"if": true, "else": true, "for": true, "while": true, "do": true,
	"break": true, "continue": true, "typeof": true, "new": true, "delete": true,
	"in": true, "of": true, "true": true, "false": true, "null": true,
	"undefined": true, "this": true, "throw": true, "try": true, "catch": true,
	"finally": true, "switch": true, "case": true, "default": true,
	"instanceof": true, "void": true, "yield": true, "import": true, "export": true,
}

type body struct {
	file  string
	name  string
	line  int
	lines int
	norm  string
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

// normalise rewrites src into its shape: comments gone, runs of whitespace
// collapsed to nothing where the grammar allows and to one space where it does
// not, string and numeric literals kept VERBATIM (a body that differs only in a
// constant is a different specification, not a renaming), and every non-keyword
// identifier replaced by #n in order of first appearance.
func normalise(src string) string {
	var out strings.Builder
	seen := map[string]int{}
	i, n := 0, len(src)
	for i < n {
		c := src[i]
		switch {
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
		case c == '"' || c == '\'':
			q := c
			out.WriteByte(c)
			i++
			for i < n && src[i] != q {
				if src[i] == '\\' && i+1 < n {
					out.WriteByte(src[i])
					i++
				}
				out.WriteByte(src[i])
				i++
			}
			out.WriteByte(q)
			i++
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case isIdentStart(c):
			j := i
			for j < n && isIdentPart(src[j]) {
				j++
			}
			w := src[i:j]
			if keywords[w] {
				// Keep a separator: `return x` must not become `returnx`.
				out.WriteString(" " + w + " ")
			} else {
				k, ok := seen[w]
				if !ok {
					k = len(seen) + 1
					seen[w] = k
				}
				fmt.Fprintf(&out, "#%d", k)
			}
			i = j
		default:
			out.WriteByte(c)
			i++
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

// braceDelta counts { and } on one line, skipping anything inside a string
// literal or a comment. Counting raw bytes instead is not good enough: a `"{"`
// in a format string or a `}` in a comment would end the function early.
func braceDelta(line string) int {
	d, i, n := 0, 0, len(line)
	for i < n {
		c := line[i]
		switch {
		case c == '/' && i+1 < n && line[i+1] == '/':
			return d
		case c == '"' || c == '\'':
			q := c
			i++
			for i < n && line[i] != q {
				if line[i] == '\\' {
					i++
				}
				i++
			}
			i++
		case c == '{':
			d++
			i++
		case c == '}':
			d--
			i++
		default:
			i++
		}
	}
	return d
}

// collect pulls every top-level `function name(...) { ... }` out of one file.
// Top-level means column 0, which is how every lib/*.metajs is written; a nested
// function is part of its parent's body and is normalised with it.
//
// The end of a function is found by BRACE COUNTING, not by looking for a `}` in
// column 0. The house style mostly puts one there, but the one-line form
// `function k1I32(v) { return ktNorm(k1Val(v), 32, 0) }` does not - and scanning
// for the next column-0 brace made that swallow the following 1,800 lines and
// report a group of three "identical" bodies of 6, 35 and 41 lines.
func collect(path string) ([]body, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(raw)
	lines := strings.Split(src, "\n")
	var out []body
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "function ") {
			continue
		}
		open := strings.Index(lines[i], "(")
		if open < 0 {
			continue
		}
		name := strings.TrimSpace(lines[i][len("function "):open])
		depth := braceDelta(lines[i])
		j := i
		for depth > 0 && j+1 < len(lines) {
			j++
			depth += braceDelta(lines[j])
		}
		if depth != 0 {
			continue // unbalanced: not something to reason about
		}
		// The body is everything between the header and the closing line. For a
		// one-liner that is empty, so it contributes no shape - which is right:
		// there is nothing there to be duplicated.
		var text string
		if j == i {
			// One-liner: the body is whatever sits between the outermost braces.
			if a, b := strings.Index(lines[i], "{"), strings.LastIndex(lines[i], "}"); a >= 0 && b > a {
				text = lines[i][a+1 : b]
			}
		} else {
			text = strings.Join(lines[i+1:j], "\n")
		}
		out = append(out, body{
			file: filepath.Base(path), name: name, line: i + 1,
			lines: j - i + 1, norm: normalise(text),
		})
		i = j
	}
	return out, nil
}

func main() {
	min := flag.Int("min", 140, "minimum normalised body length to consider")
	verbose := flag.Bool("v", false, "list every member of every group")
	max := flag.Int("max", -1, "exit 1 if more than this many groups (CI ratchet)")
	// -max alone is NOT a sufficient ratchet, and the gap is easy to hit by
	// accident: copying a body that is ALREADY duplicated into a third language
	// grows an existing group instead of creating a new one, so the group COUNT
	// does not move. Measured: pasting pyABigMul into lua-rt.metajs left the count
	// at 15 while recoverable went 481 -> 511. The recoverable-line total is the
	// number that actually tracks the debt, so tests/gates.sh ratchets both.
	maxLines := flag.Int("max-lines", -1, "exit 1 if more than this many RECOVERABLE lines (CI ratchet)")
	dir := flag.String("dir", "languages/lib", "directory of .metajs layer-2 files")
	flag.Parse()

	files, err := filepath.Glob(filepath.Join(*dir, "*.metajs"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "shape-scan: no .metajs files under %s (run from the repo root)\n", *dir)
		os.Exit(2)
	}
	sort.Strings(files)

	groups := map[string][]body{}
	total := 0
	for _, f := range files {
		bs, err := collect(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "shape-scan: %v\n", err)
			os.Exit(2)
		}
		total += len(bs)
		for _, b := range bs {
			if len(b.norm) >= *min {
				groups[b.norm] = append(groups[b.norm], b)
			}
		}
	}

	// A group is only interesting if it spans FILES. Two copies inside one file
	// are that language's own business and the merge argument does not apply.
	var keys []string
	for k, v := range groups {
		distinct := map[string]bool{}
		for _, b := range v {
			distinct[b.file] = true
		}
		if len(distinct) > 1 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(a, b int) bool {
		if len(groups[keys[a]]) != len(groups[keys[b]]) {
			return len(groups[keys[a]]) > len(groups[keys[b]])
		}
		return groups[keys[a]][0].name < groups[keys[b]][0].name
	})

	// Keeping ONE copy is the best case, so everything but the largest member of
	// each group is recoverable.
	occupied, recoverable := 0, 0
	for _, k := range keys {
		g := groups[k]
		s, big := sumLines(g), 0
		for _, b := range g {
			if b.lines > big {
				big = b.lines
			}
		}
		occupied += s
		recoverable += s - big
	}

	for _, k := range keys {
		g := groups[k]
		names := make([]string, 0, len(g))
		for _, b := range g {
			names = append(names, fmt.Sprintf("%s:%s", strings.TrimSuffix(b.file, ".metajs"), b.name))
		}
		fmt.Printf("%2d bodies, %3d lines  %s\n", len(g), sumLines(g), strings.Join(names, "  "))
		if *verbose {
			for _, b := range g {
				fmt.Printf("      %s:%d  %s (%d lines)\n", b.file, b.line, b.name, b.lines)
			}
			fmt.Printf("      shape: %.120s...\n", k)
		}
	}
	fmt.Printf("\n%d cross-language groups, %d lines occupied, %d recoverable "+
		"(threshold %d chars, %d top-level bodies scanned)\n",
		len(keys), occupied, recoverable, *min, total)

	over := false
	if *max >= 0 && len(keys) > *max {
		fmt.Fprintf(os.Stderr, "shape-scan: %d groups exceeds the ratchet of %d\n", len(keys), *max)
		over = true
	}
	if *maxLines >= 0 && recoverable > *maxLines {
		fmt.Fprintf(os.Stderr, "shape-scan: %d recoverable lines exceeds the ratchet of %d\n", recoverable, *maxLines)
		over = true
	}
	if over {
		os.Exit(1)
	}
}

func sumLines(g []body) int {
	n := 0
	for _, b := range g {
		n += b.lines
	}
	return n
}
