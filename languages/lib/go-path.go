// Package path: Go's path package WRITTEN IN GO, compiled with a program that
// imports "path" (core.importFile in both go grammars): slash-separated paths,
// cleaned lexically.
package path

import "strings"

// The shortest path equivalent to p: one slash between elements, no "." element,
// no "dir/.." pair, no ".." at the start of a rooted path, "." for an empty one.
func Clean(p string) string {
	if p == "" {
		return "."
	}
	rooted := p[0] == '/'
	parts := strings.Split(p, "/")
	out := []string{}
	for _, e := range parts {
		switch {
		case e == "" || e == ".":
		case e == "..":
			if len(out) > 0 && out[len(out)-1] != ".." {
				out = out[:len(out)-1]
			} else if !rooted {
				out = append(out, "..")
			}
		default:
			out = append(out, e)
		}
	}
	s := strings.Join(out, "/")
	if rooted {
		return "/" + s
	}
	if s == "" {
		return "."
	}
	return s
}

func Join(elem ...string) string {
	nonEmpty := []string{}
	for _, e := range elem {
		if e != "" {
			nonEmpty = append(nonEmpty, e)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	return Clean(strings.Join(nonEmpty, "/"))
}

// Split: everything up to the last slash, and what follows it.
func Split(p string) (dir, file string) {
	i := strings.LastIndex(p, "/")
	return p[:i+1], p[i+1:]
}

func Ext(p string) string {
	for i := len(p) - 1; i >= 0 && p[i] != '/'; i-- {
		if p[i] == '.' {
			return p[i:]
		}
	}
	return ""
}

func Base(p string) string {
	if p == "" {
		return "."
	}
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[0 : len(p)-1]
	}
	if i := strings.LastIndex(p, "/"); i >= 0 {
		p = p[i+1:]
	}
	if p == "" {
		return "/"
	}
	return p
}

func Dir(p string) string {
	dir, _ := Split(p)
	return Clean(dir)
}

func IsAbs(p string) bool { return len(p) > 0 && p[0] == '/' }
