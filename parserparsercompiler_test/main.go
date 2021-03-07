package main

import (
	"fmt"

	"./ebnf"
)

// TODO: add possibility to comment the ebnf via //

// PUBLIC EBNF of EBNF:
// 	  Production  = name "=" [ Expression ] "." .
//    Expression  = Alternative { "|" Alternative } .
//    Alternative = Term { Term } .
//    Term        = name | token [ "…" token ] | Group | Option | Repetition .
//    Group       = "(" Expression ")" .
//    Option      = "[" Expression "]" .
//    Repetition  = "{" Expression "}" .

var (
	ebnfs = []string{

		// ==========================================

		// EBNF of aEBNF with tags (ORIGINAL):
		// `"EBNF of aEBNF" {
		// program = [ title ] [ tag ] "{" { production } "}" [ tag ] [ comment ] ;
		// production  = name [ tag ] "=" [ expression ] ( "." | ";" ) ;
		// expression  = sequence ;
		// sequence    = alternative { alternative } ;
		// alternative = term { "|" term } ;
		// term        = ( name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag ] ;
		// group       = "(" expression  ")" ;
		// option      = "[" expression "]" ;
		// repetition  = "{" expression "}" ;
		// skipspaces  = "+" | "-" ;

		// title = text ;
		// comment = text ;

		// name = ( small | caps ) - { small | caps | digit | "_" } + ;
		// tag  = "<" text text ">" .

		// text        = dquotetext | squotetext ;

		// dquotetext = '"' - { small | caps | digit | special | "'" | '\\"' } '"' + ;
		// squotetext = "'" - { small | caps | digit | special | '"' | "\\'" } "'" + ;

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// special = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "~" | "@" ;

		// } "Some comment"`,

		// ==========================================

		// Minimal EBNF of EBNF (can NOT parse) (sequence | alternative can both not be together with the rest of the expression alternatives, but why? <- sequence and alternative can not have itself inside):
		// `"Minimal EBNF of EBNF" {
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

		// ==========================================

		`"aEBNF of aEBNF as text"
		<~~
		var names = [];
		function getNameIdx(name) {
		  var pos = names.indexOf(name);
		  if (pos != -1) { return pos };
		  return names.push(name)-1;
		}
		~~>
		
		{
		
		program          = [ title <~~ upstream.str = upstream.str+', \\n' ~~> ] [ tag <~~ upstream.str = upstream.str+', \\n' ~~> ] "{" <~~ upstream.str = '\\n{\\n\\n' ~~> { production } "}" <~~ upstream.str = '\\n}, \\n\\n' ~~> [ tag <~~ upstream.str = upstream.str+', \\n' ~~> ] start [ comment <~~ upstream.str = ', \\n'+upstream.str+'\\n' ~~> ] ;
		production       = name <~~ upstream.str = '{"'+upstream.str+'", '+getNameIdx(upstream.str)+', ' ~~> [ tag ] "=" <~~ upstream.str = '' ~~> [ expression ] ( "." | ";" ) <~~ upstream.str = '}, \\n' ~~> ;
		expression       <~~ if (upstream.or) { upstream.str = '{"OR", '+upstream.str+'}' } ~~>   = alternative <~~ upstream.or = false ~~> { "|" <~~ upstream.str = '' ~~> alternative <~~ upstream.or = true; upstream.str = ', '+upstream.str ~~> } ;
		alternative      = term { term <~~ upstream.str = ', '+upstream.str ~~> } ;
		term             = ( name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag <~~ upstream.str = ', '+upstream.str ~~> ] ;
		group            <~~ upstream.str = '{'+upstream.str+'}' ~~>                              = "(" <~~ upstream.str = '' ~~> expression ")" <~~ upstream.str = '' ~~> ;
		option           <~~ upstream.str = '{"OPTIONAL", '+upstream.str+'}' ~~>                  = "[" <~~ upstream.str = '' ~~> expression "]" <~~ upstream.str = '' ~~> ;
		repetition       <~~ upstream.str = '{"REPEAT", '+upstream.str+'}' ~~>                    = "{" <~~ upstream.str = '' ~~> expression "}" <~~ upstream.str = '' ~~> ;
		skipspaces       = "+" <~~ upstream.str = '{"SKIPSPACES", true}' ~~> | "-" <~~ upstream.str = '{"SKIPSPACES", false}' ~~> ;
		
		title            = text ;
		start            = name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> ;
		comment          = text ;
		
		tag <~~ upstream.str = '{"TAG", '+upstream.str+'}' ~~>                                    = "<" <~~ upstream.str = '' ~~> code { "," <~~ upstream.str = '' ~~> code <~~ upstream.str = ', '+upstream.str ~~> } ">" <~~ upstream.str = '' ~~> ;
		
		code             <~~ upstream.str = '{"TERMINAL", '+upstream.str+'}' ~~>                  = '~~' - { [ "~" ] codeinner } '~~' + ;
		codeinner        = small | caps | digit | special | "'" | '"' | "\\~" ;
		
		name             = ( small | caps ) - { small | caps | digit | "_" } + ;
		
		text             <~~ upstream.str = '{"TERMINAL", '+upstream.str+'}' ~~>                  = dquotetext | squotetext ;
		dquotetext       = '"' - { small | caps | digit | special | "~" | "'" | '\\"' } '"' + ;
		squotetext       = "'" - { small | caps | digit | special | "~" | '"' | "\\'" } "'" + ;
		
		digit            = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		small            = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		caps             = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		special          = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "@" ;
		
		}
		
		<~~ print(upstream.str) ~~>
		program
		"This aEBNF contains the grammatic and semantic information for annotated EBNF.
		It allows to automatically create a compiler for everything described in aEBNF (yes, that format)."`,
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

		// ==========================================

		`"aEBNF of aEBNF as text"
		<~~
		var names = [];
		function getNameIdx(name) {
		  var pos = names.indexOf(name);
		  if (pos != -1) { return pos };
		  return names.push(name)-1;
		}
		~~>
		
		{
		
		program          = [ title <~~ upstream.str = upstream.str+', \\n' ~~> ] [ tag <~~ upstream.str = upstream.str+', \\n' ~~> ] "{" <~~ upstream.str = '\\n{\\n\\n' ~~> { production } "}" <~~ upstream.str = '\\n}, \\n\\n' ~~> [ tag <~~ upstream.str = upstream.str+', \\n' ~~> ] start [ comment <~~ upstream.str = ', \\n'+upstream.str+'\\n' ~~> ] ;
		production       = name <~~ upstream.str = '{"'+upstream.str+'", '+getNameIdx(upstream.str)+', ' ~~> [ tag ] "=" <~~ upstream.str = '' ~~> [ expression ] ( "." | ";" ) <~~ upstream.str = '}, \\n' ~~> ;
		expression       <~~ if (upstream.or) { upstream.str = '{"OR", '+upstream.str+'}' } ~~>   = alternative <~~ upstream.or = false ~~> { "|" <~~ upstream.str = '' ~~> alternative <~~ upstream.or = true; upstream.str = ', '+upstream.str ~~> } ;
		alternative      = term { term <~~ upstream.str = ', '+upstream.str ~~> } ;
		term             = ( name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag <~~ upstream.str = ', '+upstream.str ~~> ] ;
		group            <~~ upstream.str = '{'+upstream.str+'}' ~~>                              = "(" <~~ upstream.str = '' ~~> expression ")" <~~ upstream.str = '' ~~> ;
		option           <~~ upstream.str = '{"OPTIONAL", '+upstream.str+'}' ~~>                  = "[" <~~ upstream.str = '' ~~> expression "]" <~~ upstream.str = '' ~~> ;
		repetition       <~~ upstream.str = '{"REPEAT", '+upstream.str+'}' ~~>                    = "{" <~~ upstream.str = '' ~~> expression "}" <~~ upstream.str = '' ~~> ;
		skipspaces       = "+" <~~ upstream.str = '{"SKIPSPACES", true}' ~~> | "-" <~~ upstream.str = '{"SKIPSPACES", false}' ~~> ;
		
		title            = text ;
		start            = name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> ;
		comment          = text ;
		
		tag <~~ upstream.str = '{"TAG", '+upstream.str+'}' ~~>                                    = "<" <~~ upstream.str = '' ~~> code { "," <~~ upstream.str = '' ~~> code <~~ upstream.str = ', '+upstream.str ~~> } ">" <~~ upstream.str = '' ~~> ;
		
		code             <~~ upstream.str = '{"TERMINAL", '+upstream.str+'}' ~~>                  = '~~' - { [ "~" ] codeinner } '~~' + ;
		codeinner        = small | caps | digit | special | "'" | '"' | "\\~" ;
		
		name             = ( small | caps ) - { small | caps | digit | "_" } + ;
		
		text             <~~ upstream.str = '{"TERMINAL", '+upstream.str+'}' ~~>                  = dquotetext | squotetext ;
		dquotetext       = '"' - { small | caps | digit | special | "~" | "'" | '\\"' } '"' + ;
		squotetext       = "'" - { small | caps | digit | special | "~" | '"' | "\\'" } "'" + ;
		
		digit            = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		small            = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		caps             = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		special          = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "@" ;
		
		}
		
		<~~ print(upstream.str) ~~>
		program
		"This aEBNF contains the grammatic and semantic information for annotated EBNF.
		It allows to automatically create a compiler for everything described in aEBNF (yes, that format)."`,

		// ==========================================

		// `"EBNF of EBNF (can parse)" {
		// 	program = [ title ] "{" { production } "}" [ comment ] ;
		// 	production  = name "=" [ expression ] ";" ;
		// 	expression  = sequence ;
		// 	sequence    = alternative { alternative } ;
		// 	alternative = term { "|" term } ;
		// 	term        = name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ;
		// 	group       <~~ print("test") ~~>      = "(" expression ")" ;
		// 	option      = "[" expression "]" ;
		// 	repetition  = "{" expression "}" ;
		// 	skipspaces = "+" | "-" ;
		// 	title = text ;
		// 	comment = text ;
		// 	name = ( small | caps ) { small | caps | digit | "_" } ;
		// 	text = "\"" - { small | caps | digit | special } "\"" + ;
		// 	digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// 	small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// 	caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// 	special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" ;
		// 	} program "foo"`,
	}
)

