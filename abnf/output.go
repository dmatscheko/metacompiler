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
