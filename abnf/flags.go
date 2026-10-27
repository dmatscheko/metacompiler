package abnf

// Compiler flags: per-language options a grammar declares and its tag scripts
// read through c.flag(name), set on the command line with -cfN name=value.
//
// A grammar declares them in its LEADING comment block - the `//` lines before
// the first line of grammar text - one line per flag:
//
//	// @flag NAME VALUE|VALUE|... default=VALUE One-line description.
//
// That block is the flag's documentation: `mec -cf help` prints it. Only the
// leading block is read, so an @flag line further down is ordinary comment.
// NAME is [a-z][a-z0-9_]*, each VALUE is [A-Za-z0-9_.+-]+ (no commas: a -cf
// argument separates settings with them), the default is one of the values,
// and the description is required.
//
// The flags reach a script as plain strings. They fix what a grammar emits, so
// a native -exe bakes them in, and the frozen script cache is unaffected: it
// caches compiled script SOURCE, and c.flag is a run-time call into this file.

import (
	"fmt"
	"sort"
	"strings"
)

// FlagDecl is one declared compiler flag.
type FlagDecl struct {
	Name    string
	Values  []string
	Default string
	Desc    string
}

// StageFlags is what the driver knows about one pipeline stage's flags: the
// grammar file whose @flag block declares them, the declarations, and the
// values the command line set (an unset flag reads as its default).
type StageFlags struct {
	Stage int
	File  string
	Decls []FlagDecl
	Set   map[string]string
}

// FlagSetting is one name=value pair from a -cf / -cfN argument.
type FlagSetting struct{ Name, Value string }

// ParseFlagDecls reads the @flag lines of a grammar source's leading comment
// block. A malformed @flag line is an error naming the file and line.
func ParseFlagDecls(file, src string) ([]FlagDecl, error) {
	var decls []FlagDecl
	for i, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "//") {
			break
		}
		f := strings.Fields(strings.TrimPrefix(line, "//"))
		if len(f) == 0 || f[0] != "@flag" {
			continue
		}
		bad := func(why string) error {
			return fmt.Errorf("%s:%d: malformed @flag line (%s); the form is\n"+
				"  // @flag NAME VALUE|VALUE|... default=VALUE One-line description.", file, i+1, why)
		}
		if len(f) < 5 {
			return nil, bad("it needs a name, the values, default= and a description")
		}
		d := FlagDecl{Name: f[1], Values: strings.Split(f[2], "|"), Desc: strings.Join(f[4:], " ")}
		if !flagWord(d.Name, true) {
			return nil, bad(fmt.Sprintf("name %q is not [a-z][a-z0-9_]*", d.Name))
		}
		seen := map[string]bool{}
		for _, v := range d.Values {
			if !flagWord(v, false) || seen[v] {
				return nil, bad(fmt.Sprintf("value %q is empty, repeated or not [A-Za-z0-9_.+-]+", v))
			}
			seen[v] = true
		}
		if !strings.HasPrefix(f[3], "default=") || !seen[strings.TrimPrefix(f[3], "default=")] {
			return nil, bad(fmt.Sprintf("%q is not default=VALUE with one of the declared values", f[3]))
		}
		d.Default = strings.TrimPrefix(f[3], "default=")
		for _, o := range decls {
			if o.Name == d.Name {
				return nil, bad(fmt.Sprintf("flag %q is declared twice", d.Name))
			}
		}
		decls = append(decls, d)
	}
	return decls, nil
}

// flagWord checks a flag name (lower-case identifier) or a flag value.
func flagWord(s string, name bool) bool {
	if s == "" || (name && !(s[0] >= 'a' && s[0] <= 'z')) {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		ok := ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '_'
		if !name {
			ok = ok || ch >= 'A' && ch <= 'Z' || ch == '.' || ch == '+' || ch == '-'
		}
		if !ok {
			return false
		}
	}
	return true
}

// ParseFlagSettings splits one -cf argument, "a=1,b=2", into its settings.
func ParseFlagSettings(flag, arg string) ([]FlagSetting, error) {
	var out []FlagSetting
	for _, part := range strings.Split(arg, ",") {
		eq := strings.IndexByte(part, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("%s needs name=value[,name=value...] or help, got %q", flag, arg)
		}
		out = append(out, FlagSetting{part[:eq], part[eq+1:]})
	}
	return out, nil
}

// ApplyFlagSettings validates the command line's settings against the stages'
// declarations and records them in each stage's Set. Numbered settings (-cfN)
// apply to stage N only and win over unnumbered ones (-cf), which apply to every
// stage that declares the name; within either kind the last setting wins.
func ApplyFlagSettings(stages []*StageFlags, all []FlagSetting, numbered map[int][]FlagSetting) error {
	byStage := map[int]*StageFlags{}
	for _, s := range stages {
		byStage[s.Stage] = s
		s.Set = map[string]string{}
	}
	for _, kv := range all {
		found := false
		for _, s := range stages {
			if d := s.decl(kv.Name); d != nil {
				found = true
				if err := s.check("-cf", d, kv.Value); err != nil {
					return err
				}
				s.Set[kv.Name] = kv.Value
			}
		}
		if !found {
			return fmt.Errorf("-cf: no stage declares a flag %q\n%s", kv.Name, FlagHelp(stages, 0))
		}
	}
	nums := make([]int, 0, len(numbered))
	for n := range numbered {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		s := byStage[n]
		for _, kv := range numbered[n] {
			flag := fmt.Sprintf("-cf%d", n)
			if s == nil {
				return fmt.Errorf("%s: stage %d runs no grammar file, so it has no flags\n%s", flag, n, FlagHelp(stages, 0))
			}
			d := s.decl(kv.Name)
			if d == nil {
				return fmt.Errorf("%s: %s declares no flag %q\n%s", flag, s.where(), kv.Name, FlagHelp(stages, n))
			}
			if err := s.check(flag, d, kv.Value); err != nil {
				return err
			}
			s.Set[kv.Name] = kv.Value
		}
	}
	return nil
}

func (s *StageFlags) decl(name string) *FlagDecl {
	for i := range s.Decls {
		if s.Decls[i].Name == name {
			return &s.Decls[i]
		}
	}
	return nil
}

func (s *StageFlags) where() string { return fmt.Sprintf("stage %d (grammar %s)", s.Stage, s.File) }

func (s *StageFlags) check(flag string, d *FlagDecl, v string) error {
	for _, ok := range d.Values {
		if v == ok {
			return nil
		}
	}
	return fmt.Errorf("%s: flag %q of %s takes %s, got %q\n%s", flag, d.Name, s.where(), strings.Join(d.Values, "|"), v, FlagHelp([]*StageFlags{s}, 0))
}

// FlagHelp lists the declared flags of every stage, or of stage only (> 0).
func FlagHelp(stages []*StageFlags, only int) string {
	var b strings.Builder
	for _, s := range stages {
		if only > 0 && s.Stage != only {
			continue
		}
		if len(s.Decls) == 0 {
			fmt.Fprintf(&b, "%s declares no flags\n", s.where())
			continue
		}
		fmt.Fprintf(&b, "%s flags (-cf%d name=value):\n", s.where(), s.Stage)
		for _, d := range s.Decls {
			fmt.Fprintf(&b, "  %s=%s (default %s)  %s\n", d.Name, strings.Join(d.Values, "|"), d.Default, d.Desc)
		}
	}
	if only > 0 && b.Len() == 0 {
		fmt.Fprintf(&b, "stage %d runs no grammar file, so it has no flags\n", only)
	}
	if b.Len() == 0 {
		b.WriteString("no stage is run by a grammar file, so no flags are declared\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// curStageFlags is the stage whose parse and compile is running; c.flag reads it.
var curStageFlags *StageFlags

// SetStageFlags makes s the flags c.flag answers from (nil: the stage declares none).
func SetStageFlags(s *StageFlags) { curStageFlags = s }

// flagValue is c.flag(name) in every engine that runs tag scripts: the value the
// command line set, else the declared default. Reading an undeclared name is a
// grammar error, not a silent undefined.
func flagValue(name string) string {
	s := curStageFlags
	if s != nil {
		if v, ok := s.Set[name]; ok {
			return v
		}
		if d := s.decl(name); d != nil {
			return d.Default
		}
	}
	panic(fmt.Errorf("c.flag(%q): the grammar does not declare that flag (add a `// @flag` line to its leading comment block)", name))
}
