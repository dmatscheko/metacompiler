// floor-coverage answers one question: WHICH BODIES OF languages/lib/runtime.c
// DOES THE TEST SUITE NEVER REACH?
//
// That question gates every proposal to move a floor body up into layer 2: a
// body no test reaches is a body no move can be validated against, so "move it
// and see if the suite stays green" is not a check at all for those.
//
// HOW IT WORKS, and why it does not need runtime.c edited permanently:
//
//  1. Parse languages/lib/runtime.c and find every top-level function
//     definition by a brace-depth scan (comments and literals honoured).
//  2. Write an INSTRUMENTED COPY into a temp directory: one
//     `__FC_N[k] = __FC_N[k] + 1;` at the top of each body, plus a dump
//     routine that writes `#FC <id> <count>` lines to fd 2. The copy is
//     temporary; the checked-in runtime.c is never touched. `-keep DIR` keeps
//     the copy if you want to read it.
//  3. Compile that copy with OUR OWN languages/c-to-llvm-ir.abnf, exactly the
//     way tests/gen-runtime-ll.sh compiles the real one - so what is measured
//     is the artifact this project actually links, not a clang build of the
//     same text.
//  4. Build each corpus program with `-rt <instrumented floor> -rt
//     lib/<lang>-rt.ll`. An explicit -rt REPLACES the grammar's own two
//     defaults (see any *-to-llvm-ir.abnf's libPath block), which is what makes
//     the substitution possible without touching languages/ at all.
//  5. Run each binary, read the `#FC` lines off its stderr, and sum.
//
// THE DUMP IS CHECKED, NOT ASSUMED. Every run must end its stderr with a
// `#FCEND` sentinel. A binary that crashed, was killed, or exited through a path
// the exit hook missed produced NO counts, and counting that as "nothing was
// reached" would manufacture exactly the false unreached bodies this tool exists
// to find. Such a run is reported as UNUSABLE and its language is named.
//
// WHAT IT CANNOT SEE - state these when quoting it:
//
//   - It measures THE CORPUS IT RUNS, which is the two native self-checking
//     programs per language (the ratchet and the feature matrix), i.e. what
//     tests/native-full.sh runs. An unreached body means "no program in that
//     corpus called it", NOT "dead code". A body reached only by the Go twin,
//     by llvm.Run, or by a program nobody checked in is unreached here and is
//     still live.
//   - ONLY THE THIRTEEN HANDLE-IR LANGUAGES LINK THE FLOOR. c, bash and batch
//     emit self-contained IR against their own C runtime, and the toys link
//     nothing, so they cannot exercise a floor body and are not run.
//   - It counts CALLS, not lines or branches. A body entered once with half of
//     it never taken reads as reached.
//   - The instrumented floor is a different module from the checked-in one, so
//     its timings mean nothing. Do not read performance out of this.
//   - Inlining is not a hazard here (the counter is the first statement of the
//     body, so it survives), but a body the C grammar compiles away entirely
//     would read as unreached. None does today.
//
// WHAT IT COSTS: one native build per program (the same bill tests/native-full.sh
// pays, about a minute for the whole corpus) plus the floor compile - so it is
// deliberately NOT wired into tests/gates.sh's default path. It is an instrument
// you run when you are about to move a floor body, not a gate you run every time.
// Instrumentation overhead on the binaries themselves is one increment of a
// global per call.
//
// Usage:
//
//	go run ./tools/floor-coverage                 # the whole corpus, summary + unreached list
//	go run ./tools/floor-coverage -v              # every body with its count
//	go run ./tools/floor-coverage -langs lua,ruby # a subset (faster while iterating)
//	go run ./tools/floor-coverage -by-lang        # also: which languages reach each body
//	go run ./tools/floor-coverage -keep /tmp/fc   # keep the instrumented copy and the .ll
//	go run ./tools/floor-coverage -max N          # exit 1 above N unreached bodies (a ratchet)
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// The thirteen grammars whose -exe build links languages/lib/runtime.ll. Kept as
// a literal list AND checked against the grammars at run time (see
// checkFloorLangs): a new language that links the floor and is not measured here
// would silently shrink the corpus, which is the failure mode tests/gates.sh's
// recorded counts exist to prevent.
var floorLangs = []string{
	"csharp", "dart", "go", "java", "js", "kotlin", "lua",
	"metajs", "php", "python", "ruby", "swift", "typescript",
}

