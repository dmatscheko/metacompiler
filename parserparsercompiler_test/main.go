package main

import (
	"fmt"

	"./ebnf"
)

var (
	ebnfs = []string{
		// `"a" {
		// a = "a1" ( "a2" | "a3" | "ab\\cd" | "\"" | "\\" | "123\n456" ) { "a4" } [ "a5" ] "a6" ;
		// } "z" `,
		// `{
		// expr = term { plus term } .
		// term = factor { times factor } .
		// factor = number | '(' expr ')' .

		// plus = "+" | "-" .
		// times = "*" | "/" .

		// number = digit { digit } .
		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// }`,
		// `a = "1"`,
		// `{ a = "1" ;`,
		// `{ hello world = "1"; }`,
		// `{ foo = bar . }`,
		// `{ foo = "bar" . }`,
		// `{ }`,

		// Bigger EBNF of EBNF with tags (can parse):
		`"aEBNF of aEBNF" {
		program = [ title ] [ tag ] "{" { production } "}" [ tag ] [ comment ] .
		production  = name [ tag ] "=" [ expression ] ( "." | ";" ) .
		expression  = sequence .
		sequence    = alternative { alternative } .
		alternative = term { "|" term } .
		term        = ( name | text [ "..." text ] | group | option | repetition | skipspaces ) [ tag ] .
		group       = "(" expression  ")" .
		option      = "[" expression "]" .
		repetition  = "{" expression "}" .
		skipspaces  = "+" | "-" .

		title = text .
		comment = text .

		name <"collect">  = ( small | caps ) { small | caps | digit | "_" } .
		text <"collect"> = "\"" - { small | caps | digit | special } "\"" + .

		tag  = "<" text { ";" text } ">" .

		digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		} "Some comment"`,

		// `"EBNF of EBNF (can parse)" {
		// program = [ title ] "{" { production } "}" [ comment ] .

		// production  = name "=" [ expression ] ( "." | ";" ) .
		// expression  = sequence .
		// sequence    = alternative { alternative } .
		// alternative = term { "|" term } .
		// term        = name | text [ "..." text ] | group | option | repetition | skipspaces .
		// group       = "(" expression ")" .
		// option      = "[" expression "]" .
		// repetition  = "{" expression "}" .
		// skipspaces = "+" | "-" .

		// title = text .
		// comment = text .

		// name = ( small | caps ) { small | caps | digit | "_" } .
		// text = "\"" - { small | caps | digit | special } "\"" + .

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		// special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		// } "Some comment"`,

		// Minimal EBNF of EBNF (can NOT parse) (sequence | alternative can both not be together with the rest of the expression alternatives, but why? <- sequence and alternative can not have itself inside):
		// `{
		// ebnf = "{" { production } "}" .
		// production  = name "=" [ expression ] ( "." | ";" ) .
		// expression  = name | text [ "..." text ] | group | option | repetition | skipspaces | alternative | sequence .
		// sequence    = expression expression { expression } .
		// alternative = expression "|" expression { "|" expression } .
		// group       = "(" expression ")" .
		// option      = "[" expression "]" .
		// repetition  = "{" expression "}" .
		// skipspaces = "+" | "-" .

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		// special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		// name = ( small | caps ) { small | caps | digit | "_" } .
		// text = "\"" - { small | caps | digit | special } "\"" + .
		// }`,

		// // Default EBNF of EBNF (can MAYBE parse) (sequence | alternative can both not be together with the rest of the expression alternatives, but why?):
		// `{
		// 			ebnf = [ title ] "{" { production } "}" [ comment ] .
		// 			production  = name "=" [ expression ] "." .
		// 			expression  = term | sequence | alternative .
		// 			sequence    = term term { term } .
		// 			alternative = term "|" term { "|" term } .
		// 			term        = name | text [ "..." text ] | group | option | repetition .
		// 			group       = "(" expression ")" .
		// 			option      = "[" expression "]" .
		// 			repetition  = "{" expression "}" .

		// 			title = text .
		// 			comment = text .

		// 			digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// 			small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		// 			caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		// 			special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		// 			name = ( small | caps ) { small | caps | digit | "_" } .
		// 			text = "\"" - { small | caps | digit | special } "\"" + .
		// 			}`,

		// `{ top = "ABC" ; }`,
	}

	tests = []string{
		// 		"a1a3a4a4a5a6",
		// 		"a1 a2a6",
		// 		"a1 a3 a4 a6",
		// 		"a1 a4 a5 a6",
		// 		"a1 a2 a4 a5 a5 a6",
		// 		"a1 a2 a4 a5 a6 a7",
		// 		"your ad here",
		// 		"2",
		// 		"2*3 + 4/23 - 7",
		// 		"(3 + 4) * 6-2+(4*(4))",
		// 		"-2",
		// 		"3 +",
		// 		"(4 + 3",
		// 		`{ }`,
		// 		`{ moo < "test" ; "toast" > = "ABC" | "DEF" . }`,
		//
		// `"aEBNF of aEBNF" {
		// program = [ title ] [ tag ] "{" { production } "}" [ tag ] [ comment ] .
		// production  = name [ tag ] "=" [ expression ] ( "." | ";" ) .
		// expression  = sequence .
		// sequence    = alternative { alternative } .
		// alternative = term { "|" term } .
		// term        = ( name | text [ "..." text ] | group | option | repetition | skipspaces ) [ tag ] .
		// group       = "(" expression ")" .
		// option      = "[" expression "]" .
		// repetition  = "{" expression "}" .
		// skipspaces  < "foo" > = "+" | "-" .

		// title = text .
		// comment = text .

		// name = ( small | caps ) { small | caps | digit | "_" } .
		// text = "\"" - { small | caps | digit | special } "\"" + .

		// tag = "<" text { ";" text } ">" .

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		// special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		// } "Some comment"`,

		// `{ top = abc { uvw } ; abc = "ABC" ; uvw = "XYZ" ; }`,
		`{ top = "ABC" ; }`,
	}
)

func main() {
	for _, srcEbnf := range ebnfs {
		fmt.Println()
		fmt.Println("===========================================================")
		fmt.Println()
		// parses an EBNF and configures the grammar with it
		// fmt.Println(srcEbnf)
		grammar, err := ebnf.ParseEBNF(srcEbnf)
		if err != nil {
			// fmt.Printf("%v\n\n", err)
			fmt.Println()
			continue
		}
		fmt.Println()
		fmt.Println("tests:")
		for _, srcCode := range tests {
			// uses the grammar to parse a new type of code
			// fmt.Println(srcCode)

			ebnf.PprintSrcSingleLine(srcCode)

			res, err := ebnf.ParseWithGrammar(grammar, srcCode)
			if err != nil {
				fmt.Printf("  (%v)", err)
			}
			resStr := "Success"
			if !res {
				resStr = "Fail"
			}
			fmt.Printf(" ==> %s\n\n", resStr)
		}
		fmt.Println()
	}

}
