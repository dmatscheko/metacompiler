package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"./ebnf"
)

// TODO: MAybe use the system of the default go EBNF parser with classes instead of r.Rule. This would be one value less.
// TODO: Add possibility to comment the EBNF via '//'.
// TODO: Allow to state the start rule via JS.
// TODO: Define an EOF symbol for the EBNF syntax.
// TODO: Add ability to include a-EBNFs from a-EBNFs (like modules).

// rule == production.
// factors == non-terminal expression. A subgroup of productions/rules.
// link == ident == name             //  <=  Identifies another rule (== address of the other rule).
// string == token == terminal == text.
// or == alternative.

// ==========================================

// "EBNF of EBNF" {

// EBNF        = [ Title ] "{" { Production } "}" [ Comment ] ;
// Production  = name "=" [ Expression ] ";" ;
// Expression  = Alternative { "|" Alternative } ;
// Alternative = Term { Term } ;
// Term        = name | token [ "..." token ] | Group | Option | Repetition | skipspaces ;
// Group       = "(" Expression ")" ;
// Option      = "[" Expression "]" ;
// Repetition  = "{" Expression "}" ;
// Title       = token ;
// Comment     = token ;

// }
// EBNF

// ==========================================

// "EBNF of a-EBNF" {

// AEBNF       = [ Title ] [ Tag ] "{" { Production } "}" [ Tag ] [ Comment ] ;
// Production  = name [ Tag ] "=" [ Expression ] ";" ;
// Expression  = Alternative { "|" Alternative } ;
// Alternative = Term { Term } ;
// Term        = ( name | token [ "..." token ] | Group | Option | Repetition | skipspaces ) [ Tag ] ;
// Group       = "(" Expression ")" ;
// Option      = "[" Expression "]" ;
// Repetition  = "{" Expression "}" ;
// Title       = token ;
// Comment     = token ;
// Tag         = "<" code { "," code } ">" ;

// }
// AEBNF

// ==========================================

// "'Wrong' EBNF of EBNF (can NOT parse)" {

// EBNF        = [ Title ] "{" { Production } "}" name [ Comment ] ;
// Production  = name "=" [ Expression ] ";" ;
// Expression  = name | token [ "..." token ] | Group | Option | Repetition | skipspaces | Sequence | Alternative ;
// Sequence    = Expression Expression { Expression } ;
// Alternative = Expression "|" Expression { "|" Expression } ;
// Group       = "(" Expression ")" ;
// Option      = "[" Expression "]" ;
// Repetition  = "{" Expression "}" ;
// Title       = token ;
// Comment     = token ;

// }
// EBNF

// ==========================================

// "Common syntax" {

// name        = ( Small | Caps ) - { Small | Caps | Digit | "_" } + ;
// token       = Dquotetoken | Squotetoken ;

// Dquotetoken = '"' - { Small | Caps | Digit | Special | "~" | "'" | '\\"' } '"' + ;
// Squotetoken = "'" - { Small | Caps | Digit | Special | "~" | '"' | "\\'" } "'" + ;

// Digit       = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
// Small       = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" |
//               "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
// Caps        = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" |
//               "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
// Special     = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" |
//               "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "|" | "%" | "$" | "&" | "#" |
//               "@" | "\\\\" | "\\t" | "\t" | "\\n" | "\n" | "\\r" | "\r" ;

// skipspaces  = Skip | Noskip ;
// Skip        = "+" ;  // Skip all whitespace in the future.
// Noskip      = "-" ;  // Do not skip whitspace in the future.

// }

// ==========================================