// The source extension of each language's two self-checking programs, mirroring
// the ROWS table in tests/native-full.sh. Same corpus, same files.
var langExt = map[string]string{
	"csharp": "cs", "dart": "dart", "go": "go", "java": "java", "js": "js",
	"kotlin": "kt", "lua": "lua", "metajs": "js", "php": "php",
	"python": "py", "ruby": "rb", "swift": "swift", "typescript": "ts",
}

type fnBody struct {
	name  string
	line  int // 1-based line of the opening brace
	lines int // body length in lines, for the report
	open  int // byte offset just after the opening '{'
}

// ---------------------------------------------------------------------------
// 1. Finding the bodies.

// scanBodies finds every top-level function definition in C source.
//
// The rule is narrow on purpose: at brace depth 0, a '{' whose last non-space
// predecessor is ')' opens a function body. That accepts every definition in
// runtime.c (including the one-line ones) and rejects the two other things at
// depth 0 that carry braces - a struct/union body, whose '{' follows an
// identifier, and an array initialiser, whose '{' follows '='. Prototypes end in
// ';' and never reach a '{' at all.
func scanBodies(src string) []fnBody {
	var out []fnBody
	depth := 0
	line := 1
	i := 0
	n := len(src)
	// lastCode is the offset of the most recent non-space, non-comment byte.
	lastCode := -1
	for i < n {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
			continue
		case c == '/' && i+1 < n && src[i+1] == '*':
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			i += 2
			continue
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		case c == '"' || c == '\'':
			q := c
			i++
			for i < n && src[i] != q {
				if src[i] == '\\' {
					i++
				}
				if i < n && src[i] == '\n' {
					line++
				}
				i++
			}
			i++
			lastCode = i - 1
			continue
		case c == ' ' || c == '\t' || c == '\r':
			i++
			continue
		}
		if c == '{' {
			if depth == 0 && lastCode >= 0 && src[lastCode] == ')' {
				if name := nameBefore(src, lastCode); name != "" {
					out = append(out, fnBody{name: name, line: line, open: i + 1})
				}
			}
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 && len(out) > 0 {
				b := &out[len(out)-1]
				if b.lines == 0 {
					b.lines = line - b.line + 1
				}
			}
		}
		lastCode = i
		i++
	}
	return out
}

// nameBefore walks back from the ')' that closes a parameter list to the
// identifier in front of its '('.
func nameBefore(src string, closeParen int) string {
	depth := 0
	i := closeParen
	for i >= 0 {
		if src[i] == ')' {
			depth++
		} else if src[i] == '(' {
			depth--
			if depth == 0 {
				break
			}
		}
		i--
	}
	if i < 0 {
		return ""
	}
	i--
	for i >= 0 && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n') {
		i--
	}
	end := i + 1
	for i >= 0 && (src[i] == '_' || isAlnum(src[i])) {
		i--
	}
	name := src[i+1 : end]
	if name == "" || (name[0] >= '0' && name[0] <= '9') {
		return ""
	}
	return name
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// ---------------------------------------------------------------------------
// 2. The instrumented copy.

