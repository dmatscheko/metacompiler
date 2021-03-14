package ebnf

import (
	"fmt"
	"regexp"
	"strings"

	"./r"
)

func times(s string, n int) string {
	res := s
	for ; n > 0; n-- {
		res = res + s
	}
	return res
}

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

func Shorten(s string) string {
	const maxLen = 800
	if len(s) > maxLen {
		s = s[:maxLen-5] + " ..."
	}
	return s
}

func PprintSrc(pp string) string {
	linebreaks := regexp.MustCompile(`(?s)([ \t]*[\r\n]+[ \t]*)+`)
	pp = linebreaks.ReplaceAllString(pp, "\n")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	indent := regexp.MustCompile(`(?m)^[ \t]*`)
	pp = indent.ReplaceAllString(pp, "   ")

	return Shorten(pp)
}

func PprintSrcSingleLine(pp string) string {
	linebreaks := regexp.MustCompile(`(?s)([ \t]*[\r\n]+[ \t]*)+`)
	pp = linebreaks.ReplaceAllString(pp, " ")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	indent := regexp.MustCompile(`(?m)^[ \t]*`)
	pp = indent.ReplaceAllString(pp, "   ")

	return Shorten(pp)
}

func PprintSequenceHeaderPos(rule *r.Rule, printChilds bool, printFlat bool, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := string("\"" + rule.Operator.String() + "\"")
	// res := string("\"" + rule.Operator + "\"")

	switch rule.Operator {
	case r.Token, r.Error:
		res += fmt.Sprintf(", Pos:%d, %q", rule.Pos, rule.String)
	case r.Ident, r.Production:
		res += fmt.Sprintf(", %q:%d, Pos:%d", rule.String, rule.Int, rule.Pos)
	// case r.Range:
	// 	// TODO:!
	case r.SkipSpace:
		res += fmt.Sprintf(", Pos:%d, %t", rule.Pos, rule.Bool)
	case r.Tag:
		res += fmt.Sprintf(", Pos:%d, Code:", rule.Pos)
		if printChilds {
			if printFlat {
				res += PprintRulesFlat(rule.TagChilds)
			} else {
				res += PprintRules(rule.TagChilds, sp+"  ")
			}
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

func PprintSequenceHeader(rule *r.Rule, printChilds bool, printFlat bool, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := string("\"" + rule.Operator.String() + "\"")
	// res := string("\"" + rule.Operator + "\"")

	switch rule.Operator {
	case r.Token, r.Error:
		res += fmt.Sprintf(", %q", rule.String)
	case r.Ident, r.Production:
		res += fmt.Sprintf(", %q:%d", rule.String, rule.Int)
	// case r.Range:
	// 	// TODO:!
	case r.SkipSpace:
		res += fmt.Sprintf(", %t", rule.Bool)
	case r.Tag:
		res += fmt.Sprintf(", Code:")
		if printChilds {
			if printFlat {
				res += PprintRulesFlat(rule.TagChilds)
			} else {
				res += PprintRules(rule.TagChilds, sp+"  ")
			}
		} else {
			res += "[...]"
		}
	case r.Factor:
		res += fmt.Sprintf(", Rune:'%c'", rule.Rune)
	default:
		// res += fmt.Sprintf("")
	}

	return res
}

func PprintRule(rule *r.Rule, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := "{"
	res += PprintSequenceHeaderPos(rule, true, false, sp)
	if rule.Childs != nil && len(*rule.Childs) > 0 {
		res += ", " + PprintRules(rule.Childs, sp+"  ")
	}
	res += "}"
	return res
}

func PprintRuleFlat(rule *r.Rule, printChilds bool, printPos bool) string {
	res := "{"
	if printPos {
		res += PprintSequenceHeaderPos(rule, true, true)
	} else {
		res += PprintSequenceHeader(rule, true, true)
	}
	if printChilds {
		if rule.Childs != nil && len(*rule.Childs) > 0 {
			res += ", " + PprintRulesFlat(rule.Childs)
		}
	} else {
		res += "[...]"
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
	res += PprintSequenceHeaderPos(rule, false, false, sp)
	if rule.Childs != nil && len(*rule.Childs) > 0 {
		res += ", [...]"
	}
	res += "}"
	return res
}

func PprintRules(productions *r.Rules, space ...string) string {
	sp := ""
	if len(space) > 0 {
		sp = space[0]
	}
	res := "{\n"
	for i := range *productions {
		rule := (*productions)[i]
		if i > 0 {
			res += ",\n"
		}
		res += sp + "  " + PprintRule(rule, sp)
	}
	res += "\n" + sp + "}"
	return res
}

func PprintRulesFlat(productions *r.Rules) string {
	res := "{"
	for i := range *productions {
		rule := (*productions)[i]
		if i > 0 {
			res += ", "
		}
		res += PprintRuleFlat(rule, true, false)
		// res += PprintRuleFlat(rule, true, true)
	}
	res += "}"
	return res
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
	return "position outside of EBNF"
}
