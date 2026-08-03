; gen.ll - the coroutine proof of concept, docs/runtime-next-plan.md
; ("Coroutines - the fifth-language wall").
;
; This is ONE module, run in BOTH worlds:
;
;   llvm.Run   abnf/coropoc_test.go parses it and hands it to runJSModule, so
;              js_genfn / js_yield are abnf/jsrt.go's goroutine + two channels
;              and a handle is an index into a Go table.
;   native     tests/coro-poc/build.sh links it with the PATCHED C floor, so
;              js_genfn / js_yield are a pthread + one condition variable and a
;              handle is a Cell *.
;
; It is hand-written rather than emitted because no grammar in this repository
; both EMITS js_genfn and has a layer 2 that can be linked natively today - php
; is the only one with a native layer 2, and its js_genfn is the loud stub this
; investigation exists to remove.  The shapes below are exactly what
; php-to-llvm-ir.abnf, js-to-llvm-ir.abnf, typescript-, python- and
; csharp-to-llvm-ir.abnf already emit: js_genfn over a js_closure, js_yield in
; the body, and the iterator's own `next` read with js_get.
;
; What it exercises, in order:
;   A  a FINITE generator driven five times: three yields, the return value in
;      the {value, done} record, and a next() past the end.
;   B  an INFINITE generator driven five times with several thousand
;      allocations between resumes - eager draining cannot do this, and the
;      allocations force collections while a coroutine stack is suspended.
;   C  a handle that lives ONLY in the suspended body's frame across a yield,
;      read back after more garbage.  Under MEC_GC=stress this is the
;      assertion that the collector scans a suspended coroutine's stack.

@str.println = global [7 x i8] c"println"
@str.next = global [4 x i8] c"next"
@str.value = global [5 x i8] c"value"
@str.done = global [4 x i8] c"done"

declare i64 @js_scope_new(i64 %0)
declare i64 @js_scope_get(i64 %0, i64 %1)
declare i64 @js_str_mem(i8* %0, i64 %1)
declare i64 @js_num_i(i64 %0)
declare i64 @js_arr_new()
declare i64 @js_arr_push(i64 %0, i64 %1)
declare i64 @js_call(i64 %0, i64 %1, i64 %2)
declare i64 @js_get(i64 %0, i64 %1)
declare i64 @js_closure(i64 %0, i64 %1)
declare i64 @js_genfn(i64 %0)
declare i64 @js_yield(i64 %0)

; ---------------------------------------------------------------- helpers --

define i64 @po_println(i64 %pf, i64 %v) {
entry:
	%a = call i64 @js_arr_new()
	%p = call i64 @js_arr_push(i64 %a, i64 %v)
	%r = call i64 @js_call(i64 %pf, i64 0, i64 %a)
	ret i64 0
}

; g.next() - read the member and call it, exactly as an emitter would.
define i64 @po_next(i64 %g) {
entry:
	%k = call i64 @js_str_mem(i8* getelementptr ([4 x i8], [4 x i8]* @str.next, i64 0, i64 0), i64 4)
	%n = call i64 @js_get(i64 %g, i64 %k)
	%a = call i64 @js_arr_new()
	%r = call i64 @js_call(i64 %n, i64 %g, i64 %a)
	ret i64 %r
}

; print r.value and r.done
define i64 @po_show(i64 %pf, i64 %r) {
entry:
	%kv = call i64 @js_str_mem(i8* getelementptr ([5 x i8], [5 x i8]* @str.value, i64 0, i64 0), i64 5)
	%v = call i64 @js_get(i64 %r, i64 %kv)
	%x = call i64 @po_println(i64 %pf, i64 %v)
	%kd = call i64 @js_str_mem(i8* getelementptr ([4 x i8], [4 x i8]* @str.done, i64 0, i64 0), i64 4)
	%d = call i64 @js_get(i64 %r, i64 %kd)
	%y = call i64 @po_println(i64 %pf, i64 %d)
	ret i64 0
}