func main() {
	// speedtest()
	// return

	for _, srcEBNF := range ebnfs {
		fmt.Print("\n===========================================================\nEBNF:\n===========================================================\n\n")
		ebnf.PprintSrc("Parse", srcEBNF)
		// Parses an EBNF and generates a grammar with it.
		grammar, err := ebnf.ParseEBNF(srcEBNF)
		if err != nil {
			fmt.Println("  ==> Fail")
			fmt.Println(err)
			continue
		}
		fmt.Println("  ==> Success\n\n  Grammar:")
		fmt.Println("   => Extras: " + ebnf.PprintExtrasShort(&grammar.Extras, "    "))
		fmt.Println("   => Productions: " + ebnf.PprintProductionsShort(&grammar.Productions, "    "))

		fmt.Print("\n\n==================\nTests:\n==================\n\n")
		for _, srcCode := range tests {
			fmt.Println("Parse via grammar:")
			ebnf.PprintSrcSingleLine(srcCode)
			// Uses the grammar to parse the by it described text. It generates the ASG (abstract semantic graph) of the parsed text.
			asg, err := ebnf.ParseWithGrammar(grammar, srcCode, false)
			if err != nil {
				fmt.Println("\n  ==> Fail")
				fmt.Println(err)
				continue
			}
			fmt.Println("\n  ==> Success\n\n  Abstract semantic graph:")
			fmt.Println("    " + ebnf.PprintProductionsShort(&asg, "    "))

			fmt.Println("\nCode output:")
			// Uses the annotations inside the ASG to compile it.
			upstream, err := ebnf.CompileASG(asg, &grammar.Extras, false)
			if err != nil {
				fmt.Println("\n  ==> Fail")
				fmt.Println(err)
				continue
			}
			fmt.Print("\n ==> Success\n\n")
			fmt.Printf("  Upstream Vars:\n    %#v\n\n", upstream)
		}
		fmt.Println()
	}

}

// func timeTrack(start time.Time, name string) {
// 	elapsed := time.Since(start)
// 	log.Printf("%s took %s", name, elapsed)
// }
// func speedtest() {
// 	src := `{
// 		program = [ title ] "{" { production } "}" [ comment ] ;
// 		production  = name "=" [ expression ] ";" ;
// 		expression  = sequence ;
// 		sequence    = alternative { alternative } ;
// 		alternative = term { "|" term } ;
// 		term        = name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ;
// 		group       = "(" expression ")" ;
// 		option      = "[" expression "]" ;
// 		repetition  = "{" expression "}" ;
// 		skipspaces = "+" | "-" ;
// 		title = text ;
// 		comment = text ;
// 		name = ( small | caps ) { small | caps | digit | "_" } ;
// 		text = "\"" - { small | caps | digit | special } "\"" + ;
// 		digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
// 		small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
// 		caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
// 		special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" ;
// 		} program`

// 	defer timeTrack(time.Now(), "parse DMA")
// 	var err error = nil
// 	for i := 0; i < 10000; i++ {
// 		_, err = ebnf.ParseEBNF(src)
// 	}
// 	if err != nil {
// 		fmt.Println("Error")
// 		return
// 	}
// 	// fmt.Printf("%#v", g)
// }
