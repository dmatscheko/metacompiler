package main

import (
	"fmt"

	"./ebnf"
)

var (
	ebnfs = []string{
		`"a" {
		a = "a1" ( "a2" | "a3" | "ab\\cd" | "\"" | "\\" | "123\n456" ) { "a4" } [ "a5" ] "a6" ;
		} "z" `,
		`{
		expr = term { plus term } .
		term = factor { times factor } .
		factor = number | '(' expr ')' .

		plus = "+" | "-" .
		times = "*" | "/" .

		number = digit { digit } .
		digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		}`,
		`a = "1"`,
		`{ a = "1" ;`,
		`{ hello world = "1"; }`,
		`{ foo = bar . }`,
		`{ foo = "bar" . }`,
		`{ }`,
		`"EBNF of EBNF" {
		program = [ token ] "{" { production } "}" [ token ] .
		production  = name "=" [ expression ] ( "." | ";" ) .
		expression  = alternative { "|" alternative } .
		alternative = term { term } .
		term        = name | token [ "..." token ] | group | option | repetition .
		group       = "(" expression ")" .
		option      = "[" expression "]" .
		repetition  = "{" expression "}" .

		digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		special = "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "_" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "<" | ">" | "~" .

		name = ( small | caps ) { small | caps | digit | "_" } .
		token = "\"" { small | caps | digit | special } "\"" .
		} "Some comment"`,
	}

	tests = []string{
		"a1a3a4a4a5a6",
		"a1 a2a6",
		"a1 a3 a4 a6",
		"a1 a4 a5 a6",
		"a1 a2 a4 a5 a5 a6",
		"a1 a2 a4 a5 a6 a7",
		"your ad here",
		"2",
		"2*3 + 4/23 - 7",
		"(3 + 4) * 6-2+(4*(4))",
		"-2",
		"3 +",
		"(4 + 3",
		`{ }`,
		`"EBNF of EBNF" {
		program = [ token ] "{" { production } "}" [ token ] .
		production  = name "=" [ expression ] ( "." | ";" ) .
		expression  = alternative { "|" alternative } .
		alternative = term { term } .
		term        = name | token [ "..." token ] | group | option | repetition .
		group       = "(" expression ")" .
		option      = "[" expression "]" .
		repetition  = "{" expression "}" .

		digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		special = "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "_" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "<" | ">" | "~" .

		name = ( small | caps ) { small | caps | digit | "_" } .
		token = "\"" { small | caps | digit | special } "\"" .
		} "Some comment"`,
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
			fmt.Printf("%v", err)
			continue
		}
		fmt.Println()
		fmt.Println("tests:")
		for _, srcCode := range tests {
			// uses the grammar to parse a new type of code
			// fmt.Println(srcCode)
			res, err := ebnf.ParseWithGrammar(grammar, srcCode)
			if err != nil {
				fmt.Printf("%v", err)
			}
			fmt.Printf("%q: %v\n", srcCode, res)
		}
		fmt.Println()
	}

}
