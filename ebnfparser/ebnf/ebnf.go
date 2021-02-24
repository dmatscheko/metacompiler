package ebnf

import (
	"text/scanner"
)

// ----------------------------------------------------------------------------
// Internal representation

type (
	// An Expression node represents a production expression.
	Expression interface {
		// Pos is the position of the first character of the syntactic construct
		Pos() scanner.Position
	}

	// An Alternative node represents a non-empty list of alternative expressions.
	Alternative []Expression // x | y | z

	// A Sequence node represents a non-empty list of sequential expressions.
	Sequence []Expression // x y z

	// A Name node represents a production name.
	Name struct {
		StringPos scanner.Position
		String    string
	}

	// A Token node represents a literal.
	Token struct {
		StringPos scanner.Position
		String    string
	}

	// A Range is a List node that represents a range of characters.
	Range struct {
		Begin, End *Token // begin ... end
	}

	// A Group node represents a grouped expression.
	Group struct {
		Lparen scanner.Position
		Body   Expression // (body)
	}

	// An Option node represents an optional expression.
	Option struct {
		Lbrack scanner.Position
		Body   Expression // [body]
	}

	// A Repetition node represents a repeated expression.
	Repetition struct {
		Lbrace scanner.Position
		Body   Expression // {body}
	}

	// A Production node represents an EBNF production.
	Production struct {
		Name *Name
		Expr Expression
	}

	// A Bad node stands for pieces of source code that lead to a parse error.
	Bad struct {
		TokPos scanner.Position
		Error  string // parser error message
	}

	// A Grammar is a set of EBNF productions. The map
	// is indexed by production name.
	//
	Grammar map[string]*Production
)

func (x Alternative) Pos() scanner.Position { return x[0].Pos() } // the parser always generates non-empty Alternative
func (x Sequence) Pos() scanner.Position    { return x[0].Pos() } // the parser always generates non-empty Sequences
func (x *Name) Pos() scanner.Position       { return x.StringPos }
func (x *Token) Pos() scanner.Position      { return x.StringPos }
func (x *Range) Pos() scanner.Position      { return x.Begin.Pos() }
func (x *Group) Pos() scanner.Position      { return x.Lparen }
func (x *Option) Pos() scanner.Position     { return x.Lbrack }
func (x *Repetition) Pos() scanner.Position { return x.Lbrace }
func (x *Production) Pos() scanner.Position { return x.Name.Pos() }
func (x *Bad) Pos() scanner.Position        { return x.TokPos }
