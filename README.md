# MetaCompiler

A generic compiler frontend.

This is
* a parser that parses a language specification written in annotated EBNF
* and a compiler that (at runtime) compiles the parsing result to a new parser and compiler.

The new parser and compiler are equivalent to the first ones and can replace them at runtime or can be used in parallel.

This means it is a parser and compiler with a polymorphic core.

This system should allow to define and use compiler for arbitrary computer languages, only by specifying them in plain text in annotated EBNF.

## Table of Contents
- [MetaCompiler](#metacompiler)
  - [Table of Contents](#table-of-contents)
  - [What is an annotated EBNF (ABNF)?](#what-is-an-annotated-ebnf-abnf)
  - [Small Example](#small-example)
  - [Documentation](#documentation)
    - [Build / Usage](#build--usage)
    - [High level overview](#high-level-overview)
      - [Default process steps](#default-process-steps)
    - [Exposed JS API](#exposed-js-api)
      - [General](#general)
      - [Output](#output)
      - [Strings](#strings)
      - [Variables](#variables)
        - [Local variables](#local-variables)
        - [Global variables](#global-variables)
      - [The stacks](#the-stacks)
        - [Local stacks](#local-stacks)
        - [Global stack](#global-stack)
      - [Compiler API](#compiler-api)
      - [Parser and Compiler ABNF a-grammar API](#parser-and-compiler-abnf-a-grammar-api)
        - [Builder functions](#builder-functions)
        - [Grammar functions](#grammar-functions)
        - [Text functions](#text-functions)
        - [OperatorID Constants](#operatorid-constants)
      - [LLVM IR API](#llvm-ir-api)
    - [ABNF Syntax](#abnf-syntax)
      - [EBNF of EBNF](#ebnf-of-ebnf)
      - [EBNF of ABNF](#ebnf-of-abnf)
      - [Common syntax](#common-syntax)
  - [Further Examples](#further-examples)
    - [Example of an annotated EBNF](#example-of-an-annotated-ebnf)
    - [Its output, when it gets applied on itself:](#its-output-when-it-gets-applied-on-itself)
  - [Links](#links)

## What is an annotated EBNF (ABNF)?

* The EBNF defines the syntax (the grammar) of another language.
* The annotations in the EBNF define the semantic (the meaning) of the other language.
* The combined format is therefore called annotated EBNF or ABNF.

This means ABNF is a meta language. A language to describe the syntax and semantic of another language.

## Small Example

This is a fully working calculator for integer addition and multiplication. It can parse its input and calculate the output, all while taking into account point before line calculation and bracketing:

<details>
  <summary>Click to expand!</summary>

```javascript
"Tiny calculator"
<~~c.compile(c.asg)~~>

{

Expression  =
    Term
    {
        ( "+" | "-" )           <~~pushg(up.in)~~>
        Term                    <~~
                                var t2 = popg()
                                var op = popg()
                                var t1 = popg()
                                println("calculating " + t1 + " " + op + " " + t2)
                                var res = (op=="+") ? t1 + t2 : t1 - t2
                                pushg(res)
                                ~~>
    }
    ;

Term        =
    Factor
    {
        ( "*" | "/" )           <~~pushg(up.in)~~>
        Factor                  <~~
                                var t2 = popg()
                                var op = popg()
                                var t1 = popg()
                                println("calculating " + t1 + " " + op + " " + t2)
                                var res = (op=="*") ? t1 * t2 : t1 / t2
                                pushg(res)
                                ~~>
    }
    ;

Factor      =
    (
        "("
        Expression
        ")"
    )
    |
    Number                      <~~pushg(parseFloat(up.in))~~>
    ;

Number      = ( "0" | "1"..."9" { "0"..."9" } )
              [ ( "." | "," ) "0"..."9" { "0"..."9" } ] ;

}

<~~println("\nRESULT: " + popg() + "\nFormula was: " + ltr.in)~~>

Expression
```

When fed the input `1 + 3 * (3 + (7 - 1) * 2)`, it outputs `46`.
</details>

## Documentation

### Build / Usage

```
go build .
./mec -a tests/brainfuck_a.bnf -b tests/brainfuck-test-2.txt
```
or
```
go run . -a tests/abnf-of-abnf_a.bnf -b tests/abnf-of-abnf_a.bnf
```

### High level overview

There are two basic components: A __parser__ and a __compiler__.

The __parser__ processes the target text top down, so from the topmost grammar rule, over its currently matching branches, to the leaves that match. The leaves are fixed strings that must match (also called token or terminal symbols). The parser thereby generates an ASG (abstract semantic graph). This is similar to an AST (abstract syntax tree), but the parser attaches code, provided by special grammar rules (`Tags`) to the group of matching strings (`Token`). Because of those special rules, the grammar is called annotated grammar or a-grammer.

The only hirarchy or grouping inside the ASG is done by the `Tag` rules. Each `Tag` can contain multiple other `Tags` and of course the strings (`Token`) that were found in the target text. The `Tags` contain those child `Tags` and `Token` in the `Sequence` of their occurence in the target text.

The __compiler__ processes the ASG, generated by the parser, bottom up. It first finds the outermost leaves and collects the data from there by executing attached code. This data is accumulated until the compiler reaches the topmost point. There the code inside the a-grammer can decide what to do with the collected data and how to represent it.

#### Default process steps

This is the input to the default main process:
* `initial-a-grammar` = Default annotated grammar of annotated EBNF.
* `inputA` = The content of the file given with command line parameter `-a`.
* `inputB` = The content of the file given with command line parameter `-b`.
* `*-a-grammar-prolog` = The code at the beginning of the grammar. This code starts the rest of the compile process during the compile step.

This is how that input is processed:
1. `parse(initial-a-grammar, inputA)`  = `inputA-ASG`.
2. `compile(inputA-ASG, initial-a-grammar-prolog)`  = `new-a-grammar`.
3. `parse(new-a-grammar, inputB)`  = `inputB-ASG`.
4. `compile(inputB-ASG, new-a-grammar-prolog)`  = `result`.

Of course, the `result` can again (but doesn't have to) be an `a-grammar` and can be used as input for `parse()`.

An example of this process, done fully inside the prolog code of an ABNF:

<details>
  <summary>Click to expand!</summary>

```javascript
"Parse and compile test"
<~~

// Lets build a new compiler directly from scratch:

// For this we need an agrammar. This is the information, that the metacompiler needs to actually be a compiler.

// To create an agrammar, we need an ABNF:
var ABNF = "<~\~ println('found me'); c.compile(c.asg) ~\~> {X = ( 'A' | 'B' | 'C' ) <~\~ println('and found the letter ' + up.in) ~\~>; } X"

// Now we parse the ABNF and create an ASG from it. We use c.ABNFagrammar as agrammar for this parse step, because we wrote our compiler definition in ABNF (the variable above), and the agrammar c.ABNFagrammar understands ABNF:
var ASG = c.parse(c.ABNFagrammar, ABNF)

// Lets print the resulting ASG:
println("\nASG: " + abnf.serializeRules(ASG))

// So far we have only parsed our ABNF into an ASG. Now we have to compile the ASG into something useful. We again use the agrammar c.ABNFagrammar, because this is - as stated above - the agrammar that understands ABNF:
var result = c.compileWithProlog(ASG, c.ABNFagrammar)

// The agrammar c.ABNFagrammar stores its compilation result into ltr.agrammar. The result of the compilation is the whole ltr object. Lets get our new very own agrammar:
var myAgrammar = result.agrammar

// Lets now print the resulting agrammar:
println("\nmyAgrammar: " + abnf.serializeRules(myAgrammar))

// We can now use our own agrammar to parse something new. We parse the letter B and again get an ASG from it:
var myASG = c.parse(myAgrammar, "B")

// Lets now print the resulting ASG:
println("\nmyASG: " + abnf.serializeRules(myASG))

// And to see a result, we have to compile our new ASG myASG with our grammar myAgrammar:
println("\nOutput:")
c.compileWithProlog(myASG, myAgrammar)

// Note that the ASG already includes almost all information for compiling. The agrammar is only used for its preamble. In our case this would be the Tag: "<~\~ println('found me'); c.compile(c.asg) ~\~>

~~>
{
    # No rules
}
```
</details>

### Exposed JS API

The annotations of the ABNF can contain JS code. The ASG (abstract semantic graph) gets processed from the leaves up to the stem. If annotations are encountered on the way, their JS code gets executed.

#### General

* __exit(v int)__  
Terminates the application and returns `v`.
* __sleep(d int)__  
Sleeps for `d` milliseconds.

#### Output

* __print(...)__ [fmt.Print](https://golang.org/pkg/fmt/#Print)
* __println(...)__ [fmt.Println](https://golang.org/pkg/fmt/#Println)
* __printf(...)__ [fmt.Printf](https://golang.org/pkg/fmt/#Printf)
* __sprintf(...)__ [fmt.Sprintf](https://golang.org/pkg/fmt/#Sprintf)

#### Strings

* __unescape(s string) string__  
Backslash unescapes a string. Necessary for tokens, but not for tags. It is about the inverse of `printf("%q", s)` but without the quotation marks.

#### Variables

* __append(a []object, v1 object, ...) []object__  
The function appends the objects `v1` ... `vn` to the array `a` and returns the combined array.

##### Local variables

* __up__ (for upstream)  
  All local variables. All `up.*` variables can be changed by the user. This includes `up.in`.
  * __up.in__  
  (string) The collective matched strings of all child nodes.
  * __up.\*__  
  User generated local variables. They can be arbitrary objects. Those objects are concatenated to arrays of objects when being propagated upwards.
  * __up.str\*__  
  User generated local variables. All variables that start with `str` must be strings. Those objects are concatenated as strings when being propagated upwards. up.in is an example of such string concatenation.
  * __up.arr\*__  
  User generated local variables. Those can be arrays of arbitrary objects. They are appended when being propagated upwards. If an object is not an array, it will be put into one.
  * __up.stack__  
  See [Local stacks](#local-stacks)

##### Global variables

* __ltr__ (for left to right)  
  All global variables (global JS variables can be used too). All `ltr.*` variables can be changed by the user. This includes `ltr.in`.
  * __ltr.in__  
  (string) The collective matched strings of all nodes from left to right. Only matched strings of nodes to the right (that are not processed yet), are not included.  
  * __ltr.\*__  
  User generated global variables. They can be arbitrary objects. Except for `ltr.in`, those objects are not changed by the compiler.
  * __ltr.stack__  
  See [Global stack](#global-stack)

#### The stacks

This API provides multiple local (LIFO) stacks and one global (LIFO) stack.

##### Local stacks

Each leaf starts with its own local stack. This stacks are combined hirarchically like the local variables of `up.arr`.

* __pop() object__  
Pops an arbitrary object from the local stack.
* __push(v object)__  
Pushes an arbitrary object onto the local stack.
* __up.stack__  
This stack can also be accessed via the variable `up.stack`.

##### Global stack

This one global stack is useful to e.g. bring data from one `Term` to a sibling. It is like the `ltr.*` variables:

* __popg() object__  
Pops an arbitrary object from the stack.
* __pushg(v object)__  
Pushes an arbitrary object onto the stack.
* __ltr.stack__  
This stack can also be accessed via the variable `ltr.stack`.

#### Compiler API

* __c.agrammar__  
The grammar that produced the current state, that the JS is executed in.
* __c.ABNFagrammar__  
A grammar that can parse and compile ABNF. This is the default initial grammar for the tool.
* __c.parse(agrammar []Rule, target string) []Rule__  
Parses the string `target` with the `agrammar` and returns an ASG.
* __c.asg__  
The whole abstract semantic graph.
* __c.localAsg__  
The local part of the abstract semantic graph.
* __c.compile(asg []Rule, slot int) map[string]object__  
Compiles the given ASG and returns the map of the combined upstream variables.  
Normally, `c.compile()` is called as `c.compile(c.asg)`.<br/>
The parameter `slot` states the index of the code part inside the `Tags`. It is normally 0.<br/>
The compiler works like this:  
```
    OUT
     ^
     |
     C---.      (C) If the current Rule has childs, the childs get sent to 'compile()'. (Also the childs of TAG Rules.)
     |    |
     ^    v
     *    |     (*) All upstream (up.*) values from returning 'compile()'s are combined.
    /|    |
   | | _  |
   T | |  |     (T) The text of a EBNF Terminal symbol (Token) gets returned and included into 'up.in'.
   | X |  |     (X) The script of a single TAG Rule script gets executed. This is after their childs came back from being splitted at (C).
   | | O  |     (O) Other Rules are ignored.
   | | |  |
   \ | /  |
    \|/   |
     *    |     (*) Childs from one Rule get splitted. The splitted path always only processe one Rule (That can contain childs).
     |    |
     ^    |
     IN<-'
```
* __c.compileWithProlog(asg []Rule, aGrammar []Rule, slot int) map[string]object__  
Instantiates a new compiler with `asg` and `aGrammar` and starts the prolog code of the `aGrammar`. This prolog code has to do the rest, to compile the ASG in parameter `asg`. Specifically, it has to call `c.compile(c.asg)` to compile that ASG. <br/>
The parameter `slot` states the index of the code part inside the `Tags`. It is normally 0.

#### Parser and Compiler ABNF a-grammar API

The a-grammar can be built from within JS. For this, some simple builder funcions are exposed:

##### Builder functions

* __abnf.arrayToRules(rules []object) []Rule__
* __abnf.newRule(Operator OperatorID, String string, Int int, Bool bool, Rune rune, Pos int, Childs []Rule, TagChilds []Rule) Rule__
* __abnf.newToken(String string, Pos int) Rule__
* __abnf.newName(String string, Int int, Pos int) Rule__
* __abnf.newProduction(String string, Int int, Childs []Rule, Pos int) Rule__
* __abnf.newTag(TagChilds []Rule, Childs []Rule, Pos int) Rule__
* __abnf.newSkipSpace(Bool bool, Pos int) Rule__
* __abnf.newRepetition(Childs []Rule, Pos int) Rule__
* __abnf.newOption(Childs []Rule, Pos int) Rule__
* __abnf.newGroup(Childs []Rule, Pos int) Rule__
* __abnf.newSequence(Childs []Rule, Pos int) Rule__
* __abnf.newAlternative(Childs []Rule, Pos int) Rule__
* __abnf.newRange(Childs []Rule, Pos int) Rule__

##### Grammar functions

* __abnf.getStartRule(rules []Rule) Rule__  
Returns the start rule of the a-grammar. The start rule points to the top level production.
* __abnf.getProductions(rules []Rule) []Rule__  
Returns the productions of the a-grammar.
* __abnf.getProlog(rules []Rule) Rule__  
Returns the prolog `Tag` of the a-grammar. This contains the JS code that controls the rest.
* __abnf.getTitle(rules []Rule) []Rule__  
Returns the title of the a-grammar.
* __abnf.getDescription(rules []Rule) []Rule__  
Returns the description of the a-grammar.

##### Text functions

* __abnf.serializeRule(r Rule)__
* __abnf.serializeRules(rs []Rule)__

##### OperatorID Constants

* __abnf.oid.Error__
* __abnf.oid.Success__
* __abnf.oid.Sequence__
* __abnf.oid.Group__
* __abnf.oid.Token__
* __abnf.oid.Or__
* __abnf.oid.Optional__
* __abnf.oid.Repeat__
* __abnf.oid.Range__
* __abnf.oid.SkipSpace__
* __abnf.oid.Tag__
* __abnf.oid.Production__
* __abnf.oid.Ident__

#### LLVM IR API

This tool uses the [Go LLIR/LLVM library](https://github.com/llir/llvm) to create LLVM IR and to interact with it. The API documentation can be found here: [llir/llvm overview](https://llir.github.io/document/) and here [LLIR/LLVM library documentation](https://pkg.go.dev/github.com/llir/llvm/). For more information on LLVM IR go to the [LLVM language reference](https://llvm.org/docs/LangRef.html).

The functions and constants are exposed to JS as:

* __llvm.ir.\*__ [llvm.ir](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir)
* __llvm.constant.\*__ [llvm.constant](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/constant)
* __llvm.metadata.\*__ [llvm.metadata](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/metadata)
* __llvm.types.\*__ [llvm.types](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/types)
* __llvm.enum.\*__ [llvm.enum](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/enum)

* Custom functions:
  * __llvm.Callgraph(m ir.Module) string__  
  The function `llvm.Callgraph(m ir.Module) string` creates the callgraph of the given LLVM IR module in Graphviz DOT format (can be viewed e.g. online with the [Graphviz visual editor](http://magjac.com/graphviz-visual-editor/)).
  * __llvm.Callgraph(m ir.Module, f string)__  
  The function `llvm.Callgraph(m ir.Module, f string)` tries to execute the function `f` inside the IR module `m` and returns the resulting uint32.

### ABNF Syntax

#### EBNF of EBNF

A normal EBNF syntax looks like this:

```javascript
"EBNF of EBNF" {

EBNF        = [ Title ] "{" { Production } "}" [ Comment ] ;
Production  = name "=" [ Expression ] ";" ;
Expression  = Alternative { "|" Alternative } ;
Alternative = Term { Term } ;
Term        = name | token [ "..." token ] | Group | Option | Repetition | skipspaces ;
Group       = "(" Expression ")" ;
Option      = "[" Expression "]" ;
Repetition  = "{" Expression "}" ;
Title       = token ;
Comment     = token ;

}
EBNF
```

Skip and noskip are additions to be able to parse strings correctly.

#### EBNF of ABNF

Annotated EBNF basically only adds tags to the syntax of a normal EBNF:

```javascript
"EBNF of ABNF" {

AEBNF       = [ Title ] [ Tag ] "{" { Production } "}" [ Tag ] [ Comment ] ;
Production  = name [ Tag ] "=" [ Expression ] ";" ;
Expression  = Alternative { "|" Alternative } ;
Alternative = Term { Term } ;
Term        = ( name | token [ "..." token ] | Group | Option | Repetition | skipspaces ) [ Tag ] ;
Group       = "(" Expression ")" ;
Option      = "[" Expression "]" ;
Repetition  = "{" Expression "}" ;
Title       = token ;
Comment     = token ;
Tag         = "<" code { "," code } ">" ;

}
AEBNF
```

The `Tag` is always responsible for the `Term` right before it.

The only exceptions are:
* `Repetition`, where the `Tag` would only attach to the last entry of the `Repetition` (use bracketing when you want to tag more).
* `Range`, where the `Tag` would also only attach to the second `Token` of the `Range` term.
* The `Tag` after the `Title`. There, the `Tag` is not responsible for the title but it contains the _prolog JS code_. That code is executed automatically.
* The `Tag` before the `Comment`. That `Tag` contains the _epilog JS code_. That code is executed automatically after the `c.compile()` function is finished with the ASG.

#### Common syntax

This is the definition of `name` and `token`, of `skipspaces`, and of `code`. Except of `code`, they are common to EBNF and ABNF:

```javascript
"Common syntax" {

name        = ( Small | Caps ) - { Small | Caps | Digit | "_" } + ;
token       = Dquotetoken | Squotetoken ;

code        = '~~' - { [ "~" ] Codeinner } '~~' + ;
Codeinner   = Small | Caps | Digit | Special | "'" | '"' | "\\~" ;

Dquotetoken = '"' - { Small | Caps | Digit | Special | "~" | "'" | '\\"' } '"' + ;
Squotetoken = "'" - { Small | Caps | Digit | Special | "~" | '"' | "\\'" } "'" + ;

Digit       = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
Small       = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" |
              "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
Caps        = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" |
              "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
Special     = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" |
              "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "|" | "%" | "$" | "&" | "#" |
              "@" | "\\\\" | "\\t" | "\t" | "\\n" | "\n" | "\\r" | "\r" ;

skipspaces  = Skip | Noskip ;
Skip        = ":skip(Whitespace)" ;  // Skip all whitespace in the future.
Noskip      = ":skip(Nothing)" ;  // Do not skip whitespace in the future.

}
```

As can be seen in the above EBNF, a `token` consists of one backslash escaped string, quoted in single or double quotes.
```
"This is an\nexample\tof a multiline string with one tab"
```
A `tag` starts with `<`, ends with `>`, and contain one or multiple comma separated raw strings, quoted in `~~` (two on either side). Inside a raw tag string, `\~` is a special symbol for `~` to be able to write a literal `~~` combination. Single `~` can be written without a backslash escape.
```
< ~~This is an
example of a multiline
raw string (inside a tag)
with one tilde: ~
and then two tildes: ~\~~~, ~~This is a second string inside the Tag~~ >
```

## Further Examples

### Example of an annotated EBNF

<details>
  <summary>Click to expand!</summary>

```javascript
:title("ABNF of ABNF to a-grammar") ;
:description("This ABNF contains the grammatic and semantic information for annotated EBNF.
It allows to automatically create a compiler for everything described in ABNF (yes, that format).") ;


// --- main rules

:startRule(ABNF) ;
// This is a parser command that sets the possible white space.
:skip(Whitespace) ;

// This is the start rule.
ABNF        = { Production | LineCommand } ;

Production  = Name <~~ var prodTag=undefined; var prodExpression=undefined; pushg(pop()) ~~> [ Tag <~~ prodTag=pop() ~~> ] "=" [ Expression <~~ prodExpression=pop() ~~> ] ";" <~~  pushg(buildProduction(popg(), prodTag, prodExpression)) ~~> ;

Expression  = Alternative ;

Alternative <~~ push(simplify(abnf.newAlternative(popg(), c.Pos))) ~~>
            = Sequence <~~ pushg([pop()]) ~~> { "|" Sequence <~~ pushg(append(popg(), pop())) ~~> } ;

Sequence    <~~ push(simplify(abnf.newSequence(popg(), c.Pos))) ~~>
            = Term <~~ pushg([pop()]) ~~> { Term <~~ pushg(append(popg(), pop())) ~~> } ;

Term        = TaggedTerm | Command ;

TaggedTerm <~~ push(popg()) ~~>
            = ( Name | ByteRange | Range | CharsOf | CharOf | Group | Option | Repetition | Times ) <~~ pushg(simplify(pop())) ~~> [ Tag <~~ var tag=pop(); tag.Childs=simplifyToArr(popg()); pushg(tag) ~~> ] ;

Range       <~~ push(popg()) ~~>
            = Token <~~ pushg(pop()) ~~> [ "..." Token <~~ pushg(abnf.newRange([popg(), pop()], abnf.rangeType.Rune, c.Pos)) ~~> ] ;
ByteRange   = Token <~~ pushg(pop()) ~~> "..b" Token <~~ push(abnf.newRange([popg(), pop()], abnf.rangeType.Byte, c.Pos)) ~~> ;
CharsOf     = "@" Token <~~ push(abnf.newCharsOf(pop().String, c.Pos)) ~~> "+" ;
CharOf      = "@" Token <~~ push(abnf.newCharOf(pop().String, c.Pos)) ~~> ;
Group       = "(" Expression <~~ push(abnf.newGroup(simplifyToArr(pop()), c.Pos)) ~~> ")" ;
Option      = "[" Expression <~~ push(abnf.newOption(simplifyToArr(pop()), c.Pos)) ~~> "]" ;
Repetition  = "{" Expression <~~ push(abnf.newRepetition(simplifyToArr(pop()), c.Pos)) ~~> "}" ;
Times       = CmdNumber <~~ pushg([pop()]) ~~> [ "..." ( CmdNumber | "" <~~ push(abnf.newToken("...")) ~~> ) <~~ pushg(append(popg(), pop())) ~~> ] Group <~~ push(abnf.newTimes(popg(), simplifyToArr(pop()), c.Pos)) ~~> ;

Tag         <~~ push(abnf.newTag(popg(), undefined, c.Pos)) ~~>
            = "<" ( Name | Token ) <~~ pushg([pop()]) ~~> { "," ( Name | Token ) <~~ pushg(append(popg(), pop())) ~~> } ">" ;

CmdNumber   = Number | Command ;

LineCommand = Command <~~ pushg(buildLineCommand(pop())) ~~> ";" ;

Command     <~~ push(abnf.newCommand(pop(), popg(), c.Pos)) ~~>
            = ":" CmdName "(" <~~ pushg([]) ~~> [ ( Name | Token | Number ) <~~ pushg(append(popg(), pop())) ~~> { "," ( Name | Token | Number ) <~~ pushg(append(popg(), pop())) ~~> } ] ")" ;
CmdName     <~~ push(up.in) ~~>
            = Alphabet :skip() { Alphabet | Digit | "_" } :skip(Whitespace) ;

Name        <~~ push(abnf.newIdentifier(up.in, getNameIdx(up.in), c.Pos)) ~~>
            = Alphabet :skip() { Alphabet | Digit | "_" } :skip(Whitespace) ;

Token       = Dquotetoken | Squotetoken | Code ;
Dquotetoken = '"' :skip() { AsciiNoQs | "'" | '\\"' } <~~ push(abnf.newToken(unescape(up.in), c.Pos)) ~~> '"' :skip(Whitespace) ;
Squotetoken = "'" :skip() { AsciiNoQs | '"' | "\\'" } <~~ push(abnf.newToken(unescape(up.in), c.Pos)) ~~> "'" :skip(Whitespace) ;
Code        = '~~' :skip() { [ "~" ] AllButTilde } <~~ push(abnf.newToken(unescapeTilde(up.in), c.Pos)) ~~> '~~' :skip(Whitespace) ;

Alphabet    = "a"..."z" | "A"..."Z" ;
Digit       = "0"..."9" ;
AsciiNoQs   = "\x28"..."\x7e" | "\x23"..."\x26" | @"\t\n\r !" ; // Readable ASCII without double and single quotes.
AsciiNoLb   = " "..."~" | "\t" ; // Readable ASCII without line breaks (CR and LF).
AsciiNoStSl = "\x00"...")" | "+"..."." | "0"..."~" ; // All ASCII without star (*) and slash (/).
AllButTilde = "\x00"..."}" | "\\~" | "\x7f"..."\uffff" ; // All ASCII and unicode chars. Only tilde is escaped.

Number      <~~ push(abnf.newNumber(up.in, c.Pos)) ~~>
            = "0" | "1"..."9" { "0"..."9" } ;

Whitespace  = { @"\t\n\r "+ | Comment } ;

Comment     = LineComment | "/*" :skip() { { "*" } AsciiNoStSl { "/" } } "*/" :skip(Whitespace) ;
LineComment = "//" :skip() { AsciiNoLb } ( "\n" | "\r" ) :skip(Whitespace) ;

// ---


:startScript(~~

    let names = []
    let prodsPos = {}
    let lastPos = 0

    function getNameIdx(name) {
        const pos = names.indexOf(name)
        if (pos != -1) return pos
        return names.push(name) - 1
    }
    function resolveNameIdx(productions) {
        for (let i = 0; i < productions.length; i++) {
            let rule = productions[i]
            if (rule.Childs != undefined && rule.Childs.length > 0) resolveNameIdx(rule.Childs)
            if (rule.CodeChilds != undefined && rule.CodeChilds.length > 0) resolveNameIdx(rule.CodeChilds)
            if (rule.Operator == abnf.oid.Production || rule.Operator == abnf.oid.Identifier) rule.Int = prodsPos[rule.Int]
        }
    }
    function buildProduction(prodName, prodTag, prodExpression) {
        if (prodsPos[prodName.Int] != undefined) {
            println("Error: Rule " + prodName.String + " is defined multiple times.")
            exit(0)
        }
        prodsPos[prodName.Int] = lastPos++
        if (prodTag != undefined) {
            prodTag.Childs = simplifyToArr(prodExpression)
            return abnf.newProduction(prodName.String, prodName.Int, [prodTag], prodName.Pos)
        } else {
            return abnf.newProduction(prodName.String, prodName.Int, simplifyToArr(prodExpression), prodName.Pos)
        }
    }
    function buildLineCommand(cmd) {
        cmd.Int = getNameIdx(cmd.String)
        prodsPos[cmd.Int] = lastPos++
        return cmd
    }

    // This breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.
    function simplifyArr(rules) {
        if (rules.length == 1) {
            const op = rules[0].Operator
            if (op == abnf.oid.Sequence || op == abnf.oid.Group || (op == abnf.oid.Or && rules[0].Childs.length <= 1)) return simplifyArr(rules[0].Childs)
        }
        return rules
    }

    // This also breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.
    function simplifyToArr(rule) {
        if (rule == undefined) return undefined
        return simplifyArr([rule])
    }

    // Groups with only one child can be broken apart as long as down there is an unbreakable rule. Try to find one.
    function trySimplifyDown(rule) {
        if (rule.Childs == undefined) return rule
        const op = rule.Operator
        if ((rule.Childs.length == 1) && (op == abnf.oid.Sequence || op == abnf.oid.Group || op == abnf.oid.Or)) return trySimplifyDown(rule.Childs[0])
        if (op == abnf.oid.Sequence) return undefined
        return rule
    }

    function simplify(rule) {
        let ruleDown = trySimplifyDown(rule)
        if (ruleDown != undefined) return ruleDown
        if (rule.Childs.length == 1) { // Breaking up abnf.oid.Group did not work. Getting down only with Sequence and Or.
            const op = rule.Operator
            if (op == abnf.oid.Sequence || op == abnf.oid.Or) return simplify(rule.Childs[0])
        }
        return rule
    }

    c.compile(c.asg)
    let rules = ltr.stack
    resolveNameIdx(rules)

    // To show the initial a-grammar:
    println("=> Rules: " + abnf.serializeRules(rules))

    // To return the generated a-grammar to the next parser:
    rules

~~) ;
```
</details>

### Its output, when it gets applied on itself:

<details>
  <summary>Click to expand!</summary>

```javascript
&r.Rules{&r.Rule{ Operator: r.Command, String: "title", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "ABNF of ABNF to a-grammar"
            }
        }
    }, &r.Rule{ Operator: r.Command, String: "description", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "This ABNF contains the grammatic and semantic information for annotated EBNF.\nIt allows to automatically create a compiler for everything described in ABNF (yes, that format)."
            }
        }
    }, &r.Rule{ Operator: r.Command, String: "startRule", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "ABNF", Int: 4
            }
        }
    }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "ABNF", Childs:&r.Rules{&r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Production", Int: 5
                            }, &r.Rule{ Operator: r.Identifier, String: "LineCommand", Int: 21
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Production", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " var prodTag=undefined; var prodExpression=undefined; pushg(pop()) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Name", Int: 24
                    }
                }
            }, &r.Rule{ Operator: r.Optional, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " prodTag=pop() "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Tag", Int: 19
                            }
                        }
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "="
            }, &r.Rule{ Operator: r.Optional, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " prodExpression=pop() "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Expression", Int: 6
                            }
                        }
                    }
                }
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "  pushg(buildProduction(popg(), prodTag, prodExpression)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: ";"
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Expression", Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Alternative", Int: 7
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Alternative", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(simplify(abnf.newAlternative(popg(), c.Pos))) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg([pop()]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Sequence", Int: 8
                            }
                        }
                    }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "|"
                            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Sequence", Int: 8
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Sequence", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(simplify(abnf.newSequence(popg(), c.Pos))) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg([pop()]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Term", Int: 9
                            }
                        }
                    }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Term", Int: 9
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Term", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "TaggedTerm", Int: 10
                    }, &r.Rule{ Operator: r.Identifier, String: "Command", Int: 22
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "TaggedTerm", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(popg()) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(simplify(pop())) "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Name", Int: 24
                                    }, &r.Rule{ Operator: r.Identifier, String: "ByteRange", Int: 12
                                    }, &r.Rule{ Operator: r.Identifier, String: "Range", Int: 11
                                    }, &r.Rule{ Operator: r.Identifier, String: "CharsOf", Int: 13
                                    }, &r.Rule{ Operator: r.Identifier, String: "CharOf", Int: 14
                                    }, &r.Rule{ Operator: r.Identifier, String: "Group", Int: 15
                                    }, &r.Rule{ Operator: r.Identifier, String: "Option", Int: 16
                                    }, &r.Rule{ Operator: r.Identifier, String: "Repetition", Int: 17
                                    }, &r.Rule{ Operator: r.Identifier, String: "Times", Int: 18
                                    }
                                }
                            }
                        }
                    }, &r.Rule{ Operator: r.Optional, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " var tag=pop(); tag.Childs=simplifyToArr(popg()); pushg(tag) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Tag", Int: 19
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Range", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(popg()) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(pop()) "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                            }
                        }
                    }, &r.Rule{ Operator: r.Optional, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "..."
                            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(abnf.newRange([popg(), pop()], abnf.rangeType.Rune, c.Pos)) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "ByteRange", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(pop()) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "..b"
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newRange([popg(), pop()], abnf.rangeType.Byte, c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "CharsOf", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "@"
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newCharsOf(pop().String, c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "+"
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "CharOf", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "@"
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newCharOf(pop().String, c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Group", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "("
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newGroup(simplifyToArr(pop()), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Expression", Int: 6
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: ")"
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Option", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "["
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newOption(simplifyToArr(pop()), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Expression", Int: 6
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "]"
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Repetition", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "{"
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newRepetition(simplifyToArr(pop()), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Expression", Int: 6
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "}"
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Times", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg([pop()]) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "CmdNumber", Int: 20
                    }
                }
            }, &r.Rule{ Operator: r.Optional, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "..."
                    }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(append(popg(), pop())) "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "CmdNumber", Int: 20
                                    }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newToken(\"...\")) "
                                            }
                                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: ""
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newTimes(popg(), simplifyToArr(pop()), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Group", Int: 15
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Tag", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newTag(popg(), undefined, c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "<"
                    }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg([pop()]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Name", Int: 24
                                    }, &r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                                    }
                                }
                            }
                        }
                    }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: ","
                            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Name", Int: 24
                                            }, &r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }, &r.Rule{ Operator: r.Token, String: ">"
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "CmdNumber", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Number", Int: 35
                    }, &r.Rule{ Operator: r.Identifier, String: "Command", Int: 22
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "LineCommand", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(buildLineCommand(pop())) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Command", Int: 22
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: ";"
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Command", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newCommand(pop(), popg(), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: ":"
                    }, &r.Rule{ Operator: r.Identifier, String: "CmdName", Int: 23
                    }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg([]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "("
                            }
                        }
                    }, &r.Rule{ Operator: r.Optional, Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Name", Int: 24
                                            }, &r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                                            }, &r.Rule{ Operator: r.Identifier, String: "Number", Int: 35
                                            }
                                        }
                                    }
                                }
                            }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: ","
                                    }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " pushg(append(popg(), pop())) "
                                            }
                                        }, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Name", Int: 24
                                                    }, &r.Rule{ Operator: r.Identifier, String: "Token", Int: 25
                                                    }, &r.Rule{ Operator: r.Identifier, String: "Number", Int: 35
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }, &r.Rule{ Operator: r.Token, String: ")"
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "CmdName", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(up.in) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Alphabet", Int: 29
                    }, &r.Rule{ Operator: r.Command, String: "skip"
                    }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Alphabet", Int: 29
                                    }, &r.Rule{ Operator: r.Identifier, String: "Digit", Int: 30
                                    }, &r.Rule{ Operator: r.Token, String: "_"
                                    }
                                }
                            }
                        }
                    }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Name", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newIdentifier(up.in, getNameIdx(up.in), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Alphabet", Int: 29
                    }, &r.Rule{ Operator: r.Command, String: "skip"
                    }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Alphabet", Int: 29
                                    }, &r.Rule{ Operator: r.Identifier, String: "Digit", Int: 30
                                    }, &r.Rule{ Operator: r.Token, String: "_"
                                    }
                                }
                            }
                        }
                    }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Token", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Dquotetoken", Int: 26
                    }, &r.Rule{ Operator: r.Identifier, String: "Squotetoken", Int: 27
                    }, &r.Rule{ Operator: r.Identifier, String: "Code", Int: 28
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Dquotetoken", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "\""
            }, &r.Rule{ Operator: r.Command, String: "skip"
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newToken(unescape(up.in), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "AsciiNoQs", Int: 31
                                    }, &r.Rule{ Operator: r.Token, String: "'"
                                    }, &r.Rule{ Operator: r.Token, String: "\\\""
                                    }
                                }
                            }
                        }
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "\""
            }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Squotetoken", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "'"
            }, &r.Rule{ Operator: r.Command, String: "skip"
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newToken(unescape(up.in), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "AsciiNoQs", Int: 31
                                    }, &r.Rule{ Operator: r.Token, String: "\""
                                    }, &r.Rule{ Operator: r.Token, String: "\\'"
                                    }
                                }
                            }
                        }
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "'"
            }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Code", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "~~"
            }, &r.Rule{ Operator: r.Command, String: "skip"
            }, &r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newToken(unescapeTilde(up.in), c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Optional, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "~"
                                    }
                                }
                            }, &r.Rule{ Operator: r.Identifier, String: "AllButTilde", Int: 34
                            }
                        }
                    }
                }
            }, &r.Rule{ Operator: r.Token, String: "~~"
            }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Alphabet", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "a"
                            }, &r.Rule{ Operator: r.Token, String: "z"
                            }
                        }
                    }, &r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "A"
                            }, &r.Rule{ Operator: r.Token, String: "Z"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Digit", Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "0"
                    }, &r.Rule{ Operator: r.Token, String: "9"
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "AsciiNoQs", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "("
                            }, &r.Rule{ Operator: r.Token, String: "~"
                            }
                        }
                    }, &r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "#"
                            }, &r.Rule{ Operator: r.Token, String: "&"
                            }
                        }
                    }, &r.Rule{ Operator: r.CharOf, String: "\t\n\r !"
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "AsciiNoLb", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " "
                            }, &r.Rule{ Operator: r.Token, String: "~"
                            }
                        }
                    }, &r.Rule{ Operator: r.Token, String: "\t"
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "AsciiNoStSl", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "\x00"
                            }, &r.Rule{ Operator: r.Token, String: ")"
                            }
                        }
                    }, &r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "+"
                            }, &r.Rule{ Operator: r.Token, String: "."
                            }
                        }
                    }, &r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "0"
                            }, &r.Rule{ Operator: r.Token, String: "~"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "AllButTilde", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "\x00"
                            }, &r.Rule{ Operator: r.Token, String: "}"
                            }
                        }
                    }, &r.Rule{ Operator: r.Token, String: "\\~"
                    }, &r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "\u007f"
                            }, &r.Rule{ Operator: r.Token, String: "\uffff"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Number", Childs:&r.Rules{&r.Rule{ Operator: r.Tag, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: " push(abnf.newNumber(up.in, c.Pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "0"
                            }, &r.Rule{ Operator: r.Sequence, Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "1"
                                            }, &r.Rule{ Operator: r.Token, String: "9"
                                            }
                                        }
                                    }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Range, Int: 0, CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "0"
                                                    }, &r.Rule{ Operator: r.Token, String: "9"
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Whitespace", Childs:&r.Rules{&r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.CharsOf, String: "\t\n\r "
                            }, &r.Rule{ Operator: r.Identifier, String: "Comment", Int: 37
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "Comment", Childs:&r.Rules{&r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "LineComment", Int: 38
                    }, &r.Rule{ Operator: r.Sequence, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "/*"
                            }, &r.Rule{ Operator: r.Command, String: "skip"
                            }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "*"
                                            }
                                        }
                                    }, &r.Rule{ Operator: r.Identifier, String: "AsciiNoStSl", Int: 33
                                    }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "/"
                                            }
                                        }
                                    }
                                }
                            }, &r.Rule{ Operator: r.Token, String: "*/"
                            }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Production, String: "LineComment", Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "//"
            }, &r.Rule{ Operator: r.Command, String: "skip"
            }, &r.Rule{ Operator: r.Repeat, Childs:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "AsciiNoLb", Int: 32
                    }
                }
            }, &r.Rule{ Operator: r.Or, Childs:&r.Rules{&r.Rule{ Operator: r.Token, String: "\n"
                    }, &r.Rule{ Operator: r.Token, String: "\r"
                    }
                }
            }, &r.Rule{ Operator: r.Command, String: "skip", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Identifier, String: "Whitespace", Int: 36
                    }
                }
            }
        }
    }, &r.Rule{ Operator: r.Command, String: "startScript", CodeChilds:&r.Rules{&r.Rule{ Operator: r.Token, String: "\n\n    let names = []\n    let prodsPos = {}\n    let lastPos = 0\n\n    function getNameIdx(name) {\n        const pos = names.indexOf(name)\n        if (pos != -1) return pos\n        return names.push(name) - 1\n    }\n    function resolveNameIdx(productions) {\n        for (let i = 0; i < productions.length; i++) {\n            let rule = productions[i]\n            if (rule.Childs != undefined && rule.Childs.length > 0) resolveNameIdx(rule.Childs)\n            if (rule.CodeChilds != undefined && rule.CodeChilds.length > 0) resolveNameIdx(rule.CodeChilds)\n            if (rule.Operator == abnf.oid.Production || rule.Operator == abnf.oid.Identifier) rule.Int = prodsPos[rule.Int]\n        }\n    }\n    function buildProduction(prodName, prodTag, prodExpression) {\n        if (prodsPos[prodName.Int] != undefined) {\n            println(\"Error: Rule \" + prodName.String + \" is defined multiple times.\")\n            exit(0)\n        }\n        prodsPos[prodName.Int] = lastPos++\n        if (prodTag != undefined) {\n            prodTag.Childs = simplifyToArr(prodExpression)\n            return abnf.newProduction(prodName.String, prodName.Int, [prodTag], prodName.Pos)\n        } else {\n            return abnf.newProduction(prodName.String, prodName.Int, simplifyToArr(prodExpression), prodName.Pos)\n        }\n    }\n    function buildLineCommand(cmd) {\n        cmd.Int = getNameIdx(cmd.String)\n        prodsPos[cmd.Int] = lastPos++\n        return cmd\n    }\n\n    // This breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.\n    function simplifyArr(rules) {\n        if (rules.length == 1) {\n            const op = rules[0].Operator\n            if (op == abnf.oid.Sequence || op == abnf.oid.Group || (op == abnf.oid.Or && rules[0].Childs.length <= 1)) return simplifyArr(rules[0].Childs)\n        }\n        return rules\n    }\n\n    // This also breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.\n    function simplifyToArr(rule) {\n        if (rule == undefined) return undefined\n        return simplifyArr([rule])\n    }\n\n    // Groups with only one child can be broken apart as long as down there is an unbreakable rule. Try to find one.\n    function trySimplifyDown(rule) {\n        if (rule.Childs == undefined) return rule\n        const op = rule.Operator\n        if ((rule.Childs.length == 1) && (op == abnf.oid.Sequence || op == abnf.oid.Group || op == abnf.oid.Or)) return trySimplifyDown(rule.Childs[0])\n        if (op == abnf.oid.Sequence) return undefined\n        return rule\n    }\n\n    function simplify(rule) {\n        let ruleDown = trySimplifyDown(rule)\n        if (ruleDown != undefined) return ruleDown\n        if (rule.Childs.length == 1) { // Breaking up abnf.oid.Group did not work. Getting down only with Sequence and Or.\n            const op = rule.Operator\n            if (op == abnf.oid.Sequence || op == abnf.oid.Or) return simplify(rule.Childs[0])\n        }\n        return rule\n    }\n\n    c.compile(c.asg)\n    let rules = ltr.stack\n    resolveNameIdx(rules)\n\n    // To show the initial a-grammar:\n    println(\"=> Rules: \" + abnf.serializeRules(rules))\n\n    // To return the generated a-grammar to the next parser:\n    rules\n\n"
            }
        }
    }
}
```
</details>

<br/>

More examples can be found inside the [tests directory](../master/tests).

## Links

Grammars for many languages: [Grammars written for ANTLR v4](https://github.com/antlr/grammars-v4)