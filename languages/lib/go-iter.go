// Package iter: Go's iter package WRITTEN IN GO, compiled with a program that
// imports "iter" (core.importFile in both go grammars). A range over an iter.Seq
// or iter.Seq2 calls it with a yield function (go-constfold.js, goIsSeqTy). Pull
// and Pull2 are not here.
package iter

type Seq[V any] func(yield func(V) bool)

type Seq2[K, V any] func(yield func(K, V) bool)
