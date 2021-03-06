package main

import (
	"fmt"
	"log"
	"time"

	"./ebnf"
)

// PUBLIC EBNF of EBNF:
// 	  Production  = name "=" [ Expression ] "." .
//    Expression  = Alternative { "|" Alternative } .
//    Alternative = Term { Term } .
//    Term        = name | token [ "…" token ] | Group | Option | Repetition .
//    Group       = "(" Expression ")" .
//    Option      = "[" Expression "]" .
//    Repetition  = "{" Expression "}" .

var (
	ebnfs = []string{

		// ==========================================

		// EBNF of aEBNF with tags (ORIGINAL):
		// `"EBNF of aEBNF" {
		// program = [ title ] [ tag ] "{" { production } "}" [ tag ] [ comment ] ;
		// production  = name [ tag ] "=" [ expression ] ( "." | ";" ) ;
		// expression  = sequence ;
		// sequence    = alternative { alternative } ;
		// alternative = term { "|" term } ;
		// term        = ( name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag ] ;
		// group       = "(" expression  ")" ;
		// option      = "[" expression "]" ;
		// repetition  = "{" expression "}" ;
		// skipspaces  = "+" | "-" ;

		// title = text ;
		// comment = text ;

		// name = ( small | caps ) - { small | caps | digit | "_" } + ;
		// tag  = "<" text text ">" .

		// text        = dquotetext | squotetext ;

		// dquotetext = '"' - { small | caps | digit | special | "'" | '\\"' } '"' + ;
		// squotetext = "'" - { small | caps | digit | special | '"' | "\\'" } "'" + ;

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// special = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "~" | "@" ;

		// } "Some comment"`,

		// ==========================================

		// Minimal EBNF of EBNF (can NOT parse) (sequence | alternative can both not be together with the rest of the expression alternatives, but why? <- sequence and alternative can not have itself inside):
		// `"Minimal EBNF of EBNF" {
		// ebnf = "{" { production } "}" .
		// production  = name "=" [ expression ] ( "." | ";" ) .
		// expression  = name | text [ "..." text ] | group | option | repetition | skipspaces | alternative | sequence .
		// sequence    = expression expression { expression } .
		// alternative = expression "|" expression { "|" expression } .
		// group       = "(" expression ")" .
		// option      = "[" expression "]" .
		// repetition  = "{" expression "}" .
		// skipspaces = "+" | "-" .

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		// special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		// name = ( small | caps ) { small | caps | digit | "_" } .
		// text = "\"" - { small | caps | digit | special } "\"" + .
		// }`,

		// ==========================================

		// `{ top = "AB C" ; foo = "BAR" ; xyz = rtu mmx ; rtu = foo | "B" ; mmx = [xyz] ; zzz = ("U" | "V") "W" "P" { "Z" } }`,
		// `{ top = "AB C" ; foo = "BAR" ; xyz = rtu mmx ; rtu = foo | "B" ; mmx = [xyz] ; zzz = ("U" | "V") "W" "X" "Y" [ "Z" ] "P" { "Z" } ; } `,

		// ==========================================

		// expression  <"" '{{if (eq .globalVars.or true)}}{"OR", {{.childCode}}}{{else}}{{.childCode}}{{end}}'>          = alternative <"" '{{setGlobal "or" false}}{{.childCode}}'> { "|" alternative <"" '{{if (ne .childCode "")}}{{setGlobal "or" true}}, {{.childCode}}{{end}}'> } .
		// alternative <"" '{{if (eq .globalVars.seq true)}}{{"{"}}{{.childCode}}{{"}"}}{{else}}{{.childCode}}{{end}}'>   = term <"" '{{setGlobal "seq" false}}{{.childCode}}'> { term <"" '{{if (ne .childCode "")}}{{setGlobal "seq" true}}, {{.childCode}}{{end}}'> } .

		// sequence    = alternative <"" "{{.childCode}}"> { alternative <"" ", {{.childCode}}"> } .
		// sequence    <"" "{{if (eq .seq true)}}{ {{.childCode}} }{{else}}{{.childCode}}{{end}}">      = alternative <"" "{{setGlobal \"seq\" false}}{{.childCode}}"> { alternative <"" "{{if .childCode}}{{setGlobal \"seq\" true}}{{end}}, {{.childCode}}"> } .
		// TODO: braces are not correct and comma is often missing
		// TODO: implement directChildCount or hasMultipleChilds
		// TODO: for that use objects instead of childCode  (and implement a serializer and a deserializer)

		// {{upstream .vars.name (ident .vars.name) .childObj}}

		// // Bigger EBNF of EBNF with tags (can parse):
		// `"aEBNF of aEBNF" <"" "AA - ."> {
		// program     <"" '{{"{"}}{{.childCode}}}'>                                                                = [ title ] [ tag ] "{" [ production ] { production <"" ", {{.childCode}}"> } "}" [ tag ] [ comment ] .
		// production  <"" '{"{{.vars.name}}", {{ident .vars.name}}, {{.childCode}}}; {{addProduction .vars.name (ident .vars.name) .childObj}}{{upstream .vars.name (ident .vars.name) .childObj}}'>                            = name <"name"> [ tag ] "=" [ expression ] ( "." | ";" ) .
		// expression  <"" '{{if (eq .globalVars.or true)}}{"OR", {{.childCode}}}{{else}}{{.childCode}}{{end}}'>       = alternative <"" '{{setGlobal "or" false}}{{.childCode}}'> { "|" alternative <"" '{{setGlobal "or" true}}, {{.childCode}}'> } .
		// alternative <"" '{{"{"}}{{.childCode}}{{"}"}}'>                                                          = term <"" '{{.childCode}}'> { term <"" ', {{.childCode}}'> } .
		// term        = ( name | text [ "..." text ] | group | option | repetition | skipspaces ) [ tag ] .
		// group       <"" '{{"{"}}{{.childCode}}{{"}"}}'>             = "(" expression ")" .
		// option      <"" '{"OPTIONAL", {{.childCode}}}{{if notNil .childObj}}{{upstream "OPTIONAL" .childObj}}{{end}}'>             = "[" expression "]" .
		// repetition  <"" '{"REPEAT", {{.childCode}}}{{if notNil .childObj}}{{upstream "REPEAT" .childObj}}{{end}}'>               = "{" expression "}" .
		// skipspaces  = "+" <"" '{"SKIPSPACES", true}{{upstream "SKIPSPACES" true}}'> | "-" <"" '{"SKIPSPACES", false}{{upstream "SKIPSPACES" false}}'> .

		// title = text .
		// comment = text .

		// name        <"" '{"IDENT", "{{.childStr}}", {{ident .childStr}}}{{upstream "IDENT" .childStr (ident .childStr)}}'>                = ( small | caps ) - { small | caps | digit | "_" } + .
		// text        <"" '{"TERMINAL", {{.childStr}}}{{upstream "TERMINAL" .childStr}}'>                                                   = dquotetext | squotetext .

		// dquotetext = '"' - { small | caps | digit | special } '"' + .
		// squotetext = "'" - { small | caps | digit | special } "'" + .

		// tag  = "<" text text ">" .

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		// special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		// } "Some comment"`,

		// //  <'' '{{upstream "IDENT" .childStr (ident .childStr)}}'>
		// // {{if (and (gt (objCount .childObj) 1) (ne "string" (printf "%T" (index .childObj 0))))}}{{upstream "OR" .childObj}}{{end}}

		// // TODO:  prolog and epilog [ tag ] are not evaluated
		// // TODO: TAG does not include .childObj but it is appended !!!!!!!!!!! seems like some parts do not get pushed in upstream - debug

		// // Bigger EBNF of EBNF with tags (can parse):  ====================================== EDIT THIS ==============================================
		// `"aEBNF of aEBNF" {
		// program     = [ title ] [ tag ] "{" { addedproduction } "}" [ tag ] [ comment ] ;

		// addedproduction <'{{addProduction .locals.name (ident .locals.name) .childObj}}{{upstream (group .locals.name (ident .locals.name) (group .childObj))}}'>       = production ;
		// production <'{{if (notNil .locals.tagcode)}}{{upstream (group "TAG" (group .locals.tagcode) "BBBBBBBBBB"   (group .childObj)    "BBBBBBBBBB"   )}}{{end}}{{deleteLocalVar "tagcode"}}'> = ( taggedname "=" [ expression ] ";" ) [ tag ] ;

		// taggedname  <'{{if (notNil .locals.tagcode)}}{{upstream (group "TAG" (group .locals.tagcode) "AAAAAAAAAAAA>>>>>>>" (group .childObj)  "<<<<<<AAAAAAAAAAAA"  )}}{{end}}{{deleteLocalVar "tagcode"}}'>      = name <'{{setLocal "name" .childStr}}'> [ tag ] ;
		// expression  <'{{if (eq .locals.or "true")}}{{upstream (group "OR" .childObj)}}{{end}}'>                                                                     = alternative <'{{setLocal "or" "false"}}'> { "|" <'{{setLocal "or" "true"}}'> alternative } ;
		// alternative = taggedterm { taggedterm } ;
		// taggedterm  <'{{if (notNil .locals.tagcode)}}{{upstream (group "TAG" (group .locals.tagcode) (group .childObj))}}{{end}}{{deleteLocalVar "tagcode"}}'>      = term [ tag ] ;
		// term        = name <'{{.childStr}}{{upstream (group "IDENT" .childStr (ident .childStr))}}'> | range | group | option | repetition | skipspaces ;
		// range       = ( text [ "..." text ] ) ;
		// group       <'{{upstream (group .childObj)}}'>                                                                                  = "(" expression ")" ;
		// option      <'{{if notNil .childObj}}{{upstream (group "OPTIONAL" (group .childObj))}}{{end}}'>                                 = "[" expression "]" ;
		// repetition  <'{{if notNil .childObj}}{{upstream (group "REPEAT" (group .childObj))}}{{end}}'>                                   = "{" expression "}" ;
		// skipspaces  = "+" <'{{upstream (group "SKIPSPACES" true)}}'> | "-" <'{{upstream (group "SKIPSPACES" false)}}'> ;

		// title = text ;
		// comment = text ;

		// name        = ( small | caps ) - { small | caps | digit | "_" } + ;
		// tag         <'{{upstream nil}}'>                                                                                                = "<" text <'{{setLocal "tagcode" .childObj}}'> [ text <'{{setLocal "tagoptional" .childObj}}'> ] ">" ;
		// text        <'{{upstream (group "TERMINAL" .childCode)}}'>                                                                      = dquotetext | squotetext ;

		// dquotetext = '"' - { small | caps | digit | special | "'" | '\\"' } <'{{.childStr}}'> '"' + ;
		// squotetext = "'" - { small | caps | digit | special | '"' | "\\'" } <'{{.childStr}}'> "'" + ;

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// special = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "~" | "@" ;

		// } "Some comment"`,

		// `{ top <'{{upstream (group "up foo:" .locals.foo "TEST1" .childObj "TEST2")}}'> = "abc" c ;
		// c  = b <'{{upstream .childObj "aaaaaaaa"}}'> ;
		// b <'{{upstream (group "FOO:" .locals.foo)}}{{deleteLocalVar "foo"}}'> = "Z" | x ;
		// x = { "X" | "Y" } <'{{setLocal "foo" .childStr}}'> ; }`,

		//  <'' '{{upstream "IDENT" .childStr (ident .childStr)}}'>
		// {{if (and (gt (objCount .childObj) 1) (ne "string" (printf "%T" (index .childObj 0))))}}{{upstream "OR" .childObj}}{{end}}

		// TODO:  prolog and epilog [ tag ] are not evaluated
		// TODO: TAG does not include .childObj but it is appended !!!!!!!!!!! seems like some parts do not get pushed in upstream - debug

		// Bigger EBNF of EBNF with tags (can parse):  ====================================== EDIT THIS ==============================================
		// `{
		// 	program     = { addedproduction } ;

		// 	addedproduction <'{{addProduction .locals.name (ident .locals.name) .childObj}}{{upstream (group .locals.name (ident .locals.name) (group .childObj))}}'>       = production ;
		// 	production <'{{if (notNil .locals.tagcode)}}{{upstream (group "TAG" (group .locals.tagcode) "BBBBBBBBBB"   (group .childObj)    "BBBBBBBBBB"   )}}{{end}}{{deleteLocalVar "tagcode"}}'> = ( taggedname "=" [ expression ] ";" ) [ tag ] ;

		// 	taggedname  <'{{if (notNil .locals.tagcode)}}{{upstream (group "TAG" (group .locals.tagcode) "AAAAAAAAAAAA>>>>>>>" (group .childObj)  "<<<<<<AAAAAAAAAAAA"  )}}{{end}}{{deleteLocalVar "tagcode"}}'>      = name <'{{setLocal "name" .childStr}}'> [ tag ] ;
		// 	expression  <'{{if (eq .locals.or "true")}}{{upstream (group "OR" .childObj)}}{{end}}'>                                                                     = alternative <'{{setLocal "or" "false"}}'> { "|" <'{{setLocal "or" "true"}}'> alternative } ;
		// 	alternative = taggedterm { taggedterm } ;
		// 	taggedterm  <'{{if (notNil .locals.tagcode)}}{{upstream (group "TAG" (group .locals.tagcode) (group .childObj))}}{{end}}{{deleteLocalVar "tagcode"}}'>      = term [ tag ] ;
		// 	term        = name <'{{.childStr}}{{upstream (group "IDENT" .childStr (ident .childStr))}}'> | range | group | option | repetition | skipspaces ;
		// 	range       = ( text [ "..." text ] ) ;
		// 	group       <'{{upstream (group .childObj)}}'>                                                                                  = "(" expression ")" ;
		// 	option      <'{{if notNil .childObj}}{{upstream (group "OPTIONAL" (group .childObj))}}{{end}}'>                                 = "[" expression "]" ;
		// 	repetition  <'{{if notNil .childObj}}{{upstream (group "REPEAT" (group .childObj))}}{{end}}'>                                   = "{" expression "}" ;
		// 	skipspaces  = "+" <'{{upstream (group "SKIPSPACES" true)}}'> | "-" <'{{upstream (group "SKIPSPACES" false)}}'> ;

		// 	title = text ;
		// 	comment = text ;

		// 	name        = ( small | caps ) - { small | caps | digit | "_" } + ;
		// 	tag         <'{{upstream nil}}'>                                                                                                = "<" text <'{{setLocal "tagcode" .childObj}}'> [ text <'{{setLocal "tagoptional" .childObj}}'> ] ">" ;
		// 	text        <'{{upstream (group "TERMINAL" .childCode)}}'>                                                                      = dquotetext | squotetext ;

		// 	dquotetext = '"' - { small | caps | digit | special | "'" | '\\"' } <'{{.childStr}}'> '"' + ;
		// 	squotetext = "'" - { small | caps | digit | special | '"' | "\\'" } <'{{.childStr}}'> "'" + ;

		// 	digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// 	small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// 	caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// 	special = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "|" | "%" | "$" | "&" | "#" | "~" | "@" | "\\\\" | "\\t" | "\t" | "\\n" | "\n" | "\\r" | "\r" ;

		// 	}`,

		// `"aEBNF of aEBNF as text" {
		// 	program                                                                    = [ title ] [ tag ] "{" { production } "}" [ tag ] start [ comment ] .
		// 	production                                   = name [ tag ] "=" [ expression ] ( "." | ";" ) .
		// 	expression                                   = alternative { "|" alternative } .
		// 	alternative                                  = term { term } .
		// 	term                                         = ( name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag ] .
		// 	group                                        = "(" expression ")" .
		// 	option                                       = "[" expression "]" .
		// 	repetition                                   = "{" expression "}" .
		// 	skipspaces  = "+" | "-"  .

		// 	title = text .
		// 	start = name .
		// 	comment = text .

		// 	tag  = "<" text text ">" .

		// 	name                     = ( small | caps ) - { small | caps | digit | "_" } + .
		// 	text     <'c.upstream=c.childStr'>                = dquotetext | squotetext .

		// 	dquotetext = '"' - { small | caps | digit | special | "'" | '\\"' } '"' + ;
		// 	squotetext = "'" - { small | caps | digit | special | '"' | "\\'" } "'" + ;

		// 	digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// 	small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// 	caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// 	special = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "~" | "@" ;

		// 	} program`,

		// `"aEBNF of aEBNF as text" {
		// program     <'{{"{"}}{{.childCode}}{{"}"}}'>                                                                = [ title ] [ tag ] "{" [ production ] { production <", {{.childCode}}"> } "}" [ tag ] start [ comment ] .
		// production  <'{"{{.locals.name}}", {{ident .locals.name}}, {{.childCode}}{{"}"}}; {{addProduction .locals.name (ident .locals.name) .childObj}}{{upstream (group .locals.name (ident .locals.name) .childObj)}}'>          = name <'{{setLocal "name" .childStr}}'> [ tag ] "=" [ expression ] ( "." | ";" ) .
		// expression  <'{{if (eq .locals.or true)}}{"OR", {{.childCode}}{{"}"}}{{else}}{{.childCode}}{{end}}'>       = alternative <'{{setLocal "or" false}}{{.childCode}}'> { "|" alternative <'{{setLocal "or" true}}, {{.childCode}}'> } .
		// alternative <'{{if (gt (lenx .childObj) 1)}}{{"{"}}{{.childCode}}{{"}"}}{{else}}{{.childCode}}{{end}}'>                                                          = term <'{{.childCode}}'> { term <', {{.childCode}}'> } .
		// term        = ( name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag ] .
		// group       <'{{if (gt (lenx .childObj) 1)}}{{"{"}}{{.childCode}}{{"}"}}{{upstream (group .childObj)}}{{else}}{{.childCode}}{{end}}'>             = "(" expression ")" .
		// option      <'{{if notNil .childObj}}{"OPTIONAL", {{.childCode}}}{{upstream "OPTIONAL" (group .childObj)}}{{end}}'>                              = "[" expression "]" .
		// repetition  <'{{if notNil .childObj}}{"REPEAT", {{.childCode}}}{{upstream "REPEAT" (group .childObj)}}{{end}}'>                                  = "{" expression "}" .
		// skipspaces  = "+" <'{"SKIPSPACES", true}{{upstream "SKIPSPACES" true}}'> | "-" <'{"SKIPSPACES", false}{{upstream "SKIPSPACES" false}}'> .

		// title = text .
		// start = name .
		// comment = text .

		// tag  = "<" text text ">" .

		// name        <'{"IDENT", "{{.childStr}}", {{ident .childStr}}}{{upstream "IDENT" .childStr (ident .childStr)}}'>                = ( small | caps ) - { small | caps | digit | "_" } + .
		// text        <'{"TERMINAL", {{.childStr}}}{{upstream "TERMINAL" .childStr}}'>                                                   = dquotetext | squotetext .

		// dquotetext = '"' - { small | caps | digit | special | "'" | '\\"' } '"' + ;
		// squotetext = "'" - { small | caps | digit | special | '"' | "\\'" } "'" + ;

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// special = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "~" | "@" ;

		// } program`,

		`"aEBNF of aEBNF as text" <~~ var names = []; function getNameIdx(name) { pos = names.indexOf(name); if (pos != -1) { return pos }; return names.push(name) } ~~> {
			program          = [ title ] [ tag ] "{" { production } "}" [ tag ] start [ comment ] ;
			production       = name [ tag ] "=" [ expression <~~ upstream.text = '{"PRODUCTION", ' + upstream.name + " " + upstream.text + "}" ~~>  ] ( "." | ";" ) ;
			expression       = alternative { "|" alternative } ;
			alternative      = term { term } ;
			term             = ( name <~~ upstream.name = upstream.text; upstream.text = '{"IDENT", ' + upstream.text + "}" ~~> | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag ] ;
			group            = "(" expression ")" ;
			option           = "[" expression "]" ;
			repetition       = "{" expression "}" ;
			skipspaces       = "+" | "-" ;
	
			title            = text ;
			start            = name ;
			comment          = text ;
	
			tag              = "<" code {"," code } ">" ;

			code             <~~ upstream.text = '{"TERMINAL", ' + upstream.text + "}" ~~>              = '~\~' - { { codeinner } [ "~" ] codeinner } '~\~' + ;
			codeinner        = small | caps | digit | special | "'" | '\\"' | '"' | "\\'" | "\\~" ;

			name             = ( small | caps ) - { small | caps | digit | "_" } + ;

			text             <~~ upstream.text = '{"TERMINAL", ' + upstream.text + "}" ~~>              = dquotetext | squotetext ;
			dquotetext       = '"' - { small | caps | digit | special | "~" | "'" | '\\"' } '"' + ;
			squotetext       = "'" - { small | caps | digit | special | "~" | '"' | "\\'" } "'" + ;
	
			digit            = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
			small            = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
			caps             = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
			special          = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "@" ;
	
			} <~~ print(" >>" + upstream.text + "<< ") ~~> program`,
	}

	tests = []string{
		// 		"a1a3a4a4a5a6",
		// 		"a1 a2a6",
		// 		"a1 a3 a4 a6",
		// 		"a1 a4 a5 a6",
		// 		"a1 a2 a4 a5 a5 a6",
		// 		"a1 a2 a4 a5 a6 a7",
		// 		"your ad here",
		// 		"2",
		// 		"2*3 + 4/23 - 7",
		// 		"(3 + 4) * 6-2+(4*(4))",
		// 		"-2",
		// 		"3 +",
		// 		"(4 + 3",
		// 		`{ }`,
		// 		`{ moo < "test" ; "toast" > = "ABC" | "DEF" . }`,
		//
		// `"aEBNF of aEBNF" {
		// program = [ title ] [ tag ] "{" { production } "}" [ tag ] [ comment ] .
		// production  = name [ tag ] "=" [ expression ] ( "." | ";" ) .
		// expression  = sequence .
		// sequence    = alternative { alternative } .
		// alternative = term { "|" term } .
		// term        = ( name | text [ "..." text ] | group | option | repetition | skipspaces ) [ tag ] .
		// group       = "(" expression ")" .
		// option      = "[" expression "]" .
		// repetition  = "{" expression "}" .
		// skipspaces  < "foo" > = "+" | "-" .

		// title = text .
		// comment = text .

		// name = ( small | caps ) { small | caps | digit | "_" } .
		// text = "\"" - { small | caps | digit | special } "\"" + .

		// tag = "<" text { ";" text } ">" .

		// digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
		// small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
		// caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
		// special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .

		// } "Some comment"`,

		// `abc XYXXY`,
		// `{ top = "abc" <'setLocal "foo" "test"' ''> ; }`,
		// `{ top = abc { uvw } ; abc  = 'ABC' <'aaa' ''> ; uvw = 'XYZ' ; }`,
		// `{ top = "AB C" ; foo = "BAR" ; xyz = rtu ; rtu = foo | "B" ; }`,
		// `{ top = "AB C" ; foo = "BAR" ; xyz = rtu mmx ; rtu = foo | "B" ; mmx = [xyz] ; }`,

		// `{ top = "AB C" ; foo = "BAR" ; xyz = rtu mmx ; rtu = foo | "B" ; mmx = [xyz] ; zzz = ("U" | "V") "W" "X" "Y" [ "Z" ] "P" { "Z" } ; } `,

		// `
		// 	program     = { addedproduction } ;

		// 	addedproduction <'tagcode18'>       = production ;
		// 	production <'tagcode17'> = ( taggedname "=" [ expression ] ";" ) [ tag ] ;

		// 	taggedname  <'tagcode16'>      = name <'tagcode15'> [ tag ] ;
		// 	expression  <'tagcode14'>                                                                     = alternative <'tagcode'> { "|" <'tagcode13'> alternative } ;
		// 	alternative = taggedterm { taggedterm } ;
		// 	taggedterm  <'tagcode'>      = term [ tag ] ;
		// 	term        = name <'tagcode12'> | range | group | option | repetition | skipspaces ;
		// 	range       = ( text [ "..." text ] ) ;
		// 	group       <'tagcode11'>                                                                                  = "(" expression ")" ;
		// 	option      <'tagcode10'>                                 = "[" expression "]" ;
		// 	repetition  <'tagcode9'>                                   = "{" expression "}" ;
		// 	skipspaces  = "+" <'tagcode7'> | "-" <'tagcode8'> ;

		// 	title = text ;
		// 	comment = text ;

		// 	name        = ( small | caps ) - { small | caps | digit | "_" } + ;
		// 	tag         <'tagcode6'>                                                                                                = "<" text <'tagcode4'> [ text <'tagcode5'> ] ">" ;
		// 	text        <'tagcode3'>                                                                      = dquotetext | squotetext ;

		// 	dquotetext = '"' - { small | caps | digit | special | "'" | '\\"' } <'tagcode2'> '"' + ;
		// 	squotetext = "'" - { small | caps | digit | special | '"' | "\\'" } <'tagcode1'> "'" + ;

		// 	digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		// 	small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		// 	caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		// 	special = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "~" | "@" ;

		// 	`,

		`"EBNF of EBNF (can parse)" {
			program = [ title ] "{" { production } "}" [ comment ] ;
			production  = name "=" [ expression ] ";" ;
			expression  = sequence ;
			sequence    = alternative { alternative } ;
			alternative = term { "|" term } ;
			term        = name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ;
			group       = "(" expression ")" ;
			option      = "[" expression "]" ;
			repetition  = "{" expression "}" ;
			skipspaces = "+" | "-" ;
			title = text ;
			comment = text ;
			name = ( small | caps ) { small | caps | digit | "_" } ;
			text = "\"" - { small | caps | digit | special } "\"" + ;
			digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
			small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
			caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
			special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" ;
			} program`,
	}
)

