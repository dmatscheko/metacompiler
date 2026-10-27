package abnf

import (
	"strings"
	"testing"
)

const flagSrc = `// A language.
//
// @flag mode plain|loud default=plain How to greet.
// @flag level 0|1|2 default=1 How much.

:title("x") ;
// @flag late a|b default=a Not in the leading block, so not a declaration.
`

func TestParseFlagDecls(t *testing.T) {
	decls, err := ParseFlagDecls("g.abnf", flagSrc)
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 2 || decls[0].Name != "mode" || decls[1].Default != "1" ||
		strings.Join(decls[1].Values, ",") != "0,1,2" || decls[0].Desc != "How to greet." {
		t.Fatalf("got %+v", decls)
	}
	for src, want := range map[string]string{
		"// @flag mode plain|loud default=plain\n":                   "g.abnf:1: malformed @flag line (it needs",
		"// @flag Mode a|b default=a d\n":                            `name "Mode"`,
		"// @flag m a|a default=a d\n":                               `value "a" is empty, repeated`,
		"// @flag m a|b, default=a d\n":                              `value "b,"`,
		"// @flag m a|b default=c d\n":                               `"default=c" is not default=VALUE`,
		"\n// @flag m a|b default=a d\n// @flag m b|c default=b d\n": "g.abnf:3: malformed @flag line (flag \"m\" is declared twice)",
	} {
		if _, err := ParseFlagDecls("g.abnf", src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got error %v, want it to contain %q", src, err, want)
		}
	}
}

func TestApplyFlagSettings(t *testing.T) {
	decls, _ := ParseFlagDecls("g.abnf", flagSrc)
	stages := func() []*StageFlags {
		return []*StageFlags{{Stage: 2, File: "g.abnf", Decls: decls}, {Stage: 4, File: "h.abnf"}}
	}
	settings := func(arg string) []FlagSetting {
		kv, err := ParseFlagSettings("-cf", arg)
		if err != nil {
			t.Fatal(err)
		}
		return kv
	}

	// A numbered setting wins over an unnumbered one whatever the order; within
	// one kind the last setting wins.
	s := stages()
	if err := ApplyFlagSettings(s, settings("mode=plain,level=2,level=0"), map[int][]FlagSetting{2: settings("mode=loud")}); err != nil {
		t.Fatal(err)
	}
	SetStageFlags(s[0])
	defer SetStageFlags(nil)
	if flagValue("mode") != "loud" || flagValue("level") != "0" {
		t.Fatalf("got mode=%s level=%s", flagValue("mode"), flagValue("level"))
	}
	s = stages()
	if err := ApplyFlagSettings(s, nil, nil); err != nil {
		t.Fatal(err)
	}
	SetStageFlags(s[0])
	if flagValue("mode") != "plain" || flagValue("level") != "1" {
		t.Fatal("an unset flag must read as its declared default")
	}
	func() {
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(error).Error(), `c.flag("late")`) {
				t.Errorf("reading an undeclared flag must raise, got %v", r)
			}
		}()
		flagValue("late")
	}()

	for _, c := range []struct {
		all      string
		numbered map[int][]FlagSetting
		want     string
	}{
		{"nope=1", nil, `-cf: no stage declares a flag "nope"`},
		{"mode=shout", nil, `-cf: flag "mode" of stage 2 (grammar g.abnf) takes plain|loud, got "shout"`},
		{"", map[int][]FlagSetting{2: {{"x", "1"}}}, "-cf2: stage 2 (grammar g.abnf) declares no flag \"x\"\nstage 2 (grammar g.abnf) flags (-cf2 name=value):\n  mode=plain|loud (default plain)  How to greet."},
		{"", map[int][]FlagSetting{4: {{"mode", "loud"}}}, `-cf4: stage 4 (grammar h.abnf) declares no flag "mode"`},
		{"", map[int][]FlagSetting{3: {{"mode", "loud"}}}, "-cf3: stage 3 runs no grammar file"},
	} {
		var all []FlagSetting
		if c.all != "" {
			all = settings(c.all)
		}
		if err := ApplyFlagSettings(stages(), all, c.numbered); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q %v: got %v, want it to contain %q", c.all, c.numbered, err, c.want)
		}
	}
	if _, err := ParseFlagSettings("-cf2", "mode"); err == nil {
		t.Error("a setting without '=' must be rejected")
	}
	if h := FlagHelp(stages(), 0); !strings.Contains(h, "level=0|1|2 (default 1)  How much.") || !strings.Contains(h, "stage 4 (grammar h.abnf) declares no flags") {
		t.Errorf("help: %s", h)
	}
}
