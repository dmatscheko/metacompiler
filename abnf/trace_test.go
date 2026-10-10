package abnf

import "testing"

// A -trace stream is reproducible: a value with no rendering of its own (a
// scope held in a variable, a generator, a control signal) is named by its type,
// never by Go's %v, which prints heap addresses that differ run to run.
func TestTraceValNamesAnOpaquePointerByItsType(t *testing.T) {
	rt := &jsrt{}
	for v, want := range map[interface{}]string{
		&jsScope{}:                               "[jsScope]",
		&jsGenerator{}:                           "[jsGenerator]",
		&jsCtl{}:                                 "[jsCtl]",
		&jsArray{elems: []interface{}{1.0, "a"}}: "1,a", // a pointer WITH a rendering keeps it
		2.5:                                      "2.5",
	} {
		if got := rt.traceVal(v); got != want {
			t.Errorf("traceVal(%T) = %q, want %q", v, got, want)
		}
	}
}
