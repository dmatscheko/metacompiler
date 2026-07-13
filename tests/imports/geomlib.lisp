; A project library loaded by tests/lisp-test-multifile.txt via (load "geomlib.lisp")
; (found under the -i include root: mec -i tests/imports ...). Lisp has no import form, so
; loading parses this file with the same grammar and registers its (define ...)s before the
; main file's forms run, so the main file can call these functions directly. The
; interpreter (lisp-interpreter.abnf) and the LLVM-IR compiler (lisp-to-llvm-ir.abnf) both
; load it and must agree. Integer only (32-bit); products stay well under 2^31 and gcd only
; takes mod when the divisor is non-zero.

(define (square x) (* x x))

(define (vec-dot ax ay bx by)           ; ax*bx + ay*by
  (+ (* ax bx) (* ay by)))

(define (vec-len2 x y)                  ; squared length = x*x + y*y
  (+ (square x) (square y)))

(define (gcd a b)                       ; Euclid's algorithm (recursive)
  (if (= b 0)
      a
      (gcd b (mod a b))))
