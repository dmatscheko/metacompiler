# ParserParserCompiler

This is
* a parser that parses an annotated EBNF
* and (at runtime) creates a second parser with the EBNF
* plus a compiler with the annotations in the EBNF.

That new runtime generated parser plus compiler should allow to compile an arbitrary new language, specified in the EBNF and its annotations.

## What is an annotated EBNF (a-EBNF)?

* The EBNF stores the grammar of e.g. a new language.
* The annotations in the EBNF store the semantic of the new language.
* The combined format is annotated EBNF or a-EBNF.

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

### Installation / Build

```
go get "github.com/dop251/goja" "github.com/llir/llvm/ir"
go run . -f tests/llvm-ir-tests.aebnf -s tests/tiny.aebnf
```

### Default main process

This is the input to the default main process:
* `initial-a-grammar` = program-internal annotated grammar of annotated EBNF.
* `inputA` = the content of the file given with command line parameter `-a`.
* `inputB` = the content of the file given with command line parameter `-b`.

This is how that input is processed:
1. `parse(initial-a-grammar, inputA)`  = `inputA-ASG`.
2. `compile(inputA-ASG)`  = `new-a-grammar`.
3. `parse(new-a-grammar, inputB)`  = `inputB-ASG`.
4. `compile(inputB-ASG)`  = `result`.