; n throwaway arrays, so a collection happens while a coroutine is suspended.
define i64 @po_garbage(i64 %n) {
entry:
	br label %loop

loop:
	%i = phi i64 [ 0, %entry ], [ %i2, %loop ]
	%g = call i64 @js_arr_new()
	%i2 = add i64 %i, 1
	%c = icmp slt i64 %i2, %n
	br i1 %c, label %loop, label %out

out:
	ret i64 0
}

; --------------------------------------------------------- generator bodies

; A: yield 1; yield 2; yield 3; return 99
define i64 @jsf_1(i64 %env, i64 %args) {
entry:
	%n1 = call i64 @js_num_i(i64 1)
	%y1 = call i64 @js_yield(i64 %n1)
	%n2 = call i64 @js_num_i(i64 2)
	%y2 = call i64 @js_yield(i64 %n2)
	%n3 = call i64 @js_num_i(i64 3)
	%y3 = call i64 @js_yield(i64 %n3)
	%r = call i64 @js_num_i(i64 99)
	ret i64 %r
}

; B: i = 0; while (true) { yield i * 10; i = i + 1 }  - it never returns.
define i64 @jsf_2(i64 %env, i64 %args) {
entry:
	br label %loop

loop:
	%i = phi i64 [ 0, %entry ], [ %i2, %loop ]
	%t = mul i64 %i, 10
	%v = call i64 @js_num_i(i64 %t)
	%y = call i64 @js_yield(i64 %v)
	%i2 = add i64 %i, 1
	br label %loop
}

; C: a handle reachable ONLY from this frame, held across a yield.
define i64 @jsf_3(i64 %env, i64 %args) {
entry:
	%a = call i64 @js_arr_new()
	%v42 = call i64 @js_num_i(i64 42)
	%p1 = call i64 @js_arr_push(i64 %a, i64 %v42)
	%v7 = call i64 @js_num_i(i64 7)
	%p2 = call i64 @js_arr_push(i64 %a, i64 %v7)
	%one = call i64 @js_num_i(i64 1)
	%y = call i64 @js_yield(i64 %one)
	%k0 = call i64 @js_num_i(i64 0)
	%e0 = call i64 @js_get(i64 %a, i64 %k0)
	%y2 = call i64 @js_yield(i64 %e0)
	%k1 = call i64 @js_num_i(i64 1)
	%e1 = call i64 @js_get(i64 %a, i64 %k1)
	ret i64 %e1
}

; D: a body that ALLOCATES HEAVILY before its first yield, so the collection it
; triggers runs while the RESUMER's stack is parked.
define i64 @jsf_4(i64 %env, i64 %args) {
entry:
	%q = call i64 @po_garbage(i64 4000)
	%one = call i64 @js_num_i(i64 1)
	%y = call i64 @js_yield(i64 %one)
	ret i64 %one
}

; The function table the -exe path needs: js_closure stores a raw index and the
; C floor turns it back into a call through here.
define i64 @jsdispatch(i64 %idx, i64 %env, i64 %args) {
entry:
	switch i64 %idx, label %bad [ i64 1, label %f1
	                              i64 2, label %f2
	                              i64 3, label %f3
	                              i64 4, label %f4 ]

f1:
	%r1 = call i64 @jsf_1(i64 %env, i64 %args)
	ret i64 %r1

f2:
	%r2 = call i64 @jsf_2(i64 %env, i64 %args)
	ret i64 %r2

f3:
	%r3 = call i64 @jsf_3(i64 %env, i64 %args)
	ret i64 %r3

f4:
	%r4 = call i64 @jsf_4(i64 %env, i64 %args)
	ret i64 %r4

bad:
	ret i64 0
}

; ------------------------------------------------------------------- main --

