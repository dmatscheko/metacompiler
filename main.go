package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"14.gy/mec/abnf"
	r "14.gy/mec/abnf/r"
)

// The metacompiler runs a pipeline over the files given on the command line:
//
//	./mec grammar.abnf program.txt
//	./mec meta.abnf lang.abnf mid.x program.y
//
// The FIRST file is parsed and compiled by the built-in a-grammar (the one that
// understands ABNF), which yields its a-grammar. Every FURTHER file is parsed
// and compiled by the a-grammar the previous stage produced. So each stage feeds
// its compiled grammar into the next file - a chain of any length. After the last
// file, if the grammar it produced is startScript-only (no :startRule(), so it
// takes no input), its startScript is run; that lets the last file be a
// startScript-only grammar (e.g. one that builds and runs a module by hand).
//
// Flags may appear anywhere among the files:
//
//	-v / -vN     verbose for all stages / for stage N (parse ASG + compiled result)
//	-vv / -vvN   parser+compiler trace for all stages / for stage N
//	-q / -qq     quiet (only program output+errors / only errors)
//	-frozen      run the annotation scripts goja-free (see abnf/frozen.go)
//	-verify      lint the first file's grammar and exit
//	-pretty      print the first file's serialized a-grammar and exit
//	-cfg F       write the CFG of every executed module to F (DOT; .mmd = Mermaid)
//	-trace F     stream runtime events to F (JSON lines); also the -render input
//	-callgraph F write the static call graph (.jsonl appends for -render static)
//	-render K    render a -trace/-callgraph file to DOT: calls | vars | static
//	-freeze F    (re)create the frozen bootstrap snapshot from grammar F, then exit
//	-lb / -lf    parser block-list / found-list (debugging aids)
//	-s           speed test (100 cycles on the first file)

// options is the parsed command line.
type options struct {
	files        []string
	verboseAll   bool
	traceAll     bool
	verboseStage map[int]bool // 1-indexed by stage.
	traceStage   map[int]bool

	quietMost, quietFull                  bool
	frozen, verify, pretty                bool
	speedTest, useBlockList, useFoundList bool

	freezePath, cfgPath, tracePath, callgraphPath, renderKind string
}

