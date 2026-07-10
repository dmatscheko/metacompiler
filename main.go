package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"14.gy/mec/abnf"
)

// The first option(s) of an OR must not be the beginning of a later option. Otherwise only the first option would be found and success would be returned.
// No option of a production can be equal to the production itself. This means that e.g. the production 'Test' must not have an option 'Test'.

// TODO: Maybe use the system of the default go EBNF parser with classes instead of r.Rule. This would be one less value to store (but is implicitly stored anyway).
// TODO: Define an :EOF() symbol for the EBNF syntax.
// An unknown Name still resolves to the -1 marker at parse time instead of a clear error up front; the -verify flag (abnf/verifier.go) reports such names (also those used as command parameters or in tags) before a run.
// TODO: add -c -d , ... to the cmd line

// rule == production | expression | term.
// link == identifier == name             //  <=  Identifies another rule (== position of the other rule inside the grammar rules array).
// string == token == terminal == text.
// or == alternative.

// This is the default main process:
// parse(initial-a-grammar, inputA)  = inputA-ASG -->  compile(inputA-ASG)  = new-a-grammar -->  parse(new-a-grammar, inputB)  = inputB-ASG -->  compile(inputB-ASG)  = result
func main() {
	param_a := flag.String("a", "", "The path of the ABNF file")
	param_b := flag.String("b", "", "The path of the file to process with the a-grammar from file -a")
	param_c := flag.String("c", "", "The path of the file to process with the a-grammar from file -b")

	param_slot_b := flag.Int("sb", 0, "The tag slot to use when compiling file b with the a-grammar from file -a (default is 0)")
	param_slot_c := flag.Int("sc", 0, "The tag slot to use when compiling file c with the a-grammar from file -b (default is 0)")

	param_useBlockList := flag.Bool("lb", false, "Block list. Prevent a second execution of the same rule at the same position (slow)")
	param_useFoundList := flag.Bool("lf", false, "Found list. Caches all found blocks even if the surrounding does not match. Immediately return the found block if the same rule would be applied again at the same place (very slow)")

	param_verbose_Ap := flag.Bool("va1", false, "Show verbose output for step one. The a-grammar parser, parsing the ABNF from file -a to an ASG")
	param_verbose_Ac := flag.Bool("va2", false, "Show verbose output for step two. The ASG compiler, compiling the ASG generated in step one to an a-grammar")
	param_verbose_Bp := flag.Bool("vb1", false, "Show verbose output for step three. The a-grammar parser, parsing the target file -b by applying the in step two generated a-grammar")
	param_verbose_Bc := flag.Bool("vb2", false, "Show verbose output for step four. The ASG compiler, compiling the ASG generated in step three")
	param_verbose_Cp := flag.Bool("vc1", false, "Show verbose output for step five. The a-grammar parser, parsing the target file -c by applying the in step four generated a-grammar")
	param_verbose_Cc := flag.Bool("vc2", false, "Show verbose output for step six. The ASG compiler, compiling the ASG generated in step five")
	param_verbose_All := flag.Bool("v", false, "Show all verbose output")

	param_trace_Ap := flag.Bool("vva1", false, "Show trace output for step one")
	param_trace_Ac := flag.Bool("vva2", false, "Show trace output for step two")
	param_trace_Bp := flag.Bool("vvb1", false, "Show trace output for step three")
	param_trace_Bc := flag.Bool("vvb2", false, "Show trace output for step four")
	param_trace_Cp := flag.Bool("vvc1", false, "Show trace output for step five")
	param_trace_Cc := flag.Bool("vvc2", false, "Show trace output for step six")
	param_trace_All := flag.Bool("vv", false, "Show all trace output")

	param_quiet_Most := flag.Bool("q", false, "Show only JS output or errors")
	param_quiet_Full := flag.Bool("qq", false, "Show only errors, hide JS output")

	param_speedTest := flag.Bool("s", false, "Run speed test with 100 cycles")

	param_freeze := flag.String("freeze", "", "Create the frozen MetaJS bootstrap snapshot from the given metajs-to-llvm-ir grammar (writes abnf/jsagrammar.go and abnf/jsbootstrap.ll, then rebuild)")
	param_frozen := flag.Bool("frozen", false, "Run all annotation scripts goja free: parse them with the frozen a-grammar, compile them with the frozen bootstrap on the IR machine, and execute the emitted IR")

	param_verify := flag.Bool("verify", false, "Lint the -a grammar: report used-but-undefined names (error) and defined-but-unreachable productions (warning), then exit without running the program")

	param_cfg := flag.String("cfg", "", "Write the control flow graph of every executed IR module to this file (Graphviz DOT; Mermaid when the name ends in .mmd)")
	param_traceOut := flag.String("trace", "", "Stream the runtime events of compiled programs (llvm.RunJS) to this file as JSON lines; also the input file of -render")
	param_callgraph := flag.String("callgraph", "", "Write the static call graph of every executed IR module: a .jsonl path appends records for -render static (accumulate a codebase over several runs), anything else writes Graphviz DOT")
	param_render := flag.String("render", "", "Render the -trace file to Graphviz DOT on stdout: 'calls' (dynamic call graph), 'vars' (function/variable access graph) or 'static' (-callgraph records, clustered per file)")

	flag.Parse()

	if *param_freeze != "" {
		if err := abnf.Freeze(*param_freeze, "abnf"); err != nil {
			fmt.Fprintln(os.Stderr, "Freeze failed: ", err)
			os.Exit(1)
		}
		return
	}

	if *param_render != "" {
		if err := abnf.RenderTrace(*param_render, *param_traceOut); err != nil {
			fmt.Fprintln(os.Stderr, "Render failed: ", err)
			os.Exit(1)
		}
		return
	}

	abnf.UseFrozenScripts = *param_frozen
	abnf.CFGOutPath = *param_cfg
	abnf.TraceOutPath = *param_traceOut
	abnf.CallgraphOutPath = *param_callgraph
	defer abnf.CloseTrace()

	if *param_a == "" {
		flag.Usage()
		os.Exit(2)
	}

	dat, err := ioutil.ReadFile(*param_a)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		os.Exit(1)
	}
	srcA := string(dat) // This should be an ABNF.

	srcB := ""
	if *param_b != "" {
		dat, err = ioutil.ReadFile(*param_b)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err)
			os.Exit(1)
		}
		srcB = string(dat) // This can be anything that the ABNF understands.
	}

	srcC := ""
	if *param_c != "" {
		dat, err = ioutil.ReadFile(*param_c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err)
			os.Exit(1)
		}
		srcC = string(dat) // This can be anything that the a-grammar from srcB understands.
	}

	// The program whose positions traces and diagrams refer to is the last
	// input of the pipeline: -c when present, else -b.
	if *param_c != "" {
		abnf.SetTraceSource(*param_c, srcC)
	} else if *param_b != "" {
		abnf.SetTraceSource(*param_b, srcB)
	}

	if *param_speedTest {
		speedtest(srcA, *param_a, 100, *param_useBlockList, *param_useFoundList)
		return
	}

	if *param_verbose_All {
		*param_verbose_Ap = true
		*param_verbose_Ac = true
		*param_verbose_Bp = true
		*param_verbose_Bc = true
		*param_verbose_Cp = true
		*param_verbose_Cc = true
	}

	if *param_trace_All {
		*param_trace_Ap = true
		*param_trace_Ac = true
		*param_trace_Bp = true
		*param_trace_Bc = true
		*param_trace_Cp = true
		*param_trace_Cc = true
	}

	if *param_quiet_Full {
		*param_quiet_Most = true
	}

	parseropts := &abnf.Parseropts{
		UseBlockList:         *param_useBlockList,
		UseFoundList:         *param_useFoundList,
		TraceEnabled:         *param_trace_Ap,
		PreventDefaultOutput: *param_quiet_Full,
	}

	// MAIN PROCESS ----------------------------------------------------------------------------------------------

	// Part A:

	// Use the initial a-grammar to parse an ABNF. It generates an ASG (abstract semantic graph) of the ABNF.
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "Parse source ABNF file A with initial a-grammar")
	}
	asg, err := abnf.ParseWithAgrammar(abnf.AbnfAgrammar, srcA, *param_a, parseropts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "  ==> Success, generated abstract semantic graph (ASG)")
	}
	if *param_verbose_Ap || *param_trace_Ap {
		fmt.Fprintf(os.Stderr, "   => ASG:  %s\n\n", asg.Serialize())
	}
	// Use the annotations inside the ASG to compile it. This should generate a new a-grammar.
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "Compile ASG of source ABNF")
	}
	aGrammar, err := abnf.CompileASG(asg, abnf.AbnfAgrammar, *param_a, 0, *param_trace_Ac, *param_quiet_Full)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if aGrammar == nil { // There should be a generated a-grammar
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, "Did not receive a valid a-grammar from compiler")
		os.Exit(1)
	}
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, " ==> Success, received an a-grammar from compiler")
	}
	if *param_verbose_Ac || *param_trace_Ac {
		fmt.Fprintf(os.Stderr, "   => a-grammar:  %s\n\n", aGrammar.Serialize())
	}

	// -verify: lint the a-grammar and exit (do not run the program). The grammar
	// must be assembled first - a parse merges any :include() fragments into it in
	// place (its setup phase runs before matching, and ParseWithAgrammar recovers
	// from a failed match), so an empty target is enough when no -b was given.
	if *param_verify {
		ownNames := abnf.ProductionNames(aGrammar) // Before assembly: the grammar's own productions.
		// Assemble the grammar (merge :include() fragments) only if it has any.
		// A parse triggers the merge, but it also runs the grammar - which would
		// diverge or overflow on a pathological (e.g. left-recursive) grammar, so
		// it is skipped when there is nothing to include.
		if abnf.HasInclude(aGrammar) {
			abnf.ParseWithAgrammar(aGrammar, srcB, *param_b, parseropts)
		}
		issues := abnf.Verify(aGrammar, srcA, ownNames)
		errors := 0
		for _, iss := range issues {
			where := ""
			if iss.Line > 0 {
				where = fmt.Sprintf("%s:%d: ", *param_a, iss.Line)
			}
			tag := "warning"
			if iss.IsError() {
				tag = "error"
				errors++
			}
			fmt.Fprintf(os.Stderr, "%s%s: %s\n", where, tag, iss.Message())
		}
		if len(issues) == 0 {
			fmt.Fprintf(os.Stderr, "%s: verified, no issues.\n", *param_a)
		} else {
			fmt.Fprintf(os.Stderr, "%s: %d issue(s), %d error(s).\n", *param_a, len(issues), errors)
		}
		if errors > 0 {
			os.Exit(1)
		}
		return
	}

	// Part B:

	// Use the a-grammar to parse the text it describes. It generates the ASG (abstract semantic graph) of the parsed text.
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "Parse target file B with new a-grammar")
	}
	parseropts.TraceEnabled = *param_trace_Bp
	asg, err = abnf.ParseWithAgrammar(aGrammar, srcB, *param_b, parseropts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "  ==> Success, generated abstract semantic graph (ASG)")
	}
	if *param_verbose_Bp || *param_trace_Bp {
		fmt.Fprintf(os.Stderr, "   => ASG:  %s\n\n", asg.Serialize())
	}
	// Use the annotations inside the ASG to compile it.
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "Compile ASG")
	}
	result, err := abnf.CompileASG(asg, aGrammar, *param_b, *param_slot_b, *param_trace_Bc, *param_quiet_Full)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, " ==> Success")
	}
	if *param_verbose_Bc || *param_trace_Bc {
		if result != nil {
			fmt.Fprintf(os.Stderr, "   => Result:  %s\n\n", result.Serialize())
		}
	}

	// Part C:

	// Part C only runs if a file -c was given and part B generated an a-grammar to process it with.
	if *param_c == "" {
		return
	}
	if result == nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, "File -c was given, but compiling file -b did not generate an a-grammar to process it with")
		os.Exit(1)
	}

	// Use the a-grammar to parse the text it describes. It generates the ASG (abstract semantic graph) of the parsed text.
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "Parse target file C with new a-grammar")
	}
	parseropts.TraceEnabled = *param_trace_Cp
	asg, err = abnf.ParseWithAgrammar(result, srcC, *param_c, parseropts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "  ==> Success, generated abstract semantic graph (ASG)")
	}
	if *param_verbose_Cp || *param_trace_Cp {
		fmt.Fprintf(os.Stderr, "   => ASG:  %s\n\n", asg.Serialize())
	}
	// Use the annotations inside the ASG to compile it.
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, "Compile ASG")
	}
	result, err = abnf.CompileASG(asg, result, *param_c, *param_slot_c, *param_trace_Cc, *param_quiet_Full)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !*param_quiet_Most {
		fmt.Fprintln(os.Stderr, " ==> Success")
	}
	if *param_verbose_Cc || *param_trace_Cc {
		if result != nil {
			fmt.Fprintf(os.Stderr, "   => Result:  %s\n\n", result.Serialize())
		}
	}
}