define i64 @jsmain(i64 %env, i64 %args) {
entry:
	%sc = call i64 @js_scope_new(i64 %env)
	%kp = call i64 @js_str_mem(i8* getelementptr ([7 x i8], [7 x i8]* @str.println, i64 0, i64 0), i64 7)
	%pf = call i64 @js_scope_get(i64 %sc, i64 %kp)

	; --- A: a finite generator, five next() calls -----------------------
	%c1 = call i64 @js_closure(i64 1, i64 %sc)
	%gf1 = call i64 @js_genfn(i64 %c1)
	%a1 = call i64 @js_arr_new()
	%g1 = call i64 @js_call(i64 %gf1, i64 0, i64 %a1)
	%ra = call i64 @po_next(i64 %g1)
	%sa = call i64 @po_show(i64 %pf, i64 %ra)
	%rb = call i64 @po_next(i64 %g1)
	%sb = call i64 @po_show(i64 %pf, i64 %rb)
	%rc = call i64 @po_next(i64 %g1)
	%scw = call i64 @po_show(i64 %pf, i64 %rc)
	%rd = call i64 @po_next(i64 %g1)
	%sd = call i64 @po_show(i64 %pf, i64 %rd)
	%re = call i64 @po_next(i64 %g1)
	%se = call i64 @po_show(i64 %pf, i64 %re)

	; --- B: an infinite generator, pulled five times --------------------
	%c2 = call i64 @js_closure(i64 2, i64 %sc)
	%gf2 = call i64 @js_genfn(i64 %c2)
	%a2 = call i64 @js_arr_new()
	%g2 = call i64 @js_call(i64 %gf2, i64 0, i64 %a2)
	%q0 = call i64 @po_garbage(i64 3000)
	%rf = call i64 @po_next(i64 %g2)
	%sf = call i64 @po_show(i64 %pf, i64 %rf)
	%q1 = call i64 @po_garbage(i64 3000)
	%rg = call i64 @po_next(i64 %g2)
	%sg = call i64 @po_show(i64 %pf, i64 %rg)
	%q2 = call i64 @po_garbage(i64 3000)
	%rh = call i64 @po_next(i64 %g2)
	%sh = call i64 @po_show(i64 %pf, i64 %rh)
	%q3 = call i64 @po_garbage(i64 3000)
	%ri = call i64 @po_next(i64 %g2)
	%si = call i64 @po_show(i64 %pf, i64 %ri)
	%q4 = call i64 @po_garbage(i64 3000)
	%rj = call i64 @po_next(i64 %g2)
	%sj = call i64 @po_show(i64 %pf, i64 %rj)

	; --- C: a handle live only in the suspended frame -------------------
	%c3 = call i64 @js_closure(i64 3, i64 %sc)
	%gf3 = call i64 @js_genfn(i64 %c3)
	%a3 = call i64 @js_arr_new()
	%g3 = call i64 @js_call(i64 %gf3, i64 0, i64 %a3)
	%rk = call i64 @po_next(i64 %g3)
	%sk = call i64 @po_show(i64 %pf, i64 %rk)
	%q5 = call i64 @po_garbage(i64 5000)
	%rl = call i64 @po_next(i64 %g3)
	%sl = call i64 @po_show(i64 %pf, i64 %rl)
	%q6 = call i64 @po_garbage(i64 5000)
	%rm = call i64 @po_next(i64 %g3)
	%sm = call i64 @po_show(i64 %pf, i64 %rm)

	; --- D: a value live ONLY on the RESUMER's stack while a body allocates --
	%hold = call i64 @js_arr_new()
	%v1234 = call i64 @js_num_i(i64 1234)
	%ph = call i64 @js_arr_push(i64 %hold, i64 %v1234)
	%c4 = call i64 @js_closure(i64 4, i64 %sc)
	%gf4 = call i64 @js_genfn(i64 %c4)
	%a4 = call i64 @js_arr_new()
	%g4 = call i64 @js_call(i64 %gf4, i64 0, i64 %a4)
	%rn = call i64 @po_next(i64 %g4)
	%sn = call i64 @po_show(i64 %pf, i64 %rn)
	%kh = call i64 @js_num_i(i64 0)
	%eh = call i64 @js_get(i64 %hold, i64 %kh)
	%so = call i64 @po_println(i64 %pf, i64 %eh)

	ret i64 0
}
