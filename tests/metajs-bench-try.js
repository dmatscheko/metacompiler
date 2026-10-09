// The exception-path allocation benchmark. Every js_try needs a jmp_buf; the
// buffers are pooled per DEPTH, so this loop must not grow beyond the ordinary
// arena growth (a per-ENTRY allocation would leak about 1.5 KB per iteration).
//
// The answer is deterministic, so ./test.sh can compare the two engines on it;
// tests/bench-alloc.sh runs it natively and reports time and peak RSS.
function main() {
	var s = 0
	var i = 0
	while (i < 20000) {
		try {
			if (i % 3 == 0) { throw i }
			s = s + 1
		} catch (e) {
			s = s + 2
		} finally {
			s = s + 1
		}
		i = i + 1
	}
	println(s)
	return 0
}