// instrument returns the instrumented source and the body list it indexes.
//
// main is deliberately NOT instrumented: it is reached by definition, and it is
// rewritten here into __fc_main so that the appended main() can dump the counts
// on every normal return. The nine `exit(...)` sites are routed through
// __fc_exit for the same reason - a floor that exits (an uncaught exception, an
// out-of-memory abort, the program's own exit()) must still report.
func instrument(src string) (string, []fnBody, error) {
	all := scanBodies(src)
	var bodies []fnBody
	for _, b := range all {
		if b.name == "main" {
			continue
		}
		bodies = append(bodies, b)
	}
	if len(bodies) == 0 {
		return "", nil, fmt.Errorf("no function bodies found - the scanner is wrong, not the file")
	}

	// The declarations go in immediately after the libc prototype block, which
	// is the first point where `write` is declared and therefore the earliest
	// place the dump routine can be announced.
	anchor := "void exit(int code);\n"
	at := strings.Index(src, anchor)
	if at < 0 {
		return "", nil, fmt.Errorf("could not find %q to anchor the instrumentation declarations", strings.TrimSpace(anchor))
	}
	at += len(anchor)

	type splice struct {
		off  int
		text string
	}
	sp := []splice{{at, fmt.Sprintf(`
/* ---- floor-coverage instrumentation (tools/floor-coverage); TEMPORARY ---- */
long __FC_N[%d];
char __FC_BUF[64];
long __FC_DONE;
void __fc_dump(void);
void __fc_exit(int code);
long __fc_digits(long v, long i);
/* -------------------------------------------------------------------------- */
`, len(bodies))}}
	for i, b := range bodies {
		sp = append(sp, splice{b.open, fmt.Sprintf(" __FC_N[%d] = __FC_N[%d] + 1;", i, i)})
	}
	sort.Slice(sp, func(a, b int) bool { return sp[a].off < sp[b].off })

	var sb strings.Builder
	prev := 0
	for _, s := range sp {
		sb.WriteString(src[prev:s.off])
		sb.WriteString(s.text)
		prev = s.off
	}
	sb.WriteString(src[prev:])
	out := sb.String()

	// Route every exit() CALL through the dump. The prototype must survive, so
	// it is restored afterwards; doing it in this order means no call site can
	// be missed by a cleverer pattern.
	out = strings.ReplaceAll(out, "exit(", "__fc_exit(")
	out = strings.ReplaceAll(out, "void __fc_exit(int code);", "void exit(int code);\nvoid __fc_exit(int code);")
	if !strings.Contains(out, "\nvoid exit(int code);\n") {
		return "", nil, fmt.Errorf("the exit() prototype did not survive rewriting")
	}
	// getenv/putenv and friends are untouched; only `exit(` matched, and the
	// one identifier that ends in it in this file is the prototype above.
	if strings.Contains(out, "int __fc_main(void)") {
		return "", nil, fmt.Errorf("runtime.c already contains __fc_main")
	}
	if !strings.Contains(out, "int main(void) {") {
		return "", nil, fmt.Errorf("could not find `int main(void) {` to rename")
	}
	out = strings.Replace(out, "int main(void) {", "int __fc_main(void) {", 1)

	out += fmt.Sprintf(`
/* ---- floor-coverage instrumentation (tools/floor-coverage); TEMPORARY ----
 * __fc_dump writes one "#FC <id> <count>" line per REACHED body to fd 2 and
 * then the "#FCEND" sentinel. The sentinel is the point: a run that does not
 * print it measured nothing, and tools/floor-coverage refuses to fold such a
 * run into the totals rather than reading its silence as "reached nothing".
 * Only reached bodies are printed - the unreached ones are the answer and the
 * tool derives them from the ids that did not appear.
 */
long __fc_digits(long v, long i) {
	char rev[24];
	long n = 0;
	long k;
	if (v == 0) { __FC_BUF[i] = 48; return i + 1; }
	while (v > 0) { rev[n] = (char)(48 + (v %% 10)); v = v / 10; n = n + 1; }
	k = n - 1;
	while (k >= 0) { __FC_BUF[i] = rev[k]; i = i + 1; k = k - 1; }
	return i;
}

void __fc_dump(void) {
	long i = 0;
	long o;
	if (__FC_DONE != 0) { return; }
	__FC_DONE = 1;
	while (i < %d) {
		if (__FC_N[i] != 0) {
			o = 0;
			__FC_BUF[o] = 35; o = o + 1;
			__FC_BUF[o] = 70; o = o + 1;
			__FC_BUF[o] = 67; o = o + 1;
			__FC_BUF[o] = 32; o = o + 1;
			o = __fc_digits(i, o);
			__FC_BUF[o] = 32; o = o + 1;
			o = __fc_digits(__FC_N[i], o);
			__FC_BUF[o] = 10; o = o + 1;
			write(2, __FC_BUF, (unsigned long)o);
		}
		i = i + 1;
	}
	__FC_BUF[0] = 35; __FC_BUF[1] = 70; __FC_BUF[2] = 67;
	__FC_BUF[3] = 69; __FC_BUF[4] = 78; __FC_BUF[5] = 68; __FC_BUF[6] = 10;
	write(2, __FC_BUF, 7);
}

void __fc_exit(int code) { __fc_dump(); exit(code); }

int main(void) {
	int r = __fc_main();
	__fc_dump();
	return r;
}
`, len(bodies))
	return out, bodies, nil
}

// ---------------------------------------------------------------------------
// 3-5. Compile, build, run, collect.

func run(dir string, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var so, se strings.Builder
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	return so.String(), se.String(), err
}

