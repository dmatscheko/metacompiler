package main

import (
	"bytes"
	"fmt"
	"io"
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
//  -v, -vN       verbose for all stages / for stage N (parse ASG + compiled result)
//  -vv, -vvN     parser+compiler trace for all stages / for stage N
//  -slotN V      compile stage N with tag slot V (default 0)
//  -q, -qq       quiet (only program output+errors / only errors)
//  -v-error      on a parse failure, dump the fuller parsed-so-far form (SerializeCompact)
//                instead of the default minimal tree
//  -frozen       run the annotation scripts goja-free (see abnf/frozen.go)
//  -verify       lint the first file's grammar and exit
//  -pretty       print the first file's serialized a-grammar and exit
//  -i DIR        add an include root for project-file imports (repeatable; an import
//                like 'a.b.C' is searched as a/b/C.<ext> under the program's own
//                directory first, then under each -i root in order)
//  -warn-imports warn and skip imports a grammar cannot resolve (default: abort)
//  -warn-unsupported  warn and placeholder parsed-but-unimplemented syntax (default: abort);
//                lets call graphs / CFGs / traces be built from partially understood languages
//  -main NAME    call NAME as the program entry point instead of main (grammars that
//                support it read it as c.mainName)
//  -code SRC     take the final program's source from SRC (given inline) instead of a
//                file, e.g. calculator-interpreter-1.abnf -code '9*(2+3)'
//  -code-stdin   take the final program's source from stdin instead of a file
//  -pipe         start a new pipeline segment: the text a language prints becomes the
//                program input of the next segment, so one language (e.g. a preprocessor)
//                can transform the source another language then consumes. Example:
//                mec c-preprocessor.abnf prog.c -pipe c-to-llvm-ir.abnf
//  -cfg F        write the control flow graph of every executed module to file F (DOT; .mmd = Mermaid)
//  -trace F      stream runtime events to file F as JSON lines; also the -render input
//  -callgraph F  write the static call graph to file F (.jsonl appends for -render static)
//  -render K     standalone (no pipeline): read the JSON-lines file named by -trace F
//                and write graph K to stdout as Graphviz DOT, then exit. K is calls or vars
//                (from a -trace run) or static (from a -callgraph run)
//  -freeze F     (re)create the frozen bootstrap snapshot from grammar file F, then exit
//  -lb, -lf      parser block-list / found-list (debugging aids)
//  -speed N      speed test: warm up once, then time N parse+compile cycles of the first file

// options is the parsed command line.
type options struct {
	files        []string
	verboseAll   bool
	traceAll     bool
	verboseStage map[int]bool // 1-indexed by stage.
	traceStage   map[int]bool
	slotStage    map[int]int // Tag slot to compile a stage with (default 0).

	quietMost, quietFull                  bool
	frozen, verify, pretty                bool
	verboseError                          bool // -v-error: dump the fuller SerializeCompact form on a parse failure.
	warnImports                           bool     // -warn-imports: warn+skip unresolved imports instead of aborting.
	importRoots                           []string // -i include roots for project-file imports, in order.
	warnUnsupported                       bool   // -warn-unsupported: warn+placeholder for not-implemented syntax instead of aborting.
	entryPoint                            string // -main: entry-point function name a compiled program calls (default "main").
	code                                  string // -code VALUE: the final program's source, given inline instead of as a file.
	codeSet, codeStdin                    bool   // -code / -code-stdin were passed (codeStdin reads the source from stdin).
	speedTest, useBlockList, useFoundList bool
	speedCount                            int   // Timed cycle count for -speed (>0 when set).
	pipeBounds                            []int // -pipe boundaries: file indices where a new pipeline segment starts.

	freezePath, cfgPath, tracePath, callgraphPath, renderKind string
}

// parseArgs classifies the command line into files (positional) and flags
// (anything starting with '-'), so the two may be freely interspersed - unlike
// the standard flag package, which stops at the first positional argument.
func parseArgs(args []string) (*options, error) {
	o := &options{verboseStage: map[int]bool{}, traceStage: map[int]bool{}, slotStage: map[int]int{}}
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
		case "-i":
			var dir string
			if dir, err = takeVal(); err == nil {
				o.importRoots = append(o.importRoots, dir)
			}
		case "-warn-imports":
			o.warnImports = true
		case "-warn-unsupported":
			o.warnUnsupported = true
		case "-main":
			o.entryPoint, err = takeVal()
		case "-code":
			o.code, err = takeVal()
			o.codeSet = true
		case "-code-stdin":
			o.codeStdin = true
		case "-pipe":
			// A pipeline segment boundary: the TEXT output of the segment so far
			// becomes the program input of the next segment (see runPipeline).
			if len(o.files) == 0 {
				return nil, fmt.Errorf("-pipe needs a preceding segment (grammar + program)")
			}
			o.pipeBounds = append(o.pipeBounds, len(o.files))
		case "-speed":
			var v string
			if v, err = takeVal(); err == nil {
				n, serr := strconv.Atoi(v)
				if serr != nil || n < 1 {
					return nil, fmt.Errorf("flag -speed needs a positive integer cycle count, got %q", v)
				}
				o.speedTest, o.speedCount = true, n
			}
		case "-lb":
			o.useBlockList = true
		case "-lf":
			o.useFoundList = true
		case "-v":
			o.verboseAll = true
		case "-v-error":
			o.verboseError = true
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
			if n, ok := stageFlag(name, "-slot"); ok {
				v, verr := takeVal()
				if verr != nil {
					return nil, verr
				}
				slot, serr := strconv.Atoi(v)
				if serr != nil {
					return nil, fmt.Errorf("flag %s needs an integer slot, got %q", name, v)
				}
				o.slotStage[n] = slot
			} else if n, ok := stageFlag(name, "-vv"); ok {
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

	// -code / -code-stdin supply the final program's source inline (or from stdin) instead
	// of reading it from a file: the code becomes a synthetic last file that the (compiled)
	// grammar parses. A grammar file is still required as the first positional argument.
	codeIdx := -1
	var codeText string
	if o.codeSet || o.codeStdin {
		if o.codeSet && o.codeStdin {
			fmt.Fprintln(os.Stderr, "Error: -code and -code-stdin are mutually exclusive")
			os.Exit(2)
		}
		if len(o.files) == 0 {
			fmt.Fprintln(os.Stderr, "Error: -code / -code-stdin needs a grammar file")
			os.Exit(2)
		}
		codeText = o.code
		name := "(code)"
		if o.codeStdin {
			dat, e := ioutil.ReadAll(os.Stdin)
			if e != nil {
				fmt.Fprintln(os.Stderr, "Error reading stdin: ", e)
				os.Exit(1)
			}
			codeText = string(dat)
			name = "(stdin)"
		}
		o.files = append(o.files, name)
		codeIdx = len(o.files) - 1
	}

	abnf.UseFrozenScripts = o.frozen
	abnf.VerboseParseErrors = o.verboseError
	abnf.WarnUnresolvedImports = o.warnImports
	abnf.ImportRoots = o.importRoots
	abnf.WarnUnsupported = o.warnUnsupported
	if o.entryPoint != "" {
		abnf.EntryPoint = o.entryPoint
	}
	abnf.CFGOutPath = o.cfgPath
	abnf.TraceOutPath = o.tracePath
	abnf.CallgraphOutPath = o.callgraphPath
	abnf.OpenTrace() // Truncate up front: a zero-event run must not leave a stale file.
	defer abnf.CloseTrace()

	if len(o.files) == 0 {
		printUsage()
		os.Exit(2)
	}
	// No -pipe segment may be empty (that includes a trailing or doubled -pipe).
	{
		b := append(append([]int{0}, o.pipeBounds...), len(o.files))
		for i := 0; i+1 < len(b); i++ {
			if b[i] >= b[i+1] {
				fmt.Fprintln(os.Stderr, "Error: empty -pipe segment (each segment needs at least one file)")
				os.Exit(2)
			}
		}
	}
	// The real stage numbering (runPipeline) goes beyond the file count: every
	// -pipe segment behind the first adds a "(piped)" input stage, and the last
	// segment may add a trailing run stage for a startScript-only grammar - a
	// cap at len(o.files) rejected -vN/-vvN/-slotN for stages the run printed.
	maxStage := len(o.files) + len(o.pipeBounds) + 1
	for stage := range o.verboseStage {
		if stage > maxStage {
			fmt.Fprintf(os.Stderr, "Error: -v%d, but there are at most %d stage(s)\n", stage, maxStage)
			os.Exit(2)
		}
	}
	for stage := range o.traceStage {
		if stage > maxStage {
			fmt.Fprintf(os.Stderr, "Error: -vv%d, but there are at most %d stage(s)\n", stage, maxStage)
			os.Exit(2)
		}
	}
	for stage := range o.slotStage {
		if stage > maxStage {
			fmt.Fprintf(os.Stderr, "Error: -slot%d, but there are at most %d stage(s)\n", stage, maxStage)
			os.Exit(2)
		}
	}

	srcs := make([]string, len(o.files))
	for i, f := range o.files {
		if i == codeIdx { // The synthetic -code / -code-stdin file: its source is already in hand.
			srcs[i] = codeText
			continue
		}
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
		speedtest(srcs[0], o.files[0], o.speedCount, o.useBlockList, o.useFoundList)
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

	runPipeline(o, srcs, parseropts)
}

// runPipeline executes the file pipeline. Without -pipe there is a single
// segment and it behaves exactly like the original loop (a chain of a-grammars,
// each stage's compiled grammar feeding the next file). Each -pipe starts a new
// segment: an independent a-grammar chain whose PROGRAM input is the captured
// text output (script print) of the previous segment - so a language (e.g. a
// preprocessor) can transform the source before another language consumes it.
func runPipeline(o *options, srcs []string, parseropts *abnf.Parseropts) {
	// Segment [start,end) ranges over o.files, split at the -pipe boundaries.
	bounds := append(append([]int{0}, o.pipeBounds...), len(o.files))
	globalStage := 0
	var piped *string // Text output of the previous segment, or nil for the first.
	for s := 0; s+1 < len(bounds); s++ {
		start, end := bounds[s], bounds[s+1]
		isLast := s+2 == len(bounds)

		// This segment's grammar files and their sources; a piped-in text from the
		// previous segment is appended as the final program input.
		files := append([]string{}, o.files[start:end]...)
		segSrcs := append([]string{}, srcs[start:end]...)
		if piped != nil {
			files = append(files, "(piped)")
			segSrcs = append(segSrcs, *piped)
		}

		// A non-last (producer) segment has its stdout captured and fed forward, so
		// its script output must be enabled even under -q/-qq.
		quietFull := o.quietFull
		var buf bytes.Buffer
		var prevOut io.Writer
		if !isLast {
			quietFull = false
			prevOut = abnf.SetOutput(&buf)
		}
		parseropts.PreventDefaultOutput = quietFull

		grammar := abnf.AbnfAgrammar
		for j := range files {
			globalStage++
			verbose := o.verboseAll || o.verboseStage[globalStage]
			trace := o.traceAll || o.traceStage[globalStage]
			if isLast && j == len(files)-1 {
				// Positions in traces/diagrams refer to the final program.
				abnf.SetTraceSource(files[j], segSrcs[j])
			}
			grammar = runStage(grammar, files[j], segSrcs[j], globalStage, o.slotStage[globalStage], verbose, trace, o.quietMost, quietFull, parseropts)
		}

		// A startScript-only trailing grammar (last segment only) runs on empty input.
		if isLast && abnf.GrammarStartScriptOnly(grammar) {
			globalStage++
			verbose := o.verboseAll || o.verboseStage[globalStage]
			trace := o.traceAll || o.traceStage[globalStage]
			runStage(grammar, "", "", globalStage, o.slotStage[globalStage], verbose, trace, o.quietMost, quietFull, parseropts)
		}

		if !isLast {
			abnf.SetOutput(prevOut)
			t := buf.String()
			piped = &t
		}
	}
}

// runStage parses a file with the given a-grammar and compiles the resulting
// ASG; the compiled result is the a-grammar for the next stage. It exits the
// process on any error (the exit code of a compiled program is set by the
// program itself, via the exit() it calls).
func runStage(grammar *r.Rules, file, src string, stage, slot int, verbose, trace, quietMost, quietFull bool, parseropts *abnf.Parseropts) *r.Rules {
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
	result, err := abnf.CompileASG(asg, grammar, file, slot, trace, quietFull)
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
		// A failed assembly (a missing or broken :include() file) is the real
		// finding: swallowed, it surfaced only as an "undefined name" for every
		// fragment production, hiding the cause.
		if _, err := abnf.ParseWithAgrammar(grammar, assemblySrc, assemblyName, parseropts); err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: cannot assemble the grammar's includes: %s\n", o.files[0], err)
			os.Exit(1)
		}
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
  -slotN V      compile stage N with tag slot V (default 0)
  -q, -qq       quiet (program output + errors / errors only)
  -v-error      on a parse failure, dump the fuller parsed-so-far form (SerializeCompact)
                instead of the default minimal tree
  -frozen       run the annotation scripts without goja
  -verify       lint the first file's grammar and exit
  -pretty       print the first file's serialized a-grammar and exit
  -i DIR        add an include root for project-file imports (repeatable); an import
                names a file relative to the program's directory or a root
  -warn-imports warn and skip imports a grammar cannot resolve (default: abort)
  -warn-unsupported  warn+placeholder parsed-but-unimplemented syntax instead of aborting;
                lets call graphs / CFGs / traces be built from partially understood languages
  -main NAME    call NAME as the program entry point instead of main (c.mainName)
  -code SRC     take the final program's source from SRC (inline) instead of a file,
                e.g. languages/calculator-interpreter-1.abnf -code '9*(2+3)'
  -code-stdin   take the final program's source from stdin instead of a file
  -pipe         start a new pipeline segment fed by the previous segment's text output
                (e.g. c-preprocessor.abnf prog.c -pipe c-to-llvm-ir.abnf)
  -cfg F        write the control flow graph of every executed module to file F (DOT; .mmd = Mermaid)
  -trace F      stream runtime events to file F as JSON lines
  -callgraph F  write the static call graph to file F (.jsonl appends for -render static)
  -render K     standalone: read the JSON-lines file named by -trace F and write graph
                K to stdout as Graphviz DOT, then exit. K = calls | vars (from a -trace
                run) or static (from a -callgraph run)
  -freeze F     (re)create the frozen bootstrap snapshot from grammar file F, then exit
  -lb, -lf      parser block-list / found-list debugging aids
  -speed N      speed test: warm up once, then time N parse+compile cycles of the first file
`)
}

// speedtest benchmarks the metacompiler on the first file. It parses and
// compiles the file once as an untimed warm-up (so engine init, lazy caches and
// the one-off goja tag-script compile stay out of the numbers), then times only
// the repeated cycles: N parses, then N compiles of the pre-parsed ASG. The
// result therefore reflects steady-state throughput, not the program start-up or
// the file I/O.
func speedtest(src, fileName string, count int, useBlockList, useFoundList bool) {
	parseropts := &abnf.Parseropts{
		UseBlockList:         useBlockList,
		UseFoundList:         useFoundList,
		TraceEnabled:         false,
		PreventDefaultOutput: true,
	}

	// Warm up once, untimed.
	asg, err := abnf.ParseWithAgrammar(abnf.AbnfAgrammar, src, fileName, parseropts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Speed test: parse failed:", err)
		return
	}
	if _, err = abnf.CompileASG(asg, abnf.AbnfAgrammar, fileName, 0, false, true); err != nil {
		fmt.Fprintln(os.Stderr, "Speed test: compile failed:", err)
		return
	}

	// Time N parse cycles.
	start := time.Now()
	for i := 0; i < count; i++ {
		if asg, err = abnf.ParseWithAgrammar(abnf.AbnfAgrammar, src, fileName, parseropts); err != nil {
			fmt.Fprintln(os.Stderr, "Speed test: parse failed:", err)
			return
		}
	}
	reportSpeed("Parse", time.Since(start), count)

	// Time N compile cycles on the ASG from the last parse.
	start = time.Now()
	for i := 0; i < count; i++ {
		if _, err = abnf.CompileASG(asg, abnf.AbnfAgrammar, fileName, 0, false, true); err != nil {
			fmt.Fprintln(os.Stderr, "Speed test: compile failed:", err)
			return
		}
	}
	reportSpeed("Compile", time.Since(start), count)
	fmt.Fprintln(os.Stderr)
}

// reportSpeed prints the total and per-cycle time of a timed loop.
func reportSpeed(name string, elapsed time.Duration, count int) {
	fmt.Fprintf(os.Stderr, "%-8s %d cycle(s) in %s (%s/cycle)\n", name, count, elapsed, elapsed/time.Duration(count))
}
