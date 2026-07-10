package abnf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"14.gy/mec/abnf/r"
	"github.com/llir/llvm/ir"
)

// ----------------------------------------------------------------------------
// The freezer (-freeze)
//
// Freeze runs the given metajs-to-llvm-ir.abnf once under goja and snapshots
// everything that the frozen mode needs:
//
//   abnf/jsagrammar.go  - the serialized a-grammar (the pure Go parser input)
//   abnf/jsbootstrap.ll - one IR module holding the emitter library plus one
//                         compiled closure per distinct tag script source
//
// The bootstrap program that gets compiled looks like this (in MetaJS):
//
//   <emitter library: the start script up to the goja driver marker>
//   function jstag_1() { <tag source 1> }
//   ...
//   function main() {
//       var tags = {}
//       tags["<tag source 1>"] = jstag_1
//       ...
//       return {module: m, tags: tags}
//   }
//
// The tag map is keyed by the exact source text (not by the tag UIDs, which
// are numbered per session and would not be stable). Running jsmain on a fresh
// runtime yields a fresh module m plus the closures - that is the handshake
// that frozen.go uses for every script compilation.

// Freeze creates the frozen bootstrap snapshot from the given grammar file and
// writes it into the package source directory outDir (normally "abnf").
func Freeze(grammarPath string, outDir string) error {
	dat, err := ioutil.ReadFile(grammarPath)
	if err != nil {
		return err
	}
	src := string(dat)

	opts := &Parseropts{PreventDefaultOutput: true}
	asg, err := ParseWithAgrammar(AbnfAgrammar, src, grammarPath, opts)
	if err != nil {
		return fmt.Errorf("cannot parse %s: %s", grammarPath, err)
	}
	g, err := CompileASG(asg, AbnfAgrammar, grammarPath, 0, false, true)
	if err != nil {
		return fmt.Errorf("cannot compile %s: %s", grammarPath, err)
	}
	if g == nil {
		return fmt.Errorf("%s did not produce an a-grammar", grammarPath)
	}

	// Serialize the grammar before anything below mutates it.
	serialized := g.Serialize()

	// The emitter library is the start script up to the driver marker.
	startScript := r.GetStartScript(g)
	if startScript == nil {
		return fmt.Errorf("%s has no start script", grammarPath)
	}
	full := (*startScript.CodeChilds)[0].String
	markerPos := strings.Index(full, frozenDriverMarker)
	if markerPos < 0 {
		return fmt.Errorf("%s: start script has no %q marker", grammarPath, frozenDriverMarker)
	}
	lib := full[:markerPos]
	// The emitter library may pull shared code from tests/lib via include().
	// The bootstrap must be self contained (at -frozen startup nothing can
	// compile an included file yet - the compiler IS what is being started),
	// so the freezer inlines the files here, at freeze time.
	lib, err = inlineIncludes(lib, filepath.Dir(grammarPath))
	if err != nil {
		return err
	}

	// Collect every distinct script source of the grammar: the tag scripts
	// (all slots) and the inline :script() commands.
	var sources []string
	seen := map[string]bool{}
	var collect func(rules *r.Rules)
	collect = func(rules *r.Rules) {
		if rules == nil {
			return
		}
		for _, rule := range *rules {
			if rule.Operator == r.Tag || (rule.Operator == r.Command && rule.String == "script") {
				if rule.CodeChilds != nil {
					for _, code := range *rule.CodeChilds {
						if code.Operator == r.Token && !seen[code.String] {
							seen[code.String] = true
							sources = append(sources, code.String)
						}
					}
				}
			}
			collect(rule.Childs)
			if rule.Operator == r.Tag {
				collect(rule.CodeChilds) // Tags can nest rules inside CodeChilds in theory.
			}
		}
	}
	collect(g)
	if len(sources) == 0 {
		return fmt.Errorf("%s contains no tag scripts", grammarPath)
	}

	// Compose the bootstrap program.
	var b strings.Builder
	b.WriteString(lib)
	b.WriteString("\n// ----- frozen tag scripts -----\n")
	for i, code := range sources {
		fmt.Fprintf(&b, "function jstag_%d() {\n%s\n}\n", i, code)
	}
	b.WriteString("function main() {\n    var tags = {}\n")
	for i, code := range sources {
		fmt.Fprintf(&b, "    tags[%s] = jstag_%d\n", escapeMetaJSString(code), i)
	}
	b.WriteString("    return {module: m, tags: tags}\n}\n")
	bootstrapSrc := b.String()

	// Compile the bootstrap program with the grammar itself (under goja). The
	// driver tail of the start script is replaced: it must only build the
	// module and yield it as the completion value instead of running jsmain
	// and exiting.
	asgB, err := ParseWithAgrammar(g, bootstrapSrc, "jsbootstrap", opts)
	if err != nil {
		return fmt.Errorf("cannot parse the bootstrap program: %s\n----\n%s", err, bootstrapSrc)
	}
	(*startScript.CodeChilds)[0].String = lib + "\nc.compile(c.asg)\nm\n"
	modObj := compileASGInternal(asgB, g, "jsbootstrap", 0, false, true)
	mod, ok := modObj.(*ir.Module)
	if !ok {
		return fmt.Errorf("compiling the bootstrap program yielded %T instead of an IR module", modObj)
	}

	// Write the two snapshot files.
	agrammarSrc := "package abnf\n\nimport \"14.gy/mec/abnf/r\"\n\n" +
		"// Code generated by 'mec -freeze " + grammarPath + "'. DO NOT EDIT.\n" +
		"//\n" +
		"// jsAgrammar is the serialized a-grammar of " + grammarPath + " (the MetaJS to\n" +
		"// LLVM IR compiler). In frozen mode (-frozen) the Go core parses every\n" +
		"// annotation script with it - without goja.\n\n" +
		"var jsAgrammar = " + serialized + "\n"
	if err := ioutil.WriteFile(outDir+string(os.PathSeparator)+"jsagrammar.go", []byte(agrammarSrc), 0644); err != nil {
		return err
	}

	llSrc := "; Code generated by 'mec -freeze " + grammarPath + "'. DO NOT EDIT.\n" +
		";\n" +
		"; The frozen MetaJS bootstrap: the emitter library and the tag scripts of\n" +
		"; " + grammarPath + ", compiled to handle threaded IR by that grammar itself\n" +
		"; (running under goja). In frozen mode this module compiles every annotation\n" +
		"; script; goja is not needed anymore.\n\n" +
		mod.String()
	if err := ioutil.WriteFile(outDir+string(os.PathSeparator)+"jsbootstrap.ll", []byte(llSrc), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Frozen %d tag scripts of %s\n  => %s/jsagrammar.go\n  => %s/jsbootstrap.ll\nRebuild the binary to embed the snapshot.\n",
		len(sources), grammarPath, outDir, outDir)
	return nil
}

// inlineIncludes textually replaces whole line include("file") calls with the
// file's content, paths relative to dir (the grammar file's directory, the
// same base that the runtime include() uses).
func inlineIncludes(src string, dir string) (string, error) {
	re := regexp.MustCompile(`(?m)^[ \t]*include\("([^"]*)"\)[ \t]*$`)
	for depth := 0; ; depth++ {
		m := re.FindStringSubmatchIndex(src)
		if m == nil {
			return src, nil
		}
		if depth > 16 {
			return "", fmt.Errorf("include() nesting deeper than 16 levels (cycle?)")
		}
		name := src[m[2]:m[3]]
		dat, err := ioutil.ReadFile(dir + string(os.PathSeparator) + filepath.Clean(name))
		if err != nil {
			return "", fmt.Errorf("cannot inline include(%q): %s", name, err)
		}
		src = src[:m[0]] + "// ----- inlined include(\"" + name + "\") -----\n" +
			string(dat) + src[m[1]:]
	}
}

// escapeMetaJSString quotes s as a MetaJS double quoted string literal.
func escapeMetaJSString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			b.WriteString("\\\"")
		case c == '\\':
			b.WriteString("\\\\")
		case c == '\n':
			b.WriteString("\\n")
		case c == '\r':
			b.WriteString("\\r")
		case c == '\t':
			b.WriteString("\\t")
		case c < 0x20:
			fmt.Fprintf(&b, "\\x%02x", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
