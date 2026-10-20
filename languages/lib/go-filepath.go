// Package filepath: Go's path/filepath package for slash-separated paths, WRITTEN
// IN GO over package path (lib/go-path.go): the separator of every target here is
// '/', where the two packages agree.
package filepath

import (
	"path"
	"strings"
)

const Separator = '/'
const ListSeparator = ':'

func Clean(p string) string             { return path.Clean(p) }
func Join(elem ...string) string        { return path.Join(elem...) }
func Split(p string) (dir, file string) { return path.Split(p) }
func Ext(p string) string               { return path.Ext(p) }
func Base(p string) string              { return path.Base(p) }
func Dir(p string) string               { return path.Dir(p) }
func IsAbs(p string) bool               { return path.IsAbs(p) }
func ToSlash(p string) string           { return p }
func FromSlash(p string) string         { return p }

func SplitList(p string) []string {
	if p == "" {
		return []string{}
	}
	return strings.Split(p, ":")
}