func speedtest(srcA, fileNameA string, count int, useBlockList, useFoundList bool) {
	parseropts := &abnf.Parseropts{
		UseBlockList:         useBlockList,
		UseFoundList:         useFoundList,
		TraceEnabled:         false,
		PreventDefaultOutput: true,
	}
	speedtestParseWithGrammar(srcA, fileNameA, count, parseropts)
	speedtestCompileASG(srcA, fileNameA, count, parseropts)
	fmt.Fprintln(os.Stderr)
}
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "%s took %s\n", name, elapsed)
}

func speedtestParseWithGrammar(srcA, fileNameA string, count int, parseropts *abnf.Parseropts) {
	var err error
	defer timeTrack(time.Now(), "ParseWithGrammar")
	for i := 0; i < count; i++ {
		_, err = abnf.ParseWithAgrammar(abnf.AbnfAgrammar, srcA, fileNameA, parseropts)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseWithGrammar")
		return
	}
}
func speedtestCompileASG(srcA, fileNameA string, count int, parseropts *abnf.Parseropts) {
	asg, err := abnf.ParseWithAgrammar(abnf.AbnfAgrammar, srcA, fileNameA, parseropts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ParseWithGrammar")
		return
	}
	defer timeTrack(time.Now(), "CompileASG")
	for i := 0; i < count; i++ {
		_, err = abnf.CompileASG(asg, abnf.AbnfAgrammar, fileNameA, 0, false, true)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error CompileASG")
		return
	}
}
