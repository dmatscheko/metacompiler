module 14.gy/mec

go 1.16

require (
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/dop251/goja v0.0.0-20210317175251-bb14c2267b76
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/llir/llvm v0.3.2
)

// goja at the pinned revision, in-tree so that it can carry local fixes.
replace github.com/dop251/goja => ./third_party/goja
