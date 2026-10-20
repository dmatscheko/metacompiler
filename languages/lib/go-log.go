// Package log: Go's log package WRITTEN IN GO, compiled with a program that
// imports "log" (core.importFile in both go grammars). A line is the prefix, the
// date and time the flags ask for, then the message; this runtime has no clock,
// so the time is the zero Time's (lib/go-time.go).
package log

import (
	"fmt"
	"io"
	"os"
	"time"
)

const (
	Ldate = 1 << iota
	Ltime
	Lmicroseconds
	Llongfile
	Lshortfile
	LUTC
	Lmsgprefix
	LstdFlags = Ldate | Ltime
)

type Logger struct {
	out    io.Writer
	prefix string
	flag   int
}

func New(out io.Writer, prefix string, flag int) *Logger { return &Logger{out, prefix, flag} }

var std = New(os.Stderr, "", LstdFlags)

func Default() *Logger { return std }

func (l *Logger) SetOutput(w io.Writer)  { l.out = w }
func (l *Logger) SetPrefix(p string)     { l.prefix = p }
func (l *Logger) SetFlags(flag int)      { l.flag = flag }
func (l *Logger) Flags() int             { return l.flag }
func (l *Logger) Prefix() string         { return l.prefix }
func (l *Logger) Writer() io.Writer      { return l.out }

// One line: s with a newline unless it has one.
func (l *Logger) Output(calldepth int, s string) error {
	h := ""
	if l.flag&Lmsgprefix == 0 {
		h += l.prefix
	}
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		t := time.Now()
		if l.flag&Ldate != 0 {
			h += t.Format("2006/01/02 ")
		}
		if l.flag&Lmicroseconds != 0 {
			h += t.Format("15:04:05.000000 ")
		} else if l.flag&Ltime != 0 {
			h += t.Format("15:04:05 ")
		}
	}
	if l.flag&(Lshortfile|Llongfile) != 0 {
		h += "???:1: "
	}
	if l.flag&Lmsgprefix != 0 {
		h += l.prefix
	}
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}
	_, err := io.WriteString(l.out, h+s)
	return err
}

func (l *Logger) Print(v ...any)                 { l.Output(2, fmt.Sprint(v...)) }
func (l *Logger) Printf(format string, v ...any) { l.Output(2, fmt.Sprintf(format, v...)) }
func (l *Logger) Println(v ...any)               { l.Output(2, fmt.Sprintln(v...)) }
func (l *Logger) Fatal(v ...any) {
	l.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}
func (l *Logger) Fatalf(format string, v ...any) {
	l.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
func (l *Logger) Fatalln(v ...any) {
	l.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}
func (l *Logger) Panic(v ...any) {
	s := fmt.Sprint(v...)
	l.Output(2, s)
	panic(s)
}
func (l *Logger) Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	l.Output(2, s)
	panic(s)
}
func (l *Logger) Panicln(v ...any) {
	s := fmt.Sprintln(v...)
	l.Output(2, s)
	panic(s)
}

func SetOutput(w io.Writer)              { std.out = w }
func SetPrefix(p string)                 { std.prefix = p }
func SetFlags(flag int)                  { std.flag = flag }
func Flags() int                         { return std.flag }
func Prefix() string                     { return std.prefix }
func Writer() io.Writer                  { return std.out }
func Output(calldepth int, s string) error { return std.Output(calldepth+1, s) }
func Print(v ...any)                     { std.Output(2, fmt.Sprint(v...)) }
func Printf(format string, v ...any)     { std.Output(2, fmt.Sprintf(format, v...)) }
func Println(v ...any)                   { std.Output(2, fmt.Sprintln(v...)) }
func Fatal(v ...any) {
	std.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}
func Fatalf(format string, v ...any) {
	std.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
func Fatalln(v ...any) {
	std.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}
func Panic(v ...any) {
	s := fmt.Sprint(v...)
	std.Output(2, s)
	panic(s)
}
func Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	std.Output(2, s)
	panic(s)
}
func Panicln(v ...any) {
	s := fmt.Sprintln(v...)
	std.Output(2, s)
	panic(s)
}