// moduleOnly cuts the emitted module out of a compiler grammar's stdout by the
// same brace-depth scan tests/clang-check.sh and tests/gen-runtime-ll.sh use.
// Looking for "the last line that looks like IR" does not work; see the manual.
func moduleOnly(s string) string {
	var out []string
	depth := 0
	for _, ln := range strings.Split(s, "\n") {
		t := strings.TrimSpace(ln)
		if depth == 0 && !(t == "" || strings.HasPrefix(t, ";") || strings.HasPrefix(t, "@") ||
			strings.HasPrefix(t, "declare") || strings.HasPrefix(t, "define") ||
			strings.HasPrefix(t, "target") || strings.HasPrefix(t, "source_filename") ||
			strings.HasPrefix(t, "attributes") || strings.HasPrefix(t, "!")) {
			break
		}
		out = append(out, ln)
		depth += strings.Count(ln, "{") - strings.Count(ln, "}")
	}
	return strings.Join(out, "\n") + "\n"
}

type langResult struct {
	lang    string
	kind    string
	counts  map[int]int64
	usable  bool
	problem string
}

func main() {
	root := flag.String("root", ".", "repository root")
	verbose := flag.Bool("v", false, "list every body with its count, not just the unreached ones")
	byLang := flag.Bool("by-lang", false, "for each body, also name the languages that reach it")
	langsFlag := flag.String("langs", "", "comma-separated subset of the thirteen floor languages")
	keep := flag.String("keep", "", "keep the instrumented runtime.c and its .ll in this directory")
	jobs := flag.Int("jobs", runtime.NumCPU(), "parallel builds")
	max := flag.Int("max", -1, "exit 1 if more than N bodies are unreached (a ratchet); -1 disables")
	// -ids is how you check this tool by hand. Keep an instrumented build with
	// -keep, run any program of your own against it, and the `#FC <id> <count>`
	// lines on its stderr translate through this table.
	ids := flag.Bool("ids", false, "print the body id table used by the #FC lines, and stop")
	flag.Parse()

	abs, err := filepath.Abs(*root)
	must(err)
	srcPath := filepath.Join(abs, "languages", "lib", "runtime.c")
	raw, err := os.ReadFile(srcPath)
	must(err)

	inst, bodies, err := instrument(string(raw))
	must(err)
	fmt.Printf("floor-coverage: %d function bodies in languages/lib/runtime.c (main excluded)\n", len(bodies))
	if *ids {
		for i, b := range bodies {
			fmt.Printf("%d\t%s\truntime.c:%d\n", i, b.name, b.line)
		}
		return
	}

	work := *keep
	if work == "" {
		work, err = os.MkdirTemp("", "floorcov")
		must(err)
		defer os.RemoveAll(work)
	} else {
		must(os.MkdirAll(work, 0o755))
	}

	// THE INSTRUMENTED COPY LIVES IN THE TEMP DIR, NOT IN languages/lib.
	// runtime.c has no #include (it declares its own libc prototypes), so the
	// copy can live anywhere; a file dropped into languages/lib would be unsafe
	// beside tests/gates.sh and a killed run would leave it in `git status`.
	instPath := filepath.Join(work, "runtime.floorcov.c")
	must(os.WriteFile(instPath, []byte(inst), 0o644))

	// A private mec, so the tool never depends on a stale ./mec in the tree.
	mec := filepath.Join(work, "mec")
	if _, se, err := run(abs, "go", "build", "-o", mec, "."); err != nil {
		fatal("go build: %v\n%s", err, se)
	}

	checkFloorLangs(abs)

	fmt.Printf("floor-coverage: compiling the instrumented floor with languages/c-to-llvm-ir.abnf\n")
	so, se, err := run(abs, mec, "languages/c-to-llvm-ir.abnf", instPath, "-q")
	// The run of main() after the module cannot work here (jsmain/jsdispatch are
	// in the MetaJS module it is linked with, and setjmp has no IR external), so
	// a non-zero exit is expected and only an empty module is a failure. Same
	// reasoning as tests/gen-runtime-ll.sh.
	mod := moduleOnly(so)
	if strings.Count(mod, "\n") < 100 {
		fatal("the instrumented floor did not compile (%d lines of module, err %v)\n%s", strings.Count(mod, "\n"), err, tail(se, 20))
	}
	covLL := filepath.Join(work, "runtime.floorcov.ll")
	must(os.WriteFile(covLL, []byte(mod), 0o644))
	fmt.Printf("floor-coverage: instrumented floor is %d lines of IR\n", strings.Count(mod, "\n"))

	langs := floorLangs
	if *langsFlag != "" {
		langs = nil
		for _, l := range strings.Split(*langsFlag, ",") {
			l = strings.TrimSpace(l)
			if _, ok := langExt[l]; !ok {
				fatal("%q is not one of the thirteen languages that link the floor: %s", l, strings.Join(floorLangs, " "))
			}
			langs = append(langs, l)
		}
	}

	type job struct{ lang, kind, file string }
	var todo []job
	for _, l := range langs {
		for _, k := range []string{"full", "features"} {
			f := filepath.Join("tests", fmt.Sprintf("%s-test-%s.%s", l, k, langExt[l]))
			if _, err := os.Stat(filepath.Join(abs, f)); err == nil {
				todo = append(todo, job{l, k, f})
			}
		}
	}

	results := make([]langResult, len(todo))
	sem := make(chan struct{}, maxi(1, *jobs))
	var wg sync.WaitGroup
	for i, j := range todo {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = runOne(abs, mec, covLL, work, j.lang, j.kind, j.file)
		}(i, j)
	}
	wg.Wait()

	report(bodies, results, *verbose, *byLang, *max)
}

