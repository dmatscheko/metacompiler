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
	panic(fmt.Sprintf("error at TAG: %#v", terminal))
}

func getIDAndCodeFromTag(tagAnnotation object) (string, string) {
	tagID := ""
	tagCode := ""

	if annotationSeq, ok := tagAnnotation.(sequence); ok {
		// we have the annotation of the TAG. The annotation can be either a single TERMINAL, or a sequence of TERMINALs.
		if _, ok := annotationSeq[0].(string); ok { // single TERMINAL
			tagID = getTextFromTerminal(annotationSeq)
		} else if len(annotationSeq) == 2 { // sequence of TERMINALs (so far there is only ID and code, so 2 elements)
			tagID = getTextFromTerminal(annotationSeq[0])
			tagCode = getTextFromTerminal(annotationSeq[1])
		} else {
			panic(fmt.Sprintf("only ID and code is allowed inside TAG: %#v", tagAnnotation))
		}
	} else {
		panic(fmt.Sprintf("error at TAG: %#v", tagAnnotation))
	}

	return tagID, tagCode
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