Of course, the `result` can again (but doesn't have to) be an `a-grammar` and can be used as input for `parse()`.

### Exposed JS API

The annotations of the a-EBNF can contain JS code. The ASG (abstract semantic graph) gets processed from the leaves up to the stem. If annotations are encountered on the way, their JS code gets executed.

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

* __c.asg__
The whole abstract semantic graph.
* __c.localAsg__
The local part of the abstract semantic graph.
* __c.compile(asg []Rule) map[string]object__  
  Compiles the given ASG and returns the map of the combined upstream variables.  
Normally, `c.compile()` is called as `c.compile(c.asg);`.
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

#### LLVM IR API

This tool uses the [Go LLIR/LLVM library](https://github.com/llir/llvm) to create LLVM IR and to interact with it. The API documentation can be found here: [LLIR/LLVM library documentation](https://pkg.go.dev/github.com/llir/llvm/). For more information on LLVM IR go to the [LLVM language reference](https://llvm.org/docs/LangRef.html).

The functions and constants are exposed to JS as:

* __llvm.ir.\*__ [llvm.ir](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir)
* __llvm.constant.\*__ [llvm.constant](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/constant)
* __llvm.metadata.\*__ [llvm.metadata](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/metadata)
* __llvm.types.\*__ [llvm.types](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/types)

* Custom functions:
  * __llvm.Callgraph(m ir.Module) string__  
  The function `llvm.Callgraph(m ir.Module) string` creates the callgraph of the given LLVM IR module in Graphviz DOT format (can be viewed e.g. online with the [Graphviz visual editor](http://magjac.com/graphviz-visual-editor/)).
  * __llvm.Callgraph(m ir.Module, f string)__  
  The function `llvm.Callgraph(m ir.Module, f string)` tries to execute the function `f` inside the IR module `m` and returns the resulting uint32.

#### Compiler EBNF a-grammar API

The a-grammar can be built from within JS. For this, some simple builder funcions are exposed:

##### Builder functions

* __ebnf.arrayToRules(rules []object) []Rule__
* __ebnf.newRule(Operator OperatorID, String string, Int int, Bool bool, Rune rune, Pos int, Childs []Rule, TagChilds []Rule) Rule__
* __ebnf.newToken(String string, Pos int) Rule__
* __ebnf.newName(String string, Int int, Pos int) Rule__
* __ebnf.newProduction(String string, Int int, Childs []Rule, Pos int) Rule__
* __ebnf.newTag(TagChilds []Rule, Childs []Rule, Pos int) Rule__
* __ebnf.newSkipSpace(Bool bool, Pos int) Rule__
* __ebnf.newRepetition(Childs []Rule, Pos int) Rule__
* __ebnf.newOption(Childs []Rule, Pos int) Rule__
* __ebnf.newGroup(Childs []Rule, Pos int) Rule__
* __ebnf.newSequence(Childs []Rule, Pos int) Rule__
* __ebnf.newAlternative(Childs []Rule, Pos int) Rule__
* __ebnf.newRange(Childs []Rule, Pos int) Rule__

##### Text functions

* __ebnf.serializeRule(r Rule)__
* __ebnf.serializeRules(rs []Rule)__

##### OperatorID Constants

* __ebnf.oid.Error__
* __ebnf.oid.Success__
* __ebnf.oid.Sequence__
* __ebnf.oid.Group__
* __ebnf.oid.Token__
* __ebnf.oid.Or__
* __ebnf.oid.Optional__
* __ebnf.oid.Repeat__
* __ebnf.oid.Range__
* __ebnf.oid.SkipSpace__
* __ebnf.oid.Tag__
* __ebnf.oid.Production__
* __ebnf.oid.Ident__

### a-EBNF Syntax

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

#### EBNF of a-EBNF

Annotated EBNF basically only adds tags to the syntax of a normal EBNF:

```javascript
"EBNF of a-EBNF" {

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

This is the definition of `name` and `token`, of `skipspaces`, and of `code`. Except of `code`, they are common to EBNF and a-EBNF:

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
Skip        = "+" ;  // Skip all whitespace in the future.
Noskip      = "-" ;  // Do not skip whitspace in the future.

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
"a-EBNF of a-EBNF: Compiles to textual a-grammar"
<~~
var names = [];
function getNameIdx(name) {
    var pos = names.indexOf(name);
    if (pos != -1) { return pos };
    return names.push(name)-1;
}
c.compile(c.asg, up.in)
~~>

{

Program                                 <~~push(up.in)~~>
                =
                [
                    Title               <~~ up.in += ', \\n' ~~>
                ]
                Programtag
                "{"                     <~~ up.in = '\\n{\\n\\n' ~~>
                { Production }
                "}"                     <~~ up.in = '\\n}, \\n\\n' ~~>
                Programtag
                start
                [
                    Comment             <~~ up.in = ', \\n'+up.in+'\\n' ~~>
                ] ;

Programtag      =
                [
                    Tag                 <~~ up.in = '{"TAG", '+up.in+'}, \\n' ~~>
                ] ;

Production                              <~~ var productionTag = pop();
                                        if (productionTag != undefined) {
                                            pop()/*=undefined*/;
                                            up.in = '{"TAG", '+productionTag+', '+up.in+'}'
                                        };
                                        up.in += ',\\n' ~~>       
                =
                Name                    <~~ up.in = '{"'+up.in+'", '+getNameIdx(up.in)+', ';
                                        push(undefined) ~~>
                [
                    Tag                 <~~ push(up.in); up.in = '' ~~>
                ]
                "="                     <~~up.in=''~~>
                [ Expression ]
                ( "." | ";" )           <~~ up.in = '}' ~~> ;

Expression                              <~~ if (/*or*/ pop()) {
                                            up.in = '{"OR", '+up.in+'}'
                                        } ~~>
                =
                Alternative             <~~ /*or=false*/ push(false); ~~>
                {
                    "|"                 <~~up.in=''~~>
                    Alternative         <~~ /*or=true*/ pop();push(true); up.in = ', '+up.in ~~>
                } ;

Alternative     =
                Taggedterm
                {
                    Taggedterm          <~~ up.in = ', '+up.in ~~>
                } ;

Taggedterm                              <~~up.in=pop()~~>
                =
                Term                    <~~push(up.in)~~>
                [
                    Tag                 <~~ push('{"TAG", '+up.in+', '+pop()+'}') ~~>
                ] ;

Term            =
                (
                    Name                <~~ up.in = '{"IDENT", "'+up.in+'", '+getNameIdx(up.in)+'}' ~~>
                    |
                    ( Token [ "..." Token ] )
                    |
                    group
                    |
                    option
                    |
                    repetition
                    |
                    skipspaces
                ) ;

group           <~~ up.in = '{'+pop()+'}' ~~>                    = "(" Expression <~~push(up.in)~~> ")" ;
option          <~~ up.in = '{"OPTIONAL", '+pop()+'}' ~~>        = "[" Expression <~~push(up.in)~~> "]" ;
repetition      <~~ up.in = '{"REPEAT", '+pop()+'}' ~~>          = "{" Expression <~~push(up.in)~~> "}" ;
skipspaces      =
                "+"                     <~~ up.in = '{"SKIPSPACES", true}' ~~>
                |
                "-"                     <~~ up.in = '{"SKIPSPACES", false}' ~~> ;

Title           = Token ;
start           = Name                  <~~ up.in = '{"IDENT", "'+up.in+'", '+getNameIdx(up.in)+'}' ~~> ;
Comment         = Token ;

Tag                                     <~~up.in=pop()~~>
                =
                "<"
                Code                    <~~push(up.in)~~>
                {
                    ","
                    Code                <~~ push(pop()+', '+up.in) ~~>
                }
                ">" ;

Code                                    <~~ up.in = '{"TERMINAL", '+sprintf("%q",pop())+'}' ~~>
                =
                '~~'
                -
                { [ "~" ] Codeinner }   <~~push(up.in)~~>
                '~~'
                + ;

Codeinner       = Small | Caps | Digit | Special | "'" | '"' | "\\~" ;

Name            = ( Small | Caps ) - { Small | Caps | Digit | "_" } + ;

Token                                   <~~ up.in = '{"TERMINAL", '+sprintf("%q",pop())+'}' ~~>
                = Dqtoken | Sqtoken ;
Dqtoken         = '"' - { Small | Caps | Digit | Special | "~" | "'" | '\\"' } <~~push(up.in)~~> '"' + ;
Sqtoken         = "'" - { Small | Caps | Digit | Special | "~" | '"' | "\\'" } <~~push(up.in)~~> "'" + ;

Digit           = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
Small           = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" |
                  "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
Caps            = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" |
                  "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
Special         = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" |
                  "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "|" | "%" | "$" | "&" | "#" |
                  "@" | "\\\\" | "\\t" | "\t" | "\\n" | "\n" | "\\r" | "\r" ;

}

<~~ print(pop()) ~~>
Program
"This a-EBNF contains the grammatic and semantic information for annotated EBNF.
It allows to automatically create a compiler for everything described in a-EBNF (yes, that format)."
```
</details>

### Its output, when it gets applied on itself:

<details>
  <summary>Click to expand!</summary>

```javascript
{"TERMINAL", "a-EBNF of a-EBNF: Compiles to textual a-grammar"}, 
{"TAG", {"TERMINAL", "\nvar names = [];\nfunction getNameIdx(name) {\n    var pos = names.indexOf(name);\n    if (pos != -1) { return pos };\n    return names.push(name)-1;\n}\nc.compile(c.asg, up.in)\n"}}, 

{

{"TAG", {"TERMINAL", "push(up.in)"}, {"Program", 0, {"OPTIONAL", {"TAG", {"TERMINAL", " up.in += ', \\\\n' "}, {"IDENT", "Title", 1}}}, {"IDENT", "Programtag", 2}, {"TAG", {"TERMINAL", " up.in = '\\\\n{\\\\n\\\\n' "}, {"TERMINAL", "{"}}, {"REPEAT", {"IDENT", "Production", 3}}, {"TAG", {"TERMINAL", " up.in = '\\\\n}, \\\\n\\\\n' "}, {"TERMINAL", "}"}}, {"IDENT", "Programtag", 2}, {"IDENT", "start", 4}, {"OPTIONAL", {"TAG", {"TERMINAL", " up.in = ', \\\\n'+up.in+'\\\\n' "}, {"IDENT", "Comment", 5}}}}},
{"Programtag", 2, {"OPTIONAL", {"TAG", {"TERMINAL", " up.in = '{\"TAG\", '+up.in+'}, \\\\n' "}, {"IDENT", "Tag", 6}}}},
{"TAG", {"TERMINAL", " var productionTag = pop();\n                                        if (productionTag != undefined) {\n                                            pop()/*=undefined*/;\n                                            up.in = '{\"TAG\", '+productionTag+', '+up.in+'}'\n                                        };\n                                        up.in += ',\\\\n' "}, {"Production", 3, {"TAG", {"TERMINAL", " up.in = '{\"'+up.in+'\", '+getNameIdx(up.in)+', ';\n                                        push(undefined) "}, {"IDENT", "Name", 7}}, {"OPTIONAL", {"TAG", {"TERMINAL", " push(up.in); up.in = '' "}, {"IDENT", "Tag", 6}}}, {"TAG", {"TERMINAL", "up.in=''"}, {"TERMINAL", "="}}, {"OPTIONAL", {"IDENT", "Expression", 8}}, {"TAG", {"TERMINAL", " up.in = '}' "}, {{"OR", {"TERMINAL", "."}, {"TERMINAL", ";"}}}}}},
{"TAG", {"TERMINAL", " if (/*or*/ pop()) {\n                                            up.in = '{\"OR\", '+up.in+'}'\n                                        } "}, {"Expression", 8, {"TAG", {"TERMINAL", " /*or=false*/ push(false); "}, {"IDENT", "Alternative", 9}}, {"REPEAT", {"TAG", {"TERMINAL", "up.in=''"}, {"TERMINAL", "|"}}, {"TAG", {"TERMINAL", " /*or=true*/ pop();push(true); up.in = ', '+up.in "}, {"IDENT", "Alternative", 9}}}}},
{"Alternative", 9, {"IDENT", "Taggedterm", 10}, {"REPEAT", {"TAG", {"TERMINAL", " up.in = ', '+up.in "}, {"IDENT", "Taggedterm", 10}}}},
{"TAG", {"TERMINAL", "up.in=pop()"}, {"Taggedterm", 10, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "Term", 11}}, {"OPTIONAL", {"TAG", {"TERMINAL", " push('{\"TAG\", '+up.in+', '+pop()+'}') "}, {"IDENT", "Tag", 6}}}}},
{"Term", 11, {{"OR", {"TAG", {"TERMINAL", " up.in = '{\"IDENT\", \"'+up.in+'\", '+getNameIdx(up.in)+'}' "}, {"IDENT", "Name", 7}}, {{"IDENT", "Token", 12}, {"OPTIONAL", {"TERMINAL", "..."}, {"IDENT", "Token", 12}}}, {"IDENT", "group", 13}, {"IDENT", "option", 14}, {"IDENT", "repetition", 15}, {"IDENT", "skipspaces", 16}}}},
{"TAG", {"TERMINAL", " up.in = '{'+pop()+'}' "}, {"group", 13, {"TERMINAL", "("}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "Expression", 8}}, {"TERMINAL", ")"}}},
{"TAG", {"TERMINAL", " up.in = '{\"OPTIONAL\", '+pop()+'}' "}, {"option", 14, {"TERMINAL", "["}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "Expression", 8}}, {"TERMINAL", "]"}}},
{"TAG", {"TERMINAL", " up.in = '{\"REPEAT\", '+pop()+'}' "}, {"repetition", 15, {"TERMINAL", "{"}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "Expression", 8}}, {"TERMINAL", "}"}}},
{"skipspaces", 16, {"OR", {"TAG", {"TERMINAL", " up.in = '{\"SKIPSPACES\", true}' "}, {"TERMINAL", "+"}}, {"TAG", {"TERMINAL", " up.in = '{\"SKIPSPACES\", false}' "}, {"TERMINAL", "-"}}}},
{"Title", 1, {"IDENT", "Token", 12}},
{"start", 4, {"TAG", {"TERMINAL", " up.in = '{\"IDENT\", \"'+up.in+'\", '+getNameIdx(up.in)+'}' "}, {"IDENT", "Name", 7}}},
{"Comment", 5, {"IDENT", "Token", 12}},
{"TAG", {"TERMINAL", "up.in=pop()"}, {"Tag", 6, {"TERMINAL", "<"}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "Code", 17}}, {"REPEAT", {"TERMINAL", ","}, {"TAG", {"TERMINAL", " push(pop()+', '+up.in) "}, {"IDENT", "Code", 17}}}, {"TERMINAL", ">"}}},
{"TAG", {"TERMINAL", " up.in = '{\"TERMINAL\", '+sprintf(\"%q\",pop())+'}' "}, {"Code", 17, {"TERMINAL", "~~"}, {"SKIPSPACES", false}, {"TAG", {"TERMINAL", "push(up.in)"}, {"REPEAT", {"OPTIONAL", {"TERMINAL", "~"}}, {"IDENT", "Codeinner", 18}}}, {"TERMINAL", "~~"}, {"SKIPSPACES", true}}},
{"Codeinner", 18, {"OR", {"IDENT", "Small", 19}, {"IDENT", "Caps", 20}, {"IDENT", "Digit", 21}, {"IDENT", "Special", 22}, {"TERMINAL", "'"}, {"TERMINAL", "\""}, {"TERMINAL", "\\\\~"}}},
{"Name", 7, {{"OR", {"IDENT", "Small", 19}, {"IDENT", "Caps", 20}}}, {"SKIPSPACES", false}, {"REPEAT", {"OR", {"IDENT", "Small", 19}, {"IDENT", "Caps", 20}, {"IDENT", "Digit", 21}, {"TERMINAL", "_"}}}, {"SKIPSPACES", true}},
{"TAG", {"TERMINAL", " up.in = '{\"TERMINAL\", '+sprintf(\"%q\",pop())+'}' "}, {"Token", 12, {"OR", {"IDENT", "Dqtoken", 23}, {"IDENT", "Sqtoken", 24}}}},
{"Dqtoken", 23, {"TERMINAL", "\""}, {"SKIPSPACES", false}, {"TAG", {"TERMINAL", "push(up.in)"}, {"REPEAT", {"OR", {"IDENT", "Small", 19}, {"IDENT", "Caps", 20}, {"IDENT", "Digit", 21}, {"IDENT", "Special", 22}, {"TERMINAL", "~"}, {"TERMINAL", "'"}, {"TERMINAL", "\\\\\""}}}}, {"TERMINAL", "\""}, {"SKIPSPACES", true}},
{"Sqtoken", 24, {"TERMINAL", "'"}, {"SKIPSPACES", false}, {"TAG", {"TERMINAL", "push(up.in)"}, {"REPEAT", {"OR", {"IDENT", "Small", 19}, {"IDENT", "Caps", 20}, {"IDENT", "Digit", 21}, {"IDENT", "Special", 22}, {"TERMINAL", "~"}, {"TERMINAL", "\""}, {"TERMINAL", "\\\\'"}}}}, {"TERMINAL", "'"}, {"SKIPSPACES", true}},
{"Digit", 21, {"OR", {"TERMINAL", "0"}, {"TERMINAL", "1"}, {"TERMINAL", "2"}, {"TERMINAL", "3"}, {"TERMINAL", "4"}, {"TERMINAL", "5"}, {"TERMINAL", "6"}, {"TERMINAL", "7"}, {"TERMINAL", "8"}, {"TERMINAL", "9"}}},
{"Small", 19, {"OR", {"TERMINAL", "a"}, {"TERMINAL", "b"}, {"TERMINAL", "c"}, {"TERMINAL", "d"}, {"TERMINAL", "e"}, {"TERMINAL", "f"}, {"TERMINAL", "g"}, {"TERMINAL", "h"}, {"TERMINAL", "i"}, {"TERMINAL", "j"}, {"TERMINAL", "k"}, {"TERMINAL", "l"}, {"TERMINAL", "m"}, {"TERMINAL", "n"}, {"TERMINAL", "o"}, {"TERMINAL", "p"}, {"TERMINAL", "q"}, {"TERMINAL", "r"}, {"TERMINAL", "s"}, {"TERMINAL", "t"}, {"TERMINAL", "u"}, {"TERMINAL", "v"}, {"TERMINAL", "w"}, {"TERMINAL", "x"}, {"TERMINAL", "y"}, {"TERMINAL", "z"}}},
{"Caps", 20, {"OR", {"TERMINAL", "A"}, {"TERMINAL", "B"}, {"TERMINAL", "C"}, {"TERMINAL", "D"}, {"TERMINAL", "E"}, {"TERMINAL", "F"}, {"TERMINAL", "G"}, {"TERMINAL", "H"}, {"TERMINAL", "I"}, {"TERMINAL", "J"}, {"TERMINAL", "K"}, {"TERMINAL", "L"}, {"TERMINAL", "M"}, {"TERMINAL", "N"}, {"TERMINAL", "O"}, {"TERMINAL", "P"}, {"TERMINAL", "Q"}, {"TERMINAL", "R"}, {"TERMINAL", "S"}, {"TERMINAL", "T"}, {"TERMINAL", "U"}, {"TERMINAL", "V"}, {"TERMINAL", "W"}, {"TERMINAL", "X"}, {"TERMINAL", "Y"}, {"TERMINAL", "Z"}}},
{"Special", 22, {"OR", {"TERMINAL", "_"}, {"TERMINAL", " "}, {"TERMINAL", "."}, {"TERMINAL", ","}, {"TERMINAL", ":"}, {"TERMINAL", ";"}, {"TERMINAL", "!"}, {"TERMINAL", "?"}, {"TERMINAL", "+"}, {"TERMINAL", "-"}, {"TERMINAL", "*"}, {"TERMINAL", "/"}, {"TERMINAL", "="}, {"TERMINAL", "("}, {"TERMINAL", ")"}, {"TERMINAL", "{"}, {"TERMINAL", "}"}, {"TERMINAL", "["}, {"TERMINAL", "]"}, {"TERMINAL", "<"}, {"TERMINAL", ">"}, {"TERMINAL", "|"}, {"TERMINAL", "%"}, {"TERMINAL", "$"}, {"TERMINAL", "&"}, {"TERMINAL", "#"}, {"TERMINAL", "@"}, {"TERMINAL", "\\\\\\\\"}, {"TERMINAL", "\\\\t"}, {"TERMINAL", "\\t"}, {"TERMINAL", "\\\\n"}, {"TERMINAL", "\\n"}, {"TERMINAL", "\\\\r"}, {"TERMINAL", "\\r"}}},

}, 

{"TAG", {"TERMINAL", " print(pop()) "}}, 
{"IDENT", "Program", 0}, 
{"TERMINAL", "This a-EBNF contains the grammatic and semantic information for annotated EBNF.\nIt allows to automatically create a compiler for everything described in a-EBNF (yes, that format)."}
```
</details>


## Links

Grammars for many languages: [Grammars written for ANTLR v4](https://github.com/antlr/grammars-v4)