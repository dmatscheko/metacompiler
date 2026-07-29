package abnf

import (
	"io"
	"os"
)

// outWriter is where the annotation scripts' print / println / printf (and the
// Python-style js_pyprint of the handle runtime) send their output. It defaults
// to standard output. SetOutput redirects it so that one pipeline stage's text
// output can be captured and fed as the program input of the next stage - the
// -pipe mode of main.go, which lets a language (e.g. a preprocessor) transform
// the source before another language consumes it.
var outWriter io.Writer = os.Stdout

// SetOutput redirects script output to w and returns the previous writer, so the
// caller can restore it afterwards (pass the returned value, or os.Stdout, back).
func SetOutput(w io.Writer) io.Writer {
	prev := outWriter
	outWriter = w
	return prev
}

// warnWriter is where DIAGNOSTICS go: the `eprintln` host function of the tag
// scripts and the two Kotlin runtime warnings of jsrtkotlin.go. It is standard
// error and, unlike outWriter, it is deliberately NOT redirectable:
//
//   - a warning must never land inside an emitted module (it would make the
//     module unparseable) or inside a program's own output, and
//   - it must never be captured by -pipe, which swaps outWriter to collect one
//     stage's text as the next stage's SOURCE - a warning in there would be fed
//     to the next grammar as program text.
//
// It is also not silenced by -q/-qq: those flags are about module and program
// output, and a warning is a diagnostic about the compile, not output.
var warnWriter io.Writer = os.Stderr