func main() {
	param_aEbnf := flag.String("a", "", "The path of the a-EBNF file")
	param_srcCode := flag.String("b", "", "The path of the file to process")

	param_trace_ParseAEBNF := flag.Bool("te", false, "Show trace output for the a-EBNF parser")
	param_trace_ParseWithAGrammar := flag.Bool("tg", false, "Show trace output for the a-grammar parser")
	param_trace_CompileASG := flag.Bool("tc", false, "Show trace output for the ASG compiler")
	param_trace_All := flag.Bool("t", false, "Show all trace output")

	param_speedTest := flag.Bool("s", false, "Run speed test with 10 cycles")

	flag.Parse()

	if *param_aEbnf == "" || *param_srcCode == "" {
		flag.Usage()
		return
	}

	dat, err := ioutil.ReadFile(*param_aEbnf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return
	}
	aEbnf := string(dat)

	dat, err = ioutil.ReadFile(*param_srcCode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return
	}
	srcCode := string(dat)

	if *param_speedTest {
		speedtest(aEbnf, srcCode, 20)
		return
	}

	if *param_trace_All {
		*param_trace_ParseAEBNF = true
		*param_trace_ParseWithAGrammar = true
		*param_trace_CompileASG = true
	}

	fmt.Fprint(os.Stderr, "\n==================\nParse a-EBNF\n==================\n\n")
	fmt.Fprintln(os.Stderr, "a-EBNF:")
	fmt.Fprint(os.Stderr, ebnf.PprintSrc(aEbnf))
	// Parses an aEBNF and generates a a-grammar with it.
	aGrammar, err := ebnf.ParseAEBNF(aEbnf, *param_trace_ParseAEBNF)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprintln(os.Stderr, "  ==> Success\n\n  a-Grammar:")
	if *param_trace_ParseAEBNF {
		fmt.Fprintln(os.Stderr, "   => Extras: "+ebnf.PprintExtras(&aGrammar.Extras, "    "))
		fmt.Fprintln(os.Stderr, "   => Productions: "+ebnf.PprintProductions(&aGrammar.Productions, "    "))
	} else {
		fmt.Fprintln(os.Stderr, "   => Extras: "+ebnf.Shorten(ebnf.PprintExtras(&aGrammar.Extras, "    ")))
		fmt.Fprintln(os.Stderr, "   => Productions: "+ebnf.Shorten(ebnf.PprintProductions(&aGrammar.Productions, "    ")))
	}

	fmt.Fprint(os.Stderr, "\n\n==================\nParse target code\n==================\n\n")
	fmt.Fprintln(os.Stderr, "Parse via a-grammar:")
	fmt.Fprint(os.Stderr, ebnf.PprintSrcSingleLine(srcCode))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)
	// Uses the grammar to parse the by it described text. It generates the ASG (abstract semantic graph) of the parsed text.
	asg, err := ebnf.ParseWithGrammar(aGrammar, srcCode, false, *param_trace_ParseWithAGrammar)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\n  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprintln(os.Stderr, "\n  ==> Success\n\n  Abstract semantic graph:")
	if *param_trace_ParseWithAGrammar {
		fmt.Fprintln(os.Stderr, "    "+ebnf.PprintProductions(&asg, "    "))
	} else {
		// fmt.Fprintln(os.Stderr,"    " + ebnf.PprintProductions(&asg, "    "))
		fmt.Fprintln(os.Stderr, "    "+ebnf.Shorten(ebnf.PprintProductions(&asg, "    ")))
	}

	fmt.Fprint(os.Stderr, "\nCode output:\n\n")
	// Uses the annotations inside the ASG to compile it.
	_, err = ebnf.CompileASG(asg, &aGrammar.Extras, *param_trace_CompileASG, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\n  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprint(os.Stderr, "\n ==> Success\n\n")
	// tmpStr := fmt.Sprintf("  Upstream Vars:\n    %#v\n\n", up)

	// if !*param_trace_CompileASG {
	// 	tmpStr = ebnf.Shorten(tmpStr)
	// }

	// fmt.Fprint(os.Stderr,tmpStr)
	fmt.Fprintln(os.Stderr)
}

func speedtest(src, target string, count int) {
	speedtestParseAEBNF(src, target, count)
	speedtestParseWithGrammar(src, target, count)
	speedtestCompileASG(src, target, count)
}
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "%s took %s", name, elapsed)
}
func speedtestParseAEBNF(src, target string, count int) {
	defer timeTrack(time.Now(), "ParseAEBNF")
	var err error = nil
	for i := 0; i < count; i++ {
		_, err = ebnf.ParseAEBNF(src, false)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseAEBNF")
		return
	}
}
func speedtestParseWithGrammar(src, target string, count int) {
	aGrammar, err := ebnf.ParseAEBNF(src, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseAEBNF")
		return
	}
	defer timeTrack(time.Now(), "ParseWithGrammar")
	for i := 0; i < count; i++ {
		_, err = ebnf.ParseWithGrammar(aGrammar, target, false, false)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseWithGrammar")
		return
	}
}
func speedtestCompileASG(src, target string, count int) {
	aGrammar, err := ebnf.ParseAEBNF(src, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseAEBNF")
		return
	}
	asg, err := ebnf.ParseWithGrammar(aGrammar, target, false, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseWithGrammar")
		return
	}
	defer timeTrack(time.Now(), "CompileASG")
	for i := 0; i < count; i++ {
		_, err = ebnf.CompileASG(asg, &aGrammar.Extras, false, true)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error CompileASG")
		return
	}
}
