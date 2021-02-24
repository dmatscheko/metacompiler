// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ebnf is a library for EBNF grammars. The input is text ([]byte)
// satisfying the following grammar (represented itself in EBNF):
//
//	Production  = name "=" [ Expression ] "." .
//	Expression  = Alternative { "|" Alternative } .
//	Alternative = Term { Term } .
//	Term        = name | token [ "…" token ] | Group | Option | Repetition .
//	Group       = "(" Expression ")" .
//	Option      = "[" Expression "]" .
//	Repetition  = "{" Expression "}" .
//
// A name is a Go identifier, a token is a Go string, and comments
// and white space follow the same rules as for the Go language.
// Production names starting with an uppercase Unicode letter denote
// non-terminal productions (i.e., productions which allow white-space
// and comments between tokens); all other production names denote
// lexical productions.
//
package main

import (
	"bytes"
	"fmt"

	"./ebnf" // import "golang.org/x/exp/ebnf"
)

var goodGrammars = []string{
	`Program = .`,

	`Program = foo .
	 foo = "foo" .`,

	`Program = "a" | "b" "c" .`,

	`Program = "a" … "z" .`,

	`Program = Song .
	 Song = { Note } .
	 Note = Do | (Re | Mi | Fa | So | La) | Ti .
	 Do = "c" .
	 Re = "d" .
	 Mi = "e" .
	 Fa = "f" .
	 So = "g" .
	 La = "a" .
	 Ti = ti .
	 ti = "b" .`,

	"Program = `\"` .",

	`Program = { Production } .
	Production  = name "=" [ Expression ] "." .
	Expression  = Alternative { "|" Alternative } .
	Alternative = Term { Term } .
	Term        = name | token [ "…" token ] | Group | Option | Repetition .
	Group       = "(" Expression ")" .
	Option      = "[" Expression "]" .
	Repetition  = "{" Expression "}" .
	
	name = "A" … "Z" { "a" … "z" | "0" … "9" | "_" } .

	token = "\""name"\"" .
	`,
}

var badGrammars = []string{
	`Program = | .`,
	`Program = | b .`,
	`Program = a … b .`,
	`Program = "a" … .`,
	`Program = … "b" .`,
	`Program = () .`,
	`Program = [] .`,
	`Program = {} .`,
}

// func checkGood(src string) {
// 	grammar, err := ebnf.Parse("", bytes.NewBuffer([]byte(src)))
// 	if err != nil {
// 		fmt.Printf("Parse(%s) failed:\n\n%v\n\n\n", src, err)
// 		return
// 	}
// 	if err = ebnf.Verify(grammar, "Program"); err != nil {
// 		fmt.Printf("Verify(%s) failed:\n\n%v\n\n\n", src, err)
// 	}

// 	fmt.Printf("Grammar(%s) is:\n\n%v\n\n\n", src, grammar)
// }

// func checkBad(src string) {
// 	_, err := ebnf.Parse("", bytes.NewBuffer([]byte(src)))
// 	if err != nil {
// 		fmt.Printf("Parse(%s) failed (and should have):\n\n%v\n\n\n", src, err)
// 		return
// 	}
// 	if err == nil {
// 		fmt.Printf("Parse(%s) should have failed\n", src)
// 	}
// }

// func testGrammars() {
// 	for _, src := range goodGrammars {
// 		checkGood(src)
// 	}
// 	for _, src := range badGrammars {
// 		checkBad(src)
// 	}
// }

func parseWithGrammar(srcEbnf string, srcCode string) {
	fmt.Printf("Input:\n\n%s\n\n\n", srcEbnf)

	grammar, err := ebnf.Parse("", bytes.NewBuffer([]byte(srcEbnf)))
	if err != nil {
		fmt.Printf("Parse failed:\n\n%v\n\n\n", err)
		return
	}
	if err = ebnf.Verify(grammar, "Program"); err != nil {
		fmt.Printf("Verify failed:\n\n%v\n\n\n", err)
	}

	err = ebnf.ParseWithGrammar(grammar, "Program", "", bytes.NewBuffer([]byte(srcCode)))
	if err != nil {
		fmt.Printf("Parse with grammar failed:\n\n%v\n\n\n", err)
	}
}

func main() {
	// testGrammars()

	parseWithGrammar(goodGrammars[6], goodGrammars[6])
}
