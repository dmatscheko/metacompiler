// Package flag: Go's flag package WRITTEN IN GO, compiled with a program that
// imports "flag" (core.importFile in both go grammars), for int, int64, uint,
// string, bool and float64 flags. The command line is os.Args, which holds the
// program alone here, so flag.Parse leaves every default; a FlagSet parses what it
// is given as Go's does.
package flag

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

type ErrorHandling int

const (
	ContinueOnError ErrorHandling = iota
	ExitOnError
	PanicOnError
)

var ErrHelp = errors.New("flag: help requested")

type Flag struct {
	Name     string
	Usage    string
	DefValue string
	kind     string
	ip       *int
	i64p     *int64
	up       *uint
	sp       *string
	bp       *bool
	fp       *float64
}

// The value s, stored as Go's flag values store one: even when it does not parse
// (an int flag given "abc" becomes 0), with "parse error" or "value out of range".
func (f *Flag) set(s string) error {
	var err error
	switch f.kind {
	case "int":
		v, e := strconv.ParseInt(s, 0, 64)
		*f.ip = int(v)
		err = numError(e)
	case "int64":
		v, e := strconv.ParseInt(s, 0, 64)
		*f.i64p = v
		err = numError(e)
	case "uint":
		v, e := strconv.ParseUint(s, 0, 64)
		*f.up = uint(v)
		err = numError(e)
	case "string":
		*f.sp = s
	case "bool":
		v, e := strconv.ParseBool(s)
		*f.bp = v
		if e != nil {
			err = errParse
		}
	case "float64":
		v, e := strconv.ParseFloat(s, 64)
		*f.fp = v
		err = numError(e)
	}
	return err
}

var errParse = errors.New("parse error")
var errRange = errors.New("value out of range")

func numError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, strconv.ErrRange) {
		return errRange
	}
	return errParse
}

type FlagSet struct {
	Usage         func()
	name          string
	handling      ErrorHandling
	parsed        bool
	formal        map[string]*Flag
	order         []string
	actual        map[string]*Flag
	args          []string
	output        io.Writer
}

func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet {
	return &FlagSet{name: name, handling: errorHandling, formal: map[string]*Flag{}, actual: map[string]*Flag{}}
}

var CommandLine = NewFlagSet(os.Args[0], ExitOnError)

// Where usage and error messages go: os.Stderr unless SetOutput said otherwise.
func (f *FlagSet) Output() io.Writer {
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}
func (f *FlagSet) SetOutput(w io.Writer) { f.output = w }
func (f *FlagSet) Name() string          { return f.name }
func (f *FlagSet) Parsed() bool   { return f.parsed }
func (f *FlagSet) Args() []string { return f.args }
func (f *FlagSet) NArg() int      { return len(f.args) }
func (f *FlagSet) NFlag() int     { return len(f.actual) }
func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}

func (f *FlagSet) Lookup(name string) *Flag { return f.formal[name] }

func (f *FlagSet) add(fl *Flag) {
	if _, dup := f.formal[fl.Name]; dup {
		panic(f.name + " flag redefined: " + fl.Name)
	}
	f.formal[fl.Name] = fl
	f.order = append(f.order, fl.Name)
}

func (f *FlagSet) IntVar(p *int, name string, value int, usage string) {
	*p = value
	f.add(&Flag{Name: name, Usage: usage, DefValue: strconv.Itoa(value), kind: "int", ip: p})
}
func (f *FlagSet) Int(name string, value int, usage string) *int {
	p := new(int)
	f.IntVar(p, name, value, usage)
	return p
}
func (f *FlagSet) Int64Var(p *int64, name string, value int64, usage string) {
	*p = value
	f.add(&Flag{Name: name, Usage: usage, DefValue: strconv.FormatInt(value, 10), kind: "int64", i64p: p})
}
func (f *FlagSet) Int64(name string, value int64, usage string) *int64 {
	p := new(int64)
	f.Int64Var(p, name, value, usage)
	return p
}
func (f *FlagSet) UintVar(p *uint, name string, value uint, usage string) {
	*p = value
	f.add(&Flag{Name: name, Usage: usage, DefValue: strconv.FormatUint(uint64(value), 10), kind: "uint", up: p})
}
func (f *FlagSet) Uint(name string, value uint, usage string) *uint {
	p := new(uint)
	f.UintVar(p, name, value, usage)
	return p
}
func (f *FlagSet) StringVar(p *string, name string, value string, usage string) {
	*p = value
	f.add(&Flag{Name: name, Usage: usage, DefValue: value, kind: "string", sp: p})
}
func (f *FlagSet) String(name string, value string, usage string) *string {
	p := new(string)
	f.StringVar(p, name, value, usage)
	return p
}
func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string) {
	*p = value
	f.add(&Flag{Name: name, Usage: usage, DefValue: strconv.FormatBool(value), kind: "bool", bp: p})
}
func (f *FlagSet) Bool(name string, value bool, usage string) *bool {
	p := new(bool)
	f.BoolVar(p, name, value, usage)
	return p
}
func (f *FlagSet) Float64Var(p *float64, name string, value float64, usage string) {
	*p = value
	f.add(&Flag{Name: name, Usage: usage, DefValue: strconv.FormatFloat(value, 'g', -1, 64), kind: "float64", fp: p})
}
func (f *FlagSet) Float64(name string, value float64, usage string) *float64 {
	p := new(float64)
	f.Float64Var(p, name, value, usage)
	return p
}

