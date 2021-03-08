package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"./ebnf"
)

// TODO: Add possibility to comment the EBNF via //
// TODO: Allow to state the start rule via JS
// TODO: define an EOF symbol for the EBNF syntax

// ==========================================

// public EBNF of EBNF:
// 	  Production  = name "=" [ Expression ] "." .
//    Expression  = Alternative { "|" Alternative } .
//    Alternative = Term { Term } .
//    Term        = name | token [ "..." token ] | Group | Option | Repetition .
//    Group       = "(" Expression ")" .
//    Option      = "[" Expression "]" .
//    Repetition  = "{" Expression "}" .

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

// `"aEBNF of aEBNF as text"
// <~~
// var names = [];
// function getNameIdx(name) {
//   var pos = names.indexOf(name);
//   if (pos != -1) { return pos };
//   return names.push(name)-1;
// }
// c.compile(c.asg, upstream)
// ~~>

// {

// program          = [ title <~~ upstream.str += ', \\n' ~~> ] programtag "{" <~~ upstream.str = '\\n{\\n\\n' ~~> { production } "}" <~~ upstream.str = '\\n}, \\n\\n' ~~> programtag start [ comment <~~ upstream.str = ', \\n'+upstream.str+'\\n' ~~> ] ;
// programtag       = [ tag <~~ upstream.str = '{"TAG", '+upstream.str+'}, \\n' ~~> ] ;
// production       <~~ if (upstream.productionTag != undefined) { upstream.str = '{"TAG", '+upstream.productionTag+', '+upstream.str+'}' }; upstream.str += ', \\n' ~~>
// 				 = name <~~ upstream.str = '{"'+upstream.str+'", '+getNameIdx(upstream.str)+', ' ~~> [ tag ] <~~ upstream.productionTag = upstream.str; upstream.str = '' ~~> "=" <~~ upstream.str = '' ~~> [ expression ] ( "." | ";" ) <~~ upstream.str = '}' ~~> ;
// expression       <~~ if (upstream.or) { upstream.str = '{"OR", '+upstream.str+'}' } ~~>
// 				 = alternative <~~ upstream.or = false ~~> { "|" <~~ upstream.str = '' ~~> alternative <~~ upstream.or = true; upstream.str = ', '+upstream.str ~~> } ;
// alternative      = taggedterm { taggedterm <~~ upstream.str = ', '+upstream.str ~~> } ;

// taggedterm       <~~ if (upstream.termTag != undefined) { upstream.str = '{"TAG", '+upstream.termTag+', '+upstream.str+'}' } ~~>
// 				 = term <~~ upstream.termTag = undefined ~~> [ tag <~~ upstream.termTag = upstream.str; upstream.str = '' ~~> ] ;

// term             = ( name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) ;
// group            <~~ upstream.str = '{'+upstream.str+'}' ~~>                              = "(" <~~ upstream.str = '' ~~> expression ")" <~~ upstream.str = '' ~~> ;
// option           <~~ upstream.str = '{"OPTIONAL", '+upstream.str+'}' ~~>                  = "[" <~~ upstream.str = '' ~~> expression "]" <~~ upstream.str = '' ~~> ;
// repetition       <~~ upstream.str = '{"REPEAT", '+upstream.str+'}' ~~>                    = "{" <~~ upstream.str = '' ~~> expression "}" <~~ upstream.str = '' ~~> ;
// skipspaces       = "+" <~~ upstream.str = '{"SKIPSPACES", true}' ~~> | "-" <~~ upstream.str = '{"SKIPSPACES", false}' ~~> ;

// title            = text ;
// start            = name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> ;
// comment          = text ;

// tag              = "<" <~~ upstream.str = '' ~~> code { "," <~~ upstream.str = '' ~~> code <~~ upstream.str = ', '+upstream.str ~~> } ">" <~~ upstream.str = '' ~~> ;

// code             <~~ upstream.str = '{"TERMINAL", '+sprintf("%q",upstream.str)+'}' ~~>                  = '~~' <~~ upstream.str = '' ~~> - { [ "~" ] codeinner } '~~' <~~ upstream.str = '' ~~> + ;
// codeinner        = small | caps | digit | special | "'" | '"' | "\\~" ;

// name             = ( small | caps ) - { small | caps | digit | "_" } + ;

// text             <~~ upstream.str = '{"TERMINAL", '+sprintf("%q",upstream.str)+'}' ~~>                  = dquotetext | squotetext ;
// dquotetext       = '"' <~~ upstream.str = '' ~~> - { small | caps | digit | special | "~" | "'" | '\\"' } '"' <~~ upstream.str = '' ~~> + ;
// squotetext       = "'" <~~ upstream.str = '' ~~> - { small | caps | digit | special | "~" | '"' | "\\'" } "'" <~~ upstream.str = '' ~~> + ;

// digit            = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
// small            = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
// caps             = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
// special          = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "@" ;

// }

// <~~ print(upstream.str)

// println("\\n\\nTHE FOLLOWING IS NOT FROM THE ABOVE CODE BUT ONLY A TEST")

// println("\\nLLVM IR test:\\n------------------------------------")

// // Create convenience types and constants.
// var i32 = llvm.types.I32

// // Create a new LLVM IR module.
// var m = llvm.ir.NewModule()

// // -----------------

// var zero = llvm.constant.NewInt(i32, 0)
// var a = llvm.constant.NewInt(i32, 0x15A4E35) // multiplier of the PRNG.
// var c = llvm.constant.NewInt(i32, 1)         // increment of the PRNG.

// // Create an external function declaration and append it to the module.
// //
// //    int abs(int x);
// var abs = m.NewFunc("abs", i32, llvm.ir.NewParam("x", i32))

