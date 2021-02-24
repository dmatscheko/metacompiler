package ebnf

import (
	"fmt"
	"regexp"
	"strings"
)

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func pprint(header string, ob object) {
	pp := fmt.Sprintf("%#v", ob)
	pp = strings.ReplaceAll(pp, "[]interface {}", "")
	if strings.HasPrefix(pp, "[]int") {
		pp = strings.Replace(pp, "[]int", "", 1)
	} else if strings.HasPrefix(pp, "[]string") {
		pp = strings.Replace(pp, "[]string", "", 1)
	} else if strings.HasPrefix(pp, "[]") {
		pp = strings.Replace(pp, "[]", "", 1)
	}

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	fmt.Printf("\n%s:\n   %s\n", header, pp)
}

func pprintSrc(header string, pp string) {
	linebreaks := regexp.MustCompile(`(?s)([ \t]*[\r\n]+[ \t]*)+`)
	pp = linebreaks.ReplaceAllString(pp, "\n")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	indent := regexp.MustCompile(`(?m)^[ \t]*`)
	pp = indent.ReplaceAllString(pp, "   ")

	fmt.Printf("\n%s:\n%s\n", header, pp)
}

func PprintSrcSingleLine(pp string) {
	linebreaks := regexp.MustCompile(`(?s)([ \t]*[\r\n]+[ \t]*)+`)
	pp = linebreaks.ReplaceAllString(pp, " ")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	indent := regexp.MustCompile(`(?m)^[ \t]*`)
	pp = indent.ReplaceAllString(pp, "   ")

	fmt.Print(pp)
}