func (f *FlagSet) Set(name, value string) error {
	fl, ok := f.formal[name]
	if !ok {
		return errors.New("no such flag -" + name)
	}
	if err := fl.set(value); err != nil {
		return err
	}
	f.actual[name] = fl
	return nil
}

// The flags in lexical order, as Go's VisitAll visits them.
func (f *FlagSet) PrintDefaults() {
	names := append([]string{}, f.order...)
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	for _, name := range names {
		fl := f.formal[name]
		line := "  -" + fl.Name
		switch fl.kind {
		case "bool":
		case "float64":
			line += " float"
		case "int64":
			line += " int"
		default:
			line += " " + fl.kind
		}
		if len(line) <= 4 {
			line += "\t" + fl.Usage
		} else {
			line += "\n    \t" + fl.Usage
		}
		if fl.DefValue != "" && fl.DefValue != "0" && fl.DefValue != "false" {
			if fl.kind == "string" {
				line += " (default " + strconv.Quote(fl.DefValue) + ")"
			} else {
				line += " (default " + fl.DefValue + ")"
			}
		}
		fmt.Fprintln(f.Output(), line)
	}
}

func (f *FlagSet) usage() {
	if f.Usage != nil {
		f.Usage()
		return
	}
	if f.name == "" {
		fmt.Fprintln(f.Output(), "Usage:")
	} else {
		fmt.Fprintln(f.Output(), "Usage of "+f.name+":")
	}
	f.PrintDefaults()
}

func (f *FlagSet) fail(msg string) error {
	fmt.Fprintln(f.Output(), msg)
	f.usage()
	return errors.New(msg)
}

// One argument at the front of f.args: false when the flags have ended.
func (f *FlagSet) parseOne() (bool, error) {
	if len(f.args) == 0 {
		return false, nil
	}
	s := f.args[0]
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	minuses := 1
	if s[1] == '-' {
		minuses++
		if len(s) == 2 {
			f.args = f.args[1:]
			return false, nil
		}
	}
	name := s[minuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, f.fail("bad flag syntax: " + s)
	}
	f.args = f.args[1:]
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ {
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}
	fl, ok := f.formal[name]
	if !ok {
		if name == "help" || name == "h" {
			f.usage()
			return false, ErrHelp
		}
		return false, f.fail("flag provided but not defined: -" + name)
	}
	if fl.kind == "bool" {
		if hasValue {
			if err := fl.set(value); err != nil {
				return false, f.fail("invalid boolean value " + strconv.Quote(value) + " for -" + name + ": " + err.Error())
			}
		} else {
			fl.set("true")
		}
	} else {
		if !hasValue && len(f.args) > 0 {
			hasValue = true
			value, f.args = f.args[0], f.args[1:]
		}
		if !hasValue {
			return false, f.fail("flag needs an argument: -" + name)
		}
		if err := fl.set(value); err != nil {
			return false, f.fail("invalid value " + strconv.Quote(value) + " for flag -" + name + ": " + err.Error())
		}
	}
	f.actual[name] = fl
	return true, nil
}

func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = arguments
	for {
		seen, err := f.parseOne()
		if seen {
			continue
		}
		if err == nil {
			break
		}
		switch f.handling {
		case ContinueOnError:
			return err
		case ExitOnError:
			if err == ErrHelp {
				os.Exit(0)
			}
			os.Exit(2)
		case PanicOnError:
			panic(err)
		}
	}
	return nil
}

func Parse()                  { CommandLine.Parse(os.Args[1:]) }
func Parsed() bool            { return CommandLine.Parsed() }
func Args() []string          { return CommandLine.Args() }
func NArg() int               { return CommandLine.NArg() }
func Arg(i int) string        { return CommandLine.Arg(i) }
func NFlag() int              { return CommandLine.NFlag() }
func Lookup(name string) *Flag { return CommandLine.Lookup(name) }
func Set(name, value string) error { return CommandLine.Set(name, value) }
func PrintDefaults()          { CommandLine.PrintDefaults() }

func IntVar(p *int, name string, value int, usage string)       { CommandLine.IntVar(p, name, value, usage) }
func Int(name string, value int, usage string) *int             { return CommandLine.Int(name, value, usage) }
func Int64Var(p *int64, name string, value int64, usage string) { CommandLine.Int64Var(p, name, value, usage) }
func Int64(name string, value int64, usage string) *int64       { return CommandLine.Int64(name, value, usage) }
func UintVar(p *uint, name string, value uint, usage string)    { CommandLine.UintVar(p, name, value, usage) }
func Uint(name string, value uint, usage string) *uint          { return CommandLine.Uint(name, value, usage) }
func StringVar(p *string, name string, value string, usage string) {
	CommandLine.StringVar(p, name, value, usage)
}
func String(name string, value string, usage string) *string { return CommandLine.String(name, value, usage) }
func BoolVar(p *bool, name string, value bool, usage string)  { CommandLine.BoolVar(p, name, value, usage) }
func Bool(name string, value bool, usage string) *bool        { return CommandLine.Bool(name, value, usage) }
func Float64Var(p *float64, name string, value float64, usage string) {
	CommandLine.Float64Var(p, name, value, usage)
}
func Float64(name string, value float64, usage string) *float64 {
	return CommandLine.Float64(name, value, usage)
}