func runOne(abs, mec, covLL, work, lang, kind, file string) langResult {
	res := langResult{lang: lang, kind: kind, counts: map[int]int64{}}
	bin := filepath.Join(work, lang+"."+kind+".out")
	os.Remove(bin)
	// WHICH LAYER 2, read out of the grammar rather than assumed to be
	// lib/<lang>-rt.ll (typescript's is lib/js-rt.ll): an explicit -rt REPLACES
	// the grammar's defaults, so naming a file that does not exist is a BUILD
	// FAILED rather than a fallback.
	rt2, err := layer2Of(abs, lang)
	if err != nil {
		res.problem = err.Error()
		return res
	}
	so, se, err := run(abs, mec, filepath.Join("languages", lang+"-to-llvm-ir.abnf"), file,
		"-q", "-exe", bin,
		"-rt", covLL, "-rt", rt2)
	if err != nil {
		res.problem = "BUILD FAILED: " + tail(se+so, 3)
		return res
	}
	if _, err := os.Stat(bin); err != nil {
		res.problem = "no binary written"
		return res
	}
	_, rse, _ := run(abs, bin)
	// The exit status is NOT the verdict here: a program that fails its own
	// checks still exercised the floor on the way, and its counts are valid.
	// What must hold is that the dump ran.
	if !strings.Contains(rse, "#FCEND") {
		res.problem = "UNUSABLE: no #FCEND sentinel - the binary died before it could dump (its counts are NOT folded in)"
		return res
	}
	for _, ln := range strings.Split(rse, "\n") {
		if !strings.HasPrefix(ln, "#FC ") {
			continue
		}
		f := strings.Fields(ln)
		if len(f) != 3 {
			continue
		}
		id, e1 := strconv.Atoi(f[1])
		c, e2 := strconv.ParseInt(f[2], 10, 64)
		if e1 == nil && e2 == nil {
			res.counts[id] += c
		}
	}
	res.usable = true
	return res
}

// layer2Of reads the second -rt input out of the grammar's own libPath block, so
// this tool links exactly what a normal -exe build would.
func layer2Of(abs, lang string) (string, error) {
	g := filepath.Join(abs, "languages", lang+"-to-llvm-ir.abnf")
	b, err := os.ReadFile(g)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %v", g, err)
	}
	s := string(b)
	at := strings.Index(s, `rts = [libPath("runtime.ll"), libPath("`)
	if at < 0 {
		return "", fmt.Errorf("no `rts = [libPath(\"runtime.ll\"), libPath(...)]` in %s - the grammar's runtime selection changed shape", filepath.Base(g))
	}
	rest := s[at+len(`rts = [libPath("runtime.ll"), libPath("`):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return "", fmt.Errorf("malformed libPath in %s", filepath.Base(g))
	}
	p := filepath.Join(abs, "languages", "lib", rest[:end])
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("%s names layer 2 as %s, which does not exist", filepath.Base(g), p)
	}
	return p, nil
}

