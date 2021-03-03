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

func getTextFromTerminal(terminal object) string {
	t, ok := terminal.(sequence)
	if ok && len(t) == 2 && t[0] == "TERMINAL" {
		tStr, ok := t[1].(string)
		if ok {
			return tStr
		}
	}
	panic(fmt.Sprintf("Error at TAG TERMINAL: %#v", terminal))
}

func getIDAndCodeFromTag(tag object) string {
	if tagSeq, ok := tag.(sequence); ok && len(tagSeq) > 1 {
		// The annotation can be either a single TERMINAL, or a sequence of TERMINALs.
		if annotationSeq, ok := tagSeq[1].(sequence); ok && len(annotationSeq) > 0 {
			if _, ok := annotationSeq[0].(string); ok { // A single TERMINAL.
				return getTextFromTerminal(annotationSeq)
			}
			return getTextFromTerminal(annotationSeq[0]) // A sequence and so the code is in the first TERMINAL.
		}
	}
	panic(fmt.Sprintf("Error at TAG: %#v", tag))
}

func jsonizeObject(ob object) string {
	pp := fmt.Sprintf("%#v", ob)
	pp = strings.ReplaceAll(pp, "[]interface {}", "")
	if strings.HasPrefix(pp, "[]int") {
		pp = strings.Replace(pp, "[]int", "", 1)
	} else if strings.HasPrefix(pp, "[]string") {
		pp = strings.Replace(pp, "[]string", "", 1)
	} else if strings.HasPrefix(pp, "[][]") {
		pp = strings.Replace(pp, "[][]", "[]", 1)
	} else if strings.HasPrefix(pp, "map[string]interface {}") {
		pp = strings.Replace(pp, "map[string]interface {}", "map", 1)
	} else if strings.HasPrefix(pp, "map[string]string") {
		pp = strings.Replace(pp, "map[string]string", "map", 1)
	}

	// pp = strings.ReplaceAll(pp, "ebnf.group{}, ", "G: ")
	pp = strings.ReplaceAll(pp, "ebnf.group{}, ", "")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	return pp
}

// func jsonizeObject(ob object) string {
// 	pp := fmt.Sprintf("%#v", ob)
// 	pp = strings.ReplaceAll(pp, "[]interface {}", "")
// 	if strings.HasPrefix(pp, "[]int") {
// 		pp = strings.Replace(pp, "[]int", "", 1)
// 	} else if strings.HasPrefix(pp, "[]string") {
// 		pp = strings.Replace(pp, "[]string", "", 1)
// 	} else if strings.HasPrefix(pp, "[]") {
// 		pp = strings.Replace(pp, "[]", "", 1)
// 	} else if strings.HasPrefix(pp, "map[string]interface {}") {
// 		pp = strings.Replace(pp, "map[string]interface {}", "", 1)
// 	} else if strings.HasPrefix(pp, "map[string]string") {
// 		pp = strings.Replace(pp, "map[string]string", "", 1)
// 	}

// 	space := regexp.MustCompile(`[ \t]+`)
// 	pp = space.ReplaceAllString(pp, " ")

// 	return pp
// }

func Pprint(header string, ob object) {
	fmt.Printf("\n%s:\n   %s\n", header, jsonizeObject(ob))
}

func PprintSrc(header string, pp string) {
	linebreaks := regexp.MustCompile(`(?s)([ \t]*[\r\n]+[ \t]*)+`)
	pp = linebreaks.ReplaceAllString(pp, "\n")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	indent := regexp.MustCompile(`(?m)^[ \t]*`)
	pp = indent.ReplaceAllString(pp, "   ")

	fmt.Printf("%s:\n%s\n", header, pp)
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
