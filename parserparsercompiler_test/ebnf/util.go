package ebnf

import (
	"fmt"
	"regexp"
	"strings"

	"./seq"
)

func jsonizeObject(ob seq.Object) string {
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

	pp = strings.ReplaceAll(pp, ", TagChilds:[]seq.Sequence(nil)", "")
	pp = strings.ReplaceAll(pp, ", Childs:[]seq.Sequence(nil)", "")
	// pp = strings.ReplaceAll(pp, "TagChilds:[]seq.Sequence", "")
	// pp = strings.ReplaceAll(pp, "Childs:[]seq.Sequence", "")
	pp = strings.ReplaceAll(pp, "[]seq.Sequence{", "{")
	pp = strings.ReplaceAll(pp, "seq.Sequence{", "{")
	pp = strings.ReplaceAll(pp, "Operator:", "")
	pp = strings.ReplaceAll(pp, "TagChilds:", "")
	pp = strings.ReplaceAll(pp, "Childs:", "")
	pp = strings.ReplaceAll(pp, ", Rune:0", "")
	pp = strings.ReplaceAll(pp, ", String:\"\"", "")
	pp = strings.ReplaceAll(pp, ", Bool:false", "")
	pp = strings.ReplaceAll(pp, ", Int:0", "")
	pp = strings.ReplaceAll(pp, " String:", " ")
	pp = strings.ReplaceAll(pp, " Int:", " ")
	pp = strings.ReplaceAll(pp, " Bool:", " ")

	pp = strings.ReplaceAll(pp, "\\\"", "\"")

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

func Pprint(header string, ob seq.Object) {
	str := jsonizeObject(ob)
	if len(str) > 1200 {
		str = str[:1200] + " ..."
	}
	fmt.Printf("\n%s:\n   %s\n", header, str)
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

func PprintSequenceHeader(rule *seq.Sequence, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := string("\"" + rule.Operator + "\"")

	switch rule.Operator {
	case seq.Terminal, seq.Invalid:
		res += fmt.Sprintf(", Pos:%d, %q", rule.Pos, rule.String)
	case seq.Ident, seq.Production:
		res += fmt.Sprintf(", Pos:%d, %q:%d", rule.Pos, rule.String, rule.Int)
	case seq.Range:
		// TODO:!
	case seq.SkipSpaces:
		res += fmt.Sprintf(", Pos:%d, %t", rule.Pos, rule.Bool)
	case seq.Tag:
		res += fmt.Sprintf(", Pos:%d, Code:", rule.Pos)
		res += PprintProductions(&rule.TagChilds, sp+"  ")
	case seq.Factor:
		res += fmt.Sprintf(", Pos:%d, %c", rule.Pos, rule.Rune)
	}

	return res
}

func PprintSequence(rule *seq.Sequence, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := "{"
	res += PprintSequenceHeader(rule, sp)
	if len(rule.Childs) > 0 {
		res += ", " + PprintProductions(&rule.Childs, sp+"  ")
	}
	res += "}"
	return res
}

func PprintProductions(productions *[]seq.Sequence, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := "{\n"
	for i := range *productions {
		rule := &(*productions)[i]
		if i > 0 {
			res += ",\n"
		}
		res += sp + "  " + PprintSequence(rule, sp)
	}
	res += "\n" + sp + "}"
	return res
}

func PprintProductionsShort(productions *[]seq.Sequence, space ...string) string {
	str := PprintProductions(productions, space...)
	if len(str) > 1200 {
		str = str[:1200] + " ..."
	}
	return str
}

func PprintExtras(extras *map[string]seq.Sequence, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := "{\n"
	comma := false
	for name, rule := range *extras {
		if comma {
			res += ",\n"
		}
		res += sp + "  \"" + name + "\":" + PprintSequence(&rule, sp+"    ")
		comma = true
	}
	res += "\n" + sp + "}"
	return res
}

func PprintExtrasShort(extras *map[string]seq.Sequence, space ...string) string {
	str := PprintExtras(extras, space...)
	if len(str) > 1200 {
		str = str[:1200] + " ..."
	}
	return str
}