// // Create a global variable definition and append it to the module.
// //
// //    int seed = 0;
// var seed = m.NewGlobalDef("seed", zero)

// // Create a function definition and append it to the module.
// //
// //    int rand(void) { ... }
// var rand = m.NewFunc("rand", i32)

// // Create an unnamed entry basic block and append it to the 'rand' function.
// var entry = rand.NewBlock("")

// // Create instructions and append them to the entry basic block.
// var tmp1 = entry.NewLoad(i32, seed)
// var tmp2 = entry.NewMul(tmp1, a)
// var tmp3 = entry.NewAdd(tmp2, c)
// entry.NewStore(tmp3, seed)
// var tmp4 = entry.NewCall(abs, tmp3)
// entry.NewRet(tmp4)

// // -----------------

// // int test() { ... }
// var test = m.NewFunc("test", i32)

// // Create an unnamed entry basic block and append it to the 'test' function.
// var entry = test.NewBlock("")
// // Create instructions and append them to the entry basic block.

// // %3 = add
// var tmp = entry.NewAdd(llvm.constant.NewInt(i32, 32), llvm.constant.NewInt(i32, 32))

// // ret i32 %3
// entry.NewRet(tmp)

// // -----------------

// // Print the LLVM IR assembly of the module.
// println(m)

// println("GRAPH:\\n------------------------------------")
// println(llvm.Callgraph(m))

// println("\\nEVAL:\\n------------------------------------")
// println(llvm.Eval(m, "test"))

// ~~>
// program
// "This aEBNF contains the grammatic and semantic information for annotated EBNF.
// It allows to automatically create a compiler for everything described in aEBNF (yes, that format)."`,

// ==========================================

func main() {
	// speedtest()
	// return

	param_aEbnf := flag.String("f", "", "The path of the a-EBNF")
	param_srcCode := flag.String("s", "", "The a-EBNF gets applied to this file")

	param_trace_ParseAEBNF := flag.Bool("te", false, "Show trace output for the a-EBNF parser")
	param_trace_ParseWithAGrammar := flag.Bool("tg", false, "Show trace output for the a-grammar parser")
	param_trace_CompileASG := flag.Bool("tc", false, "Show trace output for the ASG compiler")
	param_trace_All := flag.Bool("t", false, "Show all trace output")

	flag.Parse()

	if *param_aEbnf == "" || *param_srcCode == "" {
		flag.Usage()
		return
	}

	dat, err := ioutil.ReadFile(*param_aEbnf)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	aEbnf := string(dat)

	dat, err = ioutil.ReadFile(*param_srcCode)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	srcCode := string(dat)

	if *param_trace_All {
		*param_trace_ParseAEBNF = true
		*param_trace_ParseWithAGrammar = true
		*param_trace_CompileASG = true
	}

	fmt.Print("\n==================\nParse a-EBNF\n==================\n\n")
	ebnf.PprintSrc("a-EBNF", aEbnf)
	// Parses an aEBNF and generates a a-grammar with it.
	aGrammar, err := ebnf.ParseAEBNF(aEbnf, *param_trace_ParseAEBNF)
	if err != nil {
		fmt.Println("  ==> Fail")
		fmt.Println(err)
		return
	}
	fmt.Println("  ==> Success\n\n  a-Grammar:")
	if *param_trace_ParseAEBNF {
		fmt.Println("   => Extras: " + ebnf.PprintExtras(&aGrammar.Extras, "    "))
		fmt.Println("   => Productions: " + ebnf.PprintProductions(&aGrammar.Productions, "    "))
	} else {
		fmt.Println("   => Extras: " + ebnf.PprintExtrasShort(&aGrammar.Extras, "    "))
		fmt.Println("   => Productions: " + ebnf.PprintProductionsShort(&aGrammar.Productions, "    "))
	}

	fmt.Print("\n\n==================\nParse target code\n==================\n\n")
	fmt.Println("Parse via a-grammar:")
	ebnf.PprintSrcSingleLine(srcCode)
	fmt.Println()
	fmt.Println()
	// Uses the grammar to parse the by it described text. It generates the ASG (abstract semantic graph) of the parsed text.
	asg, err := ebnf.ParseWithGrammar(aGrammar, srcCode, *param_trace_ParseWithAGrammar)
	if err != nil {
		fmt.Println("\n  ==> Fail")
		fmt.Println(err)
		return
	}
	fmt.Println("\n  ==> Success\n\n  Abstract semantic graph:")
	if *param_trace_ParseWithAGrammar {
		fmt.Println("    " + ebnf.PprintProductions(&asg, "    "))
	} else {
		fmt.Println("    " + ebnf.PprintProductionsShort(&asg, "    "))
	}

	fmt.Println("\nCode output:")
	// Uses the annotations inside the ASG to compile it.
	upstream, err := ebnf.CompileASG(asg, &aGrammar.Extras, *param_trace_CompileASG)
	if err != nil {
		fmt.Println("\n  ==> Fail")
		fmt.Println(err)
		return
	}
	fmt.Print("\n ==> Success\n\n")
	tmpStr := fmt.Sprintf("  Upstream Vars:\n    %#v\n\n", upstream)

	if !*param_trace_CompileASG {
		if len(tmpStr) > 1000 {
			tmpStr = tmpStr[:1000] + " ..."
		}
	}

	fmt.Print(tmpStr)
	fmt.Println()
}

// func timeTrack(start time.Time, name string) {
// 	elapsed := time.Since(start)
// 	log.Printf("%s took %s", name, elapsed)
// }
// func speedtest() {
// 	src := `... fill in! ...`

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