// checkFloorLangs re-derives the floor-linking language list from the grammars
// and complains if it has grown. A language added to the repo and not to
// floorLangs would quietly shrink this tool's corpus, and a coverage tool that
// silently measures less is worse than none - the same argument as the recorded
// counts in tests/gates.sh.
func checkFloorLangs(abs string) {
	got := map[string]bool{}
	files, _ := filepath.Glob(filepath.Join(abs, "languages", "*-to-llvm-ir.abnf"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil || !strings.Contains(string(b), `libPath("runtime.ll")`) {
			continue
		}
		got[strings.TrimSuffix(filepath.Base(f), "-to-llvm-ir.abnf")] = true
	}
	known := map[string]bool{}
	for _, l := range floorLangs {
		known[l] = true
	}
	var extra []string
	for l := range got {
		if !known[l] {
			extra = append(extra, l)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		fmt.Printf("floor-coverage: WARNING - %s link lib/runtime.ll and are NOT in this tool's list.\n"+
			"                Their programs are NOT being run, so every body only they reach reads as unreached.\n"+
			"                Add them to floorLangs and langExt in tools/floor-coverage/main.go.\n",
			strings.Join(extra, ", "))
	}
	if len(got) < len(floorLangs) {
		fmt.Printf("floor-coverage: WARNING - only %d of the %d listed languages still link the floor\n", len(got), len(floorLangs))
	}
}

func report(bodies []fnBody, results []langResult, verbose, byLang bool, max int) {
	total := map[int]int64{}
	reachedBy := map[int][]string{}
	var unusable []string
	usable := 0
	for _, r := range results {
		if !r.usable {
			unusable = append(unusable, fmt.Sprintf("%s/%s: %s", r.lang, r.kind, r.problem))
			continue
		}
		usable++
		for id, c := range r.counts {
			total[id] += c
			reachedBy[id] = append(reachedBy[id], r.lang)
		}
	}

	fmt.Println()
	if len(unusable) > 0 {
		fmt.Printf("%d PROGRAM(S) CONTRIBUTED NOTHING - the unreached list below is NOT a measurement\n", len(unusable))
		for _, u := range unusable {
			fmt.Printf("  %s\n", u)
		}
		fmt.Println("  A program that did not report cannot distinguish `never called it` from `never ran`.")
		fmt.Println()
	}

	var un []int
	for i := range bodies {
		if total[i] == 0 {
			un = append(un, i)
		}
	}
	fmt.Printf("%d of %d floor bodies REACHED, %d UNREACHED, over %d program(s)\n",
		len(bodies)-len(un), len(bodies), len(un), usable)

	if verbose {
		fmt.Println("\nevery body, by call count:")
		idx := make([]int, len(bodies))
		for i := range idx {
			idx[i] = i
		}
		sort.Slice(idx, func(a, b int) bool {
			if total[idx[a]] != total[idx[b]] {
				return total[idx[a]] > total[idx[b]]
			}
			return bodies[idx[a]].name < bodies[idx[b]].name
		})
		for _, i := range idx {
			fmt.Printf("  %14d  %-28s runtime.c:%d (%d lines)%s\n",
				total[i], bodies[i].name, bodies[i].line, bodies[i].lines, langsOf(byLang, reachedBy[i]))
		}
	}

	fmt.Printf("\nUNREACHED - no program in the corpus called these (%d):\n", len(un))
	sort.Slice(un, func(a, b int) bool { return bodies[un[a]].name < bodies[un[b]].name })
	for _, i := range un {
		fmt.Printf("  %-28s runtime.c:%d (%d lines)\n", bodies[i].name, bodies[i].line, bodies[i].lines)
	}
	fmt.Println("\nUNREACHED IS NOT DEAD. It means no checked-in native program called it: the Go")
	fmt.Println("twin, llvm.Run and any program outside tests/ are all invisible here. What it")
	fmt.Println("does mean is that moving one of these into layer 2 CANNOT BE VALIDATED by this")
	fmt.Println("suite - the move would be green whether it was right or wrong (docs/todo.md 3.2).")

	if max >= 0 && len(un) > max {
		fmt.Printf("\nfloor-coverage: %d unreached, above the -max %d\n", len(un), max)
		os.Exit(1)
	}
	if len(unusable) > 0 {
		os.Exit(1)
	}
}

func langsOf(on bool, ls []string) string {
	if !on || len(ls) == 0 {
		return ""
	}
	seen := map[string]bool{}
	var out []string
	for _, l := range ls {
		if !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	sort.Strings(out)
	return "  [" + strings.Join(out, " ") + "]"
}

func tail(s string, n int) string {
	ls := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(ls) > n {
		ls = ls[len(ls)-n:]
	}
	return strings.Join(ls, " | ")
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func must(err error) {
	if err != nil {
		fatal("%v", err)
	}
}

func fatal(f string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "floor-coverage: "+f+"\n", a...)
	os.Exit(2)
}
