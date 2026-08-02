package abnf

import (
	"fmt"
	"path/filepath"

	"14.gy/mec/abnf/r"
)

// WarnUnresolvedImports is set from the -warn-imports CLI flag. When false (the
// default) a program import a grammar cannot resolve aborts the compile; when
// true such an import is reported as a warning and skipped. Grammars read it as
// c.warnImports and turn it into the policy via lib resolveImports().
var WarnUnresolvedImports = false

// WarnUnsupported is set from the -warn-unsupported CLI flag. When false (the
// default) a parsed-but-not-implemented construct aborts the compile with a
// clean file:line message; when true it is reported as a warning and replaced by
// a benign placeholder (a no-op statement / undefined expression) so the rest of
// the program still compiles - enough to build call graphs, CFGs and traces from
// a language that is only partially understood. Grammars read it as
// c.warnUnsupported via the lib notImplemented() family.
var WarnUnsupported = false

// RuntimePrims is set from the -rt-prims CLI flag. It asks a compiler grammar to
// implement the derivable half of its runtime IN the source language instead of
// calling the Go externals: languages/metajs-to-llvm-ir.abnf then compiles
// languages/lib/rt-prims.metajs into the emitted module and routes js_ne, js_le,
// js_typeof and friends to those IR functions, so whatever is still `declare`d
// afterwards is the measured floor of primitives that must be written in C
// (docs/runtime-rework-plan.md, phase 1). Grammars read it as c.rtPrims.
var RuntimePrims = false

// EntryPoint is the name of the top-level function a compiled program calls as
// its entry point, set from the -main CLI flag. Grammars read it as c.mainName; it
// lets a program whose entry function is not named main - or a real-world file with
// no main() at all - be run from a chosen function instead. Empty means "unset": each
// grammar then falls back to its own default entry (main for most, Main for C#).
var EntryPoint = ""

// ExePath is the output path for a native executable, set from the -exe CLI flag.
// A -to-llvm-ir grammar reads it as c.exePath: when non-empty it hands the built
// ir.Module to llvm.BuildExecutable (clang) and writes a real binary instead of
// running the module in the built-in IR interpreter. Empty means "unset".
var ExePath = ""

// ----------------------------------------------------------------------------
// ASG compiler

// scriptEngine executes the annotation scripts of an ASG. There are two
// implementations: compilerscript runs them with goja, frozenEngine compiles
// them with the frozen MetaJS bootstrap and executes the resulting IR.
type scriptEngine interface {
	// RunTagCode executes the code of the given slot of a Tag. It returns the
	// completion value of the code and whether the tag had code for that slot.
	RunTagCode(tag *r.Rule, name scriptName, upStream map[string]r.Object, localASG *r.Rules, slot int, depth int) (r.Object, bool)
	// Ltr returns the global (left to right) variables, e.g. ltr.in.
	Ltr() map[string]r.Object
}

// ----------------------------------------------------------------------------
// The ltr.in accumulator

// ltrText holds the text of all Token the walk has seen so far - the value
// behind ltr.in. Appending to a plain string copied the whole text per Token,
// which made the walk quadratic in the input size (185 MB of copying for one
// compile of languages/kotlin-to-llvm-ir.abnf, the largest allocation site of
// the phase). The text is therefore appended to a buffer and only materialized
// into a string when a script actually READS ltr.in - which the engines
// mediate: the goja engine hands out ltr through a dynamic object and the
// frozen engine through importGoValue, and both call String() there. A
// String() method also keeps the tag trace (%v of the ltr map) unchanged.
type ltrText struct {
	buf   []byte
	str   string // The materialized text of buf, valid while fresh.
	fresh bool
}

func (t *ltrText) add(s string) {
	t.buf = append(t.buf, s...)
	t.fresh = false
}

func (t *ltrText) String() string {
	if !t.fresh {
		t.str, t.fresh = string(t.buf), true
	}
	return t.str
}

// addLtrIn appends the text of one Token to ltr.in. A script may have assigned
// a plain string to ltr.in (or the walk may not have seen a Token yet), so the
// accumulator is created on demand.
func addLtrIn(ltr map[string]r.Object, s string) {
	switch cur := ltr["in"].(type) {
	case *ltrText:
		cur.add(s)
	case string:
		t := &ltrText{buf: make([]byte, 0, len(cur)+len(s)+64)}
		t.buf = append(append(t.buf, cur...), s...)
		ltr["in"] = t
	default:
		panic("Variable 'ltr.in' must only contain strings")
	}
}

