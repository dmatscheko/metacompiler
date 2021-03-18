package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"./abnf"
)

// TODO: Maybe use the system of the default go EBNF parser with classes instead of r.Rule. This would be one less value to store (but is implicitly stored anyway).
// TODO: Add possibility to comment the EBNF via '//'.
// TODO: Allow to state the start rule via JS.
// TODO: Define an EOF symbol for the EBNF syntax.
// TODO: Add ability to include ABNFs from ABNFs (like modules).

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
// Term        = name | Range | Group | Option | Repetition | skipspaces ;
// Range       = token [ "..." token ] ;
// Group       = "(" Expression ")" ;
// Option      = "[" Expression "]" ;
// Repetition  = "{" Expression "}" ;
// Title       = token ;
// Comment     = token ;

// }
// EBNF

// ==========================================

// "EBNF of ABNF" {

// ABNF       = [ Title ] [ Tag ] "{" { Production } "}" [ Tag ] [ Comment ] ;
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

// This is the default main process:
// parse(initial-a-grammar, inputA)  = inputA-ASG -->  compile(inputA-ASG)  = new-a-grammar -->  parse(new-a-grammar, inputB)  = inputB-ASG -->  compile(inputB-ASG)  = result
func main() {
	param_abnf := flag.String("a", "", "The path of the ABNF file")
	param_srcCode := flag.String("b", "", "The path of the file to process")

	param_useBlockList := flag.Bool("lb", false, "Block list. Prevent a second execution of the same rule at the same position (slow)")
	param_useFoundList := flag.Bool("lf", false, "Found list. Caches all found blocks even if the sourrounding does not match. Immediately return the found block if the same rule would be applied again at the same place (very slow)")

	param_verbose_1 := flag.Bool("v1", false, "Show verbose output for step one. The a-grammar parser, parsing the ABNF from file -a to an ASG")
	param_verbose_2 := flag.Bool("v2", false, "Show verbose output for step two. The ASG compiler, compiling the ASG generated in step one to an a-grammar")
	param_verbose_3 := flag.Bool("v3", false, "Show verbose output for step three. The a-grammar parser, parsing the target file -b by applying the in step two generated a-grammar")
	param_verbose_4 := flag.Bool("v4", false, "Show verbose output for step four. The ASG compiler, compiling the ASG generated in step three")
	param_verbose_All := flag.Bool("v", false, "Show all verbose output")

	param_trace_1 := flag.Bool("vv1", false, "Show trace output for step one")
	param_trace_2 := flag.Bool("vv2", false, "Show trace output for step two")
	param_trace_3 := flag.Bool("vv3", false, "Show trace output for step three")
	param_trace_4 := flag.Bool("vv4", false, "Show trace output for step four")
	param_trace_All := flag.Bool("vv", false, "Show all trace output")

	param_speedTest := flag.Bool("s", false, "Run speed test with 10 cycles")

	flag.Parse()

	if *param_abnf == "" {
		flag.Usage()
		return
	}

	dat, err := ioutil.ReadFile(*param_abnf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return
	}
	aEbnf := string(dat)

	srcCode := ""
	if *param_srcCode != "" {
		dat, err = ioutil.ReadFile(*param_srcCode)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err)
			return
		}
		srcCode = string(dat)
	}

	if *param_speedTest {
		speedtest(aEbnf, srcCode, 100, *param_useBlockList, *param_useFoundList)
		return
	}

	if *param_verbose_All {
		*param_verbose_1 = true
		*param_verbose_2 = true
		*param_verbose_3 = true
		*param_verbose_4 = true
	}

	if *param_trace_All {
		*param_trace_1 = true
		*param_trace_2 = true
		*param_trace_3 = true
		*param_trace_4 = true
	}

	// *param_useBlockList := false
	// *param_useFoundList := false

	// MAIN PROCESS ----------------------------------------------------------------------------------------------

	// Use the initial a-grammar to parse an ABNF. It generates an ASG (abstract semantic graph) of the ABNF.
	fmt.Fprintln(os.Stderr, "Parse source ABNF file with initial a-grammar")
	asg, err := abnf.ParseWithAgrammar(abnf.AbnfAgrammar, aEbnf, *param_useBlockList, *param_useFoundList, *param_trace_1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprintln(os.Stderr, "  ==> Success, generated abstract semantic graph (ASG)")
	if *param_verbose_1 || *param_trace_1 {
		fmt.Fprintln(os.Stderr, "   => ASG: ", asg.Serialize(), "\n")
	}

	// Use the annotations inside the ASG to compile it. This should generate a new a-grammar.
	fmt.Fprintln(os.Stderr, "Compile ASG of source ABNF")
	aGrammar, err := abnf.CompileASG(asg, abnf.AbnfAgrammar, 0, *param_trace_2, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if aGrammar == nil { // There should be a generated a-grammar
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, "Did not receive a valid a-grammar from compiler")
		return
	}
	fmt.Fprintln(os.Stderr, " ==> Success, received an a-grammar from compiler")
	if *param_verbose_2 || *param_trace_2 {
		fmt.Fprintln(os.Stderr, "   => a-grammar: ", aGrammar.Serialize(), "\n")
	}

	// Use the a-grammar to parse the text it describes. It generates the ASG (abstract semantic graph) of the parsed text.
	fmt.Fprintln(os.Stderr, "Parse target file with new a-grammar")
	asg, err = abnf.ParseWithAgrammar(aGrammar, srcCode, *param_useBlockList, *param_useFoundList, *param_trace_3)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprintln(os.Stderr, "  ==> Success, generated abstract semantic graph (ASG)")
	if *param_verbose_3 || *param_trace_3 {
		fmt.Fprintln(os.Stderr, "   => ASG: ", asg.Serialize(), "\n")
	}

	// Use the annotations inside the ASG to compile it.
	fmt.Fprintln(os.Stderr, "Compile ASG")
	result, err := abnf.CompileASG(asg, aGrammar, 0, *param_trace_4, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprintln(os.Stderr, " ==> Success")
	if *param_verbose_4 || *param_trace_4 {
		if result != nil {
			fmt.Fprintln(os.Stderr, "   => Result: ", asg.Serialize(), "\n")
		}
	}
}

func speedtest(src, target string, count int, useBlockList, useFoundList bool) {
	speedtestParseWithGrammar(src, target, count, useBlockList, useFoundList)
	speedtestCompileASG(src, target, count, useBlockList, useFoundList)
	fmt.Fprintln(os.Stderr)
}
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "%s took %s\n", name, elapsed)
}

func speedtestParseWithGrammar(src, target string, count int, useBlockList, useFoundList bool) {
	var err error
	defer timeTrack(time.Now(), "ParseWithGrammar")
	for i := 0; i < count; i++ {
		_, err = abnf.ParseWithAgrammar(abnf.AbnfAgrammar, target, useBlockList, useFoundList, false)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseWithGrammar")
		return
	}
}
func speedtestCompileASG(src, target string, count int, useBlockList, useFoundList bool) {
	asg, err := abnf.ParseWithAgrammar(abnf.AbnfAgrammar, target, useBlockList, useFoundList, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseWithGrammar")
		return
	}
	defer timeTrack(time.Now(), "CompileASG")
	for i := 0; i < count; i++ {
		_, err = abnf.CompileASG(asg, abnf.AbnfAgrammar, 0, false, true)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error CompileASG")
		return
	}
}