// parseArgs classifies the command line into files (positional) and flags
// (anything starting with '-'), so the two may be freely interspersed - unlike
// the standard flag package, which stops at the first positional argument.
func parseArgs(args []string) (*options, error) {
	o := &options{verboseStage: map[int]bool{}, traceStage: map[int]bool{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) == 0 || a[0] != '-' || a == "-" {
			o.files = append(o.files, a)
			continue
		}
		name, val, hasVal := a, "", false
		if eq := strings.IndexByte(a, '='); eq >= 0 {
			name, val, hasVal = a[:eq], a[eq+1:], true
		}
		// takeVal returns the value of a value-flag: from "-flag=value", or the
		// next argument otherwise.
		takeVal := func() (string, error) {
			if hasVal {
				return val, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %s needs a value", name)
			}
			i++
			return args[i], nil
		}

		var err error
		switch name {
		case "-q":
			o.quietMost = true
		case "-qq":
			o.quietMost, o.quietFull = true, true
		case "-frozen":
			o.frozen = true
		case "-verify":
			o.verify = true
		case "-pretty":
			o.pretty = true
		case "-s":
			o.speedTest = true
		case "-lb":
			o.useBlockList = true
		case "-lf":
			o.useFoundList = true
		case "-v":
			o.verboseAll = true
		case "-vv":
			o.traceAll = true
		case "-freeze":
			o.freezePath, err = takeVal()
		case "-cfg":
			o.cfgPath, err = takeVal()
		case "-trace":
			o.tracePath, err = takeVal()
		case "-callgraph":
			o.callgraphPath, err = takeVal()
		case "-render":
			o.renderKind, err = takeVal()
		default:
			if n, ok := stageFlag(name, "-vv"); ok {
				o.traceStage[n] = true
			} else if n, ok := stageFlag(name, "-v"); ok {
				o.verboseStage[n] = true
			} else {
				return nil, fmt.Errorf("unknown flag %q", a)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return o, nil
}

// stageFlag parses a stage-numbered verbose/trace flag like -v3 or -vv12.
func stageFlag(name, prefix string) (int, bool) {
	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(name[len(prefix):])
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func main() {
	o, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		printUsage()
		os.Exit(2)
	}

	// Modes that do not run the file pipeline.
	if o.freezePath != "" {
		if err := abnf.Freeze(o.freezePath, "abnf"); err != nil {
			fmt.Fprintln(os.Stderr, "Freeze failed: ", err)
			os.Exit(1)
		}
		return
	}
	if o.renderKind != "" {
		if err := abnf.RenderTrace(o.renderKind, o.tracePath); err != nil {
			fmt.Fprintln(os.Stderr, "Render failed: ", err)
			os.Exit(1)
		}
		return
	}

	abnf.UseFrozenScripts = o.frozen
	abnf.CFGOutPath = o.cfgPath
	abnf.TraceOutPath = o.tracePath
	abnf.CallgraphOutPath = o.callgraphPath
	defer abnf.CloseTrace()

	if len(o.files) == 0 {
		printUsage()
		os.Exit(2)
	}
	for stage := range o.verboseStage {
		if stage > len(o.files) {
			fmt.Fprintf(os.Stderr, "Error: -v%d, but there are only %d stage(s)\n", stage, len(o.files))
			os.Exit(2)
		}
	}
	for stage := range o.traceStage {
		if stage > len(o.files) {
			fmt.Fprintf(os.Stderr, "Error: -vv%d, but there are only %d stage(s)\n", stage, len(o.files))
			os.Exit(2)
		}
	}

	srcs := make([]string, len(o.files))
	for i, f := range o.files {
		dat, e := ioutil.ReadFile(f)
		if e != nil {
			fmt.Fprintln(os.Stderr, "Error: ", e)
			os.Exit(1)
		}
		srcs[i] = string(dat)
	}

	parseropts := &abnf.Parseropts{
		UseBlockList:         o.useBlockList,
		UseFoundList:         o.useFoundList,
		PreventDefaultOutput: o.quietFull,
	}

	if o.speedTest {
		speedtest(srcs[0], o.files[0], 100, o.useBlockList, o.useFoundList)
		return
	}

	// -verify / -pretty inspect the first file's compiled a-grammar and exit.
	if o.verify || o.pretty {
		grammar := compileFirst(o.files[0], srcs[0], parseropts, o.quietMost, o.quietFull)
		if o.pretty {
			fmt.Println(abnf.SerializeGrammarPretty(grammar))
			return
		}
		runVerify(o, grammar, srcs, parseropts)
		return
	}

	// The pipeline.
	grammar := abnf.AbnfAgrammar
	for i := range o.files {
		stage := i + 1
		verbose := o.verboseAll || o.verboseStage[stage]
		trace := o.traceAll || o.traceStage[stage]
		if i == len(o.files)-1 {
			// Positions in traces/diagrams refer to the last file (the program).
			abnf.SetTraceSource(o.files[i], srcs[i])
		}
		grammar = runStage(grammar, o.files[i], srcs[i], stage, verbose, trace, o.quietMost, o.quietFull, parseropts)
	}

	// If the last file produced a startScript-only grammar, run it on empty input.
	if abnf.GrammarStartScriptOnly(grammar) {
		last := len(o.files)
		verbose := o.verboseAll || o.verboseStage[last]
		trace := o.traceAll || o.traceStage[last]
		runStage(grammar, "", "", last, verbose, trace, o.quietMost, o.quietFull, parseropts)
	}
}

// runStage parses a file with the given a-grammar and compiles the resulting
// ASG; the compiled result is the a-grammar for the next stage. It exits the
// process on any error (the exit code of a compiled program is set by the
// program itself, via the exit() it calls).
func runStage(grammar *r.Rules, file, src string, stage int, verbose, trace, quietMost, quietFull bool, parseropts *abnf.Parseropts) *r.Rules {
	target := file
	if target == "" {
		target = "(run)"
	}
	if !quietMost {
		fmt.Fprintf(os.Stderr, "Stage %d: parse %s\n", stage, target)
	}
	parseropts.TraceEnabled = trace
	asg, err := abnf.ParseWithAgrammar(grammar, src, file, parseropts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !quietMost {
		fmt.Fprintln(os.Stderr, "  ==> Success, generated abstract semantic graph (ASG)")
	}
	if verbose && asg != nil {
		fmt.Fprintf(os.Stderr, "   => ASG:  %s\n\n", asg.Serialize())
	}

	if !quietMost {
		fmt.Fprintf(os.Stderr, "Stage %d: compile\n", stage)
	}
	result, err := abnf.CompileASG(asg, grammar, file, 0, trace, quietFull)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !quietMost {
		fmt.Fprintln(os.Stderr, " ==> Success")
	}
	if verbose && result != nil {
		fmt.Fprintf(os.Stderr, "   => Result:  %s\n\n", result.Serialize())
	}
	return result
}

// compileFirst parses and compiles the first file with the built-in a-grammar,
// returning its a-grammar (used by -verify and -pretty). Exits on failure.
func compileFirst(file, src string, parseropts *abnf.Parseropts, quietMost, quietFull bool) *r.Rules {
	asg, err := abnf.ParseWithAgrammar(abnf.AbnfAgrammar, src, file, parseropts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	grammar, err := abnf.CompileASG(asg, abnf.AbnfAgrammar, file, 0, false, quietFull)
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ==> Fail")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if grammar == nil {
		fmt.Fprintln(os.Stderr, "Error: the first file did not compile to an a-grammar")
		os.Exit(1)
	}
	return grammar
}

// runVerify lints a compiled a-grammar and exits with the right code.
func runVerify(o *options, grammar *r.Rules, srcs []string, parseropts *abnf.Parseropts) {
	ownNames := abnf.ProductionNames(grammar) // Before assembly: the grammar's own productions.
	if abnf.HasInclude(grammar) {
		// Assemble the :include() fragments by parsing the second file (or empty).
		assemblySrc, assemblyName := "", ""
		if len(o.files) > 1 {
			assemblySrc, assemblyName = srcs[1], o.files[1]
		}
		abnf.ParseWithAgrammar(grammar, assemblySrc, assemblyName, parseropts)
	}
	issues := abnf.Verify(grammar, srcs[0], ownNames)
	errors := 0
	for _, iss := range issues {
		where := ""
		if iss.Line > 0 {
			where = fmt.Sprintf("%s:%d: ", o.files[0], iss.Line)
		}
		tag := "warning"
		if iss.IsError() {
			tag = "error"
			errors++
		}
		fmt.Fprintf(os.Stderr, "%s%s: %s\n", where, tag, iss.Message())
	}
	if len(issues) == 0 {
		fmt.Fprintf(os.Stderr, "%s: verified, no issues.\n", o.files[0])
	} else {
		fmt.Fprintf(os.Stderr, "%s: %d issue(s), %d error(s).\n", o.files[0], len(issues), errors)
	}
	if errors > 0 {
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `Usage: mec [flags] grammar.abnf [file ...]

The first file is compiled by the built-in ABNF a-grammar; each further file is
parsed and compiled by the grammar the previous stage produced. Flags may appear
anywhere among the files.

  -v, -vN       verbose for all stages / stage N (ASG + compiled result)
  -vv, -vvN     parser+compiler trace for all stages / stage N
  -q, -qq       quiet (program output + errors / errors only)
  -frozen       run the annotation scripts without goja
  -verify       lint the first file's grammar and exit
  -pretty       print the first file's serialized a-grammar and exit
  -cfg F        write the control flow graph of every executed module to F
  -trace F      stream runtime events to F as JSON lines
  -callgraph F  write the static call graph (.jsonl appends for -render static)
  -render K     render a -trace/-callgraph file to DOT: calls | vars | static
  -freeze F     (re)create the frozen bootstrap snapshot from grammar F, then exit
  -lb, -lf      parser block-list / found-list debugging aids
  -s            speed test (100 cycles on the first file)
`)
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