func main() {
	// speedtest()
	// return

	for _, srcEBNF := range ebnfs {
		fmt.Print("\n===========================================================\nEBNF:\n===========================================================\n\n")
		ebnf.PprintSrc("Parse", srcEBNF)
		// Parses an EBNF and generates a grammar with it.
		grammar, err := ebnf.ParseEBNF(srcEBNF)
		if err != nil {
			fmt.Println("  ==> Fail")
			fmt.Println(err)
			continue
		}
		fmt.Println("  ==> Success\n\n  Grammar:")
		fmt.Println("   => Extras: " + ebnf.PprintExtrasShort(&grammar.Extras, "    "))
		fmt.Println("   => Productions: " + ebnf.PprintProductionsShort(&grammar.Productions, "    "))

		fmt.Print("\n\n==================\nTests:\n==================\n\n")
		for _, srcCode := range tests {
			fmt.Println("Parse via grammar:")
			ebnf.PprintSrcSingleLine(srcCode)
			// Uses the grammar to parse the with it described text. It generates the ASG (abstract semantic graph) of the parsed text.
			asg, err := ebnf.ParseWithGrammar(grammar, srcCode, false)
			if err != nil {
				fmt.Println("\n  ==> Fail")
				fmt.Println(err)
				continue
			}
			fmt.Println("\n  ==> Success\n\n  Abstract syntax tree:")
			fmt.Println("    " + ebnf.PprintProductionsShort(&asg, "    "))

			fmt.Println("\nCode output:")
			// Uses the annotations inside the ASG to compile it.
			_, err = ebnf.CompileASG(asg, &grammar.Extras, true)
			if err != nil {
				fmt.Println("\n  ==> Fail")
				fmt.Println(err)
				continue
			}
			fmt.Print("\n ==> Success\n\n")
		}
		fmt.Println()
	}

}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
func speedtest() {
	src := `{
		program = [ title ] "{" { production } "}" [ comment ] ;
		production  = name "=" [ expression ] ";" ;
		expression  = sequence ;
		sequence    = alternative { alternative } ;
		alternative = term { "|" term } ;
		term        = name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ;
		group       = "(" expression ")" ;
		option      = "[" expression "]" ;
		repetition  = "{" expression "}" ;
		skipspaces = "+" | "-" ;
		title = text ;
		comment = text ;
		name = ( small | caps ) { small | caps | digit | "_" } ;
		text = "\"" - { small | caps | digit | special } "\"" + ;
		digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
		small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
		caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
		special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" ;
		} program`

	defer timeTrack(time.Now(), "parse DMA")
	var err error = nil
	for i := 0; i < 10000; i++ {
		_, err = ebnf.ParseEBNF(src)
	}
	if err != nil {
		fmt.Println("Error")
		return
	}
	// fmt.Printf("%#v", g)
}