// ----------------------------------------------------------------------------
// Upstream keys

// The kinds of upstream variable, by name: 'in' and up.str* are concatenated as
// strings, 'stack' and up.arr* are appended as arrays, everything else is
// collected into an array of the sibling values.
const (
	upOther = iota
	upIn
	upStr
	upStack
	upArr
)

// upKeyKind classifies one upstream key. The merge asks for every key of every
// child, so it switches on the first byte instead of running two
// strings.HasPrefix over each of them.
func upKeyKind(k string) int {
	if len(k) < 2 {
		return upOther
	}
	switch k[0] {
	case 'i':
		if k == "in" {
			return upIn
		}
	case 's':
		if k == "stack" {
			return upStack
		}
		if len(k) >= 3 && k[1] == 't' && k[2] == 'r' {
			return upStr
		}
	case 'a':
		if len(k) >= 3 && k[1] == 'r' && k[2] == 'r' {
			return upArr
		}
	}
	return upOther
}

// appendUpValue adds one child value to a 'stack'/up.arr* accumulator: an array
// contributes its elements, anything else itself.
func appendUpValue(dst []interface{}, v r.Object) []interface{} {
	if arr, ok := v.([]interface{}); ok {
		return append(dst, arr...)
	}
	return append(dst, v)
}

type compiler struct {
	eng        scriptEngine
	fileName   string // The compile target (the parsed input file).
	moduleName string // The grammar's :origin() (fallback: fileName). Tag scripts run under this module, so their include/load/store resolve grammar-relative.
}

//	 OUT
//	  ^
//	  |
//	  C---.      (C) If the current Rule has childs, the childs get sent to 'compile()'. (Also the childs of Tag Rules.)
//	  |    |
//	  ^    v
//	  *    |     (*) All upstream (up.*) values from returning 'compile()'s are combined.
//	 /|    |
//	| | _  |
//	T | |  |     (T) The text of an EBNF Terminal symbol (Token) gets returned and included into 'up.in'.
//	| X |  |     (X) The script of a single Tag Rule gets executed. This is after its childs came back from being split at (C).
//	| | O  |     (O) Other Rules are ignored.
//	| | |  |
//	\ | /  |
//	 \|/   |
//	  *    |     (*) Childs from one Rule get split. Each split path only processes one rule (that can contain childs).
//	  |    |
//	  ^    |
//	  IN<-'
//
// 'upStream' holds the variables that only go up (basically the local variables, 'up' in JS).
// When the branches of sibling rules meet while propagating upwards, their upStream maps are
// merged: 'in' and 'str*' are concatenated as strings, 'stack' and 'arr*' are appended as
// arrays, and all other keys are collected into arrays of their values.
// 'ltrStream' holds the global variables ('ltr' in JS). It is one single map that all rules
// share from left to right, e.g. 'ltr.in' collects the text of all Token seen so far.
func (co *compiler) compile(localASG *r.Rules, slot int, depth int) map[string]r.Object { // => (upStream)
	if localASG == nil || len(*localASG) == 0 {
		return map[string]r.Object{"in": ""}
	}

	// ----------------------------------
	// Split and collect

	if len(*localASG) > 1 { // "SEQUENCE" Iterate through all rules and applies.

		// 'in' and the up.str*/up.arr* keys are accumulated in local buffers and
		// stored into the map ONCE, at the end: merging a sibling list by
		// concatenating string into string per child made a node with n children
		// copy its text n times (the widest node of languages/kotlin-to-llvm-ir.abnf
		// has 1102 children), and storing a slice back into an interface{} per
		// child boxed a fresh slice header every time.
		var inBuf []byte
		var strBufs map[string][]byte     // up.str*, on demand: most nodes have none.
		var arrs map[string][]interface{} // up.arr*, likewise.
		stack := []interface{}{}          // Stays a non-nil empty array when no child has one.
		upStreamMerged := map[string]r.Object{}

		for _, rule := range *localASG {
			// Compile:
			upStreamNew := co.compileRule(rule, nil, slot, depth+1)

			for k, v := range upStreamNew {
				// A key that no earlier sibling had starts empty; a value that
				// is not a string where a string belongs is a real error.
				switch upKeyKind(k) {
				case upIn, upStr:
					str, ok := v.(string)
					if !ok {
						panic(fmt.Sprintf("Right variable 'up.%s' must only contain strings. Contains: %#v in rule %s.", k, v, rule.ToString()))
					}
					if k == "in" {
						inBuf = append(inBuf, str...)
						continue
					}
					if strBufs == nil {
						strBufs = map[string][]byte{}
					}
					strBufs[k] = append(strBufs[k], str...)
				case upStack:
					stack = appendUpValue(stack, v)
				case upArr:
					if arrs == nil {
						arrs = map[string][]interface{}{}
					}
					arrs[k] = appendUpValue(arrs[k], v)
				default:
					// If upStreamMerged[k] already holds an array, it must stay that array and must NOT get filled with newer v.
					// So if upStreamMerged[k] has no previous entry, create an array inside and add the array v one object.
					arr, _ := upStreamMerged[k].([]interface{})
					upStreamMerged[k] = append(arr, v)
				}
			}
		}

		upStreamMerged["in"] = string(inBuf)
		upStreamMerged["stack"] = stack
		for k, buf := range strBufs {
			upStreamMerged[k] = string(buf)
		}
		for k, arr := range arrs {
			upStreamMerged[k] = arr
		}

		return upStreamMerged
	}

	// ----------------------------------
	// Inside each split arm do this

	// There is only one production:
	return co.compileRule((*localASG)[0], localASG, slot, depth)
}

// compileRule is the walk of ONE rule (one split arm). localASG is the single
// rule list the enclosing compile() was called with - the list a Tag script sees
// as c.localAsg. The sibling walk above has no such list and passes nil; only a
// Tag then needs one, so only a Tag allocates it (it used to wrap every child of
// every node in a one element r.Rules just to re-enter compile()).
func (co *compiler) compileRule(rule *r.Rule, localASG *r.Rules, slot int, depth int) map[string]r.Object { // => (upStream)
	switch rule.Operator {
	case r.Token:
		addLtrIn(co.eng.Ltr(), rule.String)
		return map[string]r.Object{"in": rule.String, "stack": []interface{}{}}
	case r.Tag:
		// First collect all the data.
		upStream := co.compile(rule.Childs, slot, depth+1) // Evaluate the child productions of the TAG to collect their values.
		// The tag sees the source position of its node as up.pos (the builders
		// capture it for traces and diagrams); it does not propagate upwards.
		upStream["pos"] = rule.Pos
		if localASG == nil {
			localASG = &r.Rules{rule}
		}
		// Then run the script on it. The module is the GRAMMAR (the tag code
		// lives there), so a load()/include() in a tag resolves relative to the
		// grammar under goja exactly like under -frozen; the position suffix
		// still names the input node for error messages.
		co.eng.RunTagCode(rule, nodeScript(co.moduleName, ":tag:pos:", rule.Pos), upStream, localASG, slot, depth)
		delete(upStream, "pos")
		return upStream
	default:
		// Not all rules have childs. E.g. a Number (from :number()) is a leaf like a Token, but without text.
		if rule.Childs != nil && len(*rule.Childs) > 0 {
			return co.compile(rule.Childs, slot, depth+1) // Evaluate the child productions of groups to collect their values.
		}
	}

	return map[string]r.Object{"in": ""}
}

func compileASGInternal(asg *r.Rules, aGrammar *r.Rules, fileName string, slot int, traceEnabled bool, preventDefaultOutput bool) interface{} {
	var co compiler

	if UseFrozenScripts {
		co.eng = newFrozenEngine(&co, asg, aGrammar, traceEnabled, preventDefaultOutput)
	} else {
		co.eng = NewCompilerScript(&co, asg, aGrammar, traceEnabled, preventDefaultOutput)
	}
	co.fileName = filepath.Clean(fileName)
	// Scripts belong to the GRAMMAR: both the start script and the tags run
	// under the grammar's module name (include/load/store resolve relative to
	// it), falling back to the compile target for grammars without an
	// :origin() stamp.
	co.moduleName = r.GetOrigin(aGrammar)
	if co.moduleName == "" {
		co.moduleName = co.fileName
	}

	startScript := r.GetStartScript(aGrammar)

	var res interface{}
	if startScript != nil {
		upStream := map[string]r.Object{ // Basically the local variables.
			"in": "", // This is the parser input (the terminals).
		}
		// The actual co.compile() of the ASG is called from inside the start script (via the JS function c.compile()).
		v, ran := co.eng.RunTagCode(startScript, fileScript(co.moduleName+":startScript"), upStream, asg, slot, 0)
		if ran { // False if the start script has no code for the requested slot.
			res = v
		}
	}

	return res
}

