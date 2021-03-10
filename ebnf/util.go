package ebnf

import (
	"fmt"
	"regexp"
	"strings"

	"./r"
)

func jsonizeObject(ob r.Object) string {
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

	pp = strings.ReplaceAll(pp, ", TagChilds:[]r.Rule(nil)", "")
	pp = strings.ReplaceAll(pp, ", Childs:[]r.Rule(nil)", "")
	// pp = strings.ReplaceAll(pp, "TagChilds:[]r.Rule", "")
	// pp = strings.ReplaceAll(pp, "Childs:[]r.Rule", "")
	pp = strings.ReplaceAll(pp, "[]r.Rule{", "{")
	pp = strings.ReplaceAll(pp, "r.Rule{", "{")
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

func Pprint(header string, ob r.Object) {
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

	if len(pp) > 1200 {
		pp = pp[:1200] + " ..."
	}

	fmt.Printf("%s:\n%s\n", header, pp)
}

func PprintSrcSingleLine(pp string) {
	linebreaks := regexp.MustCompile(`(?s)([ \t]*[\r\n]+[ \t]*)+`)
	pp = linebreaks.ReplaceAllString(pp, " ")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	indent := regexp.MustCompile(`(?m)^[ \t]*`)
	pp = indent.ReplaceAllString(pp, "   ")

	if len(pp) > 1200 {
		pp = pp[:1200] + " ..."
	}

	fmt.Print(pp)
}

func PprintSequenceHeader(rule *r.Rule, printChilds bool, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := string("\"" + r.RuleTypes[rule.Operator] + "\"")
	// res := string("\"" + rule.Operator + "\"")

	switch rule.Operator {
	case r.Terminal, r.Invalid:
		res += fmt.Sprintf(", Pos:%d, %q", rule.Pos, rule.String)
	case r.Ident, r.Production:
		res += fmt.Sprintf(", %q:%d, Pos:%d", rule.String, rule.Int, rule.Pos)
	case r.Range:
		// TODO:!
	case r.SkipSpaces:
		res += fmt.Sprintf(", Pos:%d, %t", rule.Pos, rule.Bool)
	case r.Tag:
		res += fmt.Sprintf(", Pos:%d, Code:", rule.Pos)
		if printChilds {
			res += PprintProductions(&rule.TagChilds, sp+"  ")
		} else {
			res += "[...]"
		}
	case r.Factor:
		res += fmt.Sprintf(", Pos:%d, Rune:'%c'", rule.Pos, rule.Rune)
	default:
		res += fmt.Sprintf(", Pos:%d", rule.Pos)
	}

	return res
}

func PprintRule(rule *r.Rule, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := "{"
	res += PprintSequenceHeader(rule, true, sp)
	if len(rule.Childs) > 0 {
		res += ", " + PprintProductions(&rule.Childs, sp+"  ")
	}
	res += "}"
	return res
}

func PprintRuleOnly(rule *r.Rule, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := "{"
	res += PprintSequenceHeader(rule, false, sp)
	if len(rule.Childs) > 0 {
		res += ", [...]"
	}
	res += "}"
	return res
}

func PprintProductions(productions *[]r.Rule, space ...string) string {
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
		res += sp + "  " + PprintRule(rule, sp)
	}
	res += "\n" + sp + "}"
	return res
}

func PprintProductionsShort(productions *[]r.Rule, space ...string) string {
	str := PprintProductions(productions, space...)
	if len(str) > 1200 {
		str = str[:1200] + " ..."
	}
	return str
}

func PprintExtras(extras *map[string]r.Rule, space ...string) string {
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
		res += sp + "  \"" + name + "\":" + PprintRule(&rule, sp+"    ")
		comma = true
	}
	res += "\n" + sp + "}"
	return res
}

func PprintExtrasShort(extras *map[string]r.Rule, space ...string) string {
	str := PprintExtras(extras, space...)
	if len(str) > 1200 {
		str = str[:1200] + " ..."
	}
	return str
}

func LinePosFromStrPos(data string, pos int) string {
	lines := strings.Split(data, "\n")
	chars := 0
	if pos >= 0 {
		for i, line := range lines {
			lineLen := len(line)
			chars += lineLen
			if pos < chars { // Its in the last line.
				if i == 0 {
					pos += 2 // Correct for missing '\r\n'.
				}
				return fmt.Sprintf("ln %d, col %d", i+1, pos-(chars-lineLen)-1)
			}
		}
	}
	return "position outside of string"
}