// CompileASG compiles an "abstract semantic graph". This is similar to an AST, but it also contains the semantic of the language.
// The aGrammar is only needed for its start script (the parser needs it for everything else, the ASG already contains the rest).
func CompileASG(asg *r.Rules, aGrammar *r.Rules, fileName string, slot int, traceEnabled, preventDefaultOutput bool) (res *r.Rules, e error) {
	defer func() {
		if err := recover(); err != nil {
			res = nil
			e = fmt.Errorf("%s", err)
		}
	}()

	resObj := compileASGInternal(asg, aGrammar, fileName, slot, traceEnabled, preventDefaultOutput)

	// If the start script returned an a-grammar, convert and return it. Everything else
	// (e.g. a number or a string from a calculator grammar) results in res == nil.
	switch resultAGrammar := resObj.(type) {
	case *r.Rules: // The script used e.g. abnf.arrayToRules() or returned c.agrammar.
		res = resultAGrammar
	case []interface{}: // The script returned a plain JS array that hopefully contains rules.
		res = &r.Rules{}
		for _, rule := range resultAGrammar {
			if r, ok := rule.(*r.Rule); ok {
				*res = append(*res, r)
			} else {
				return nil, nil
			}
		}
	}
	if res != nil {
		resolveIncludePaths(res, fileName)
		// Stamp the source file onto the grammar (unless a recompile already
		// carries one): the start script and its include()/load()/store()
		// then resolve relative to the GRAMMAR, not the parsed input.
		if r.GetOrigin(res) == "" {
			*res = append(*res, &r.Rule{Operator: r.Command, String: "origin",
				CodeChilds: &r.Rules{&r.Rule{Operator: r.Token, String: filepath.Clean(fileName)}}})
		}
	}
	return res, nil
}

// GrammarStartScriptOnly reports whether an a-grammar has a :startScript() but
// no :startRule(). Such a grammar takes no input - it can only be run, with its
// startScript executing on an empty ASG. The pipeline runs it as the final
// stage, so the last file of a run may be a startScript-only grammar.
func GrammarStartScriptOnly(aGrammar *r.Rules) bool {
	return aGrammar != nil && r.GetStartRule(aGrammar) == nil && r.GetStartScript(aGrammar) != nil
}

// SerializeGrammarPretty renders an a-grammar as a pretty-printed Go literal in
// the canonical bootstrap form used by abnf/agrammar.go: the runtime :origin()
// provenance stamp that CompileASG adds is dropped, so the output is the
// grammar's structure alone (directly comparable to AbnfAgrammar). Backs the
// -pretty CLI flag.
func SerializeGrammarPretty(aGrammar *r.Rules) string {
	if aGrammar == nil {
		return "<nil>"
	}
	canonical := r.Rules{}
	for _, rule := range *aGrammar {
		if rule.Operator == r.Command && rule.String == "origin" {
			continue
		}
		canonical = append(canonical, rule)
	}
	return canonical.SerializePretty()
}

// resolveIncludePaths anchors the constant file name of every top level
// :include() command to the grammar file's directory. The include itself only
// runs when the a-grammar is USED (see the parser), where just the INPUT file
// name is known - without this pass the path would resolve relative to
// whatever file happens to be parsed. Rule.Int == 1 marks an anchored path;
// a dynamic (non constant) file name keeps the legacy input relative lookup.
func resolveIncludePaths(agrammar *r.Rules, fileName string) {
	for _, rule := range *agrammar {
		if rule.Operator != r.Command || rule.String != "include" || rule.Int != 0 {
			continue
		}
		if rule.CodeChilds == nil || len(*rule.CodeChilds) == 0 {
			continue
		}
		param := (*rule.CodeChilds)[0]
		if param.Operator != r.Token || filepath.IsAbs(param.String) {
			continue
		}
		param.String = filepath.Join(filepath.Dir(fileName), param.String)
		rule.Int = 1
	}
}
