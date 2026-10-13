package abnf

// jsrtluaos.go -- Lua's os library for the compiler grammar
// (languages/lua-to-llvm-ir.abnf): time, clock, date, difftime, getenv and
// exit. The civil time is a line-by-line port of the lo* section of
// languages/lib/lua-strlib.js, which the interpreter half includes and layer 2
// imports; keep the three in step. The machine is asked through osQuery
// (jsrtint.go), the host builtin both of the others call.

import (
	"math"
	"os"
	"strings"
)

var loDayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
var loMonNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September",
	"October", "November", "December"}

const loIntMax = 2147483647

func loFloor(a, b float64) float64 { return math.Floor(a / b) }
func loMod(a, b float64) float64   { return a - math.Floor(a/b)*b }

// loDays is the day count from 1970-01-01 to y-m-d (m 1..12), proleptic Gregorian.
func loDays(y, m, d float64) float64 {
	yy := y
	if m <= 2 {
		yy = y - 1
	}
	era := loFloor(yy, 400)
	yoe := yy - era*400
	mp := m + 9
	if m > 2 {
		mp = m - 3
	}
	doy := loFloor(153*mp+2, 5) + d - 1
	return era*146097 + yoe*365 + loFloor(yoe, 4) - loFloor(yoe, 100) + doy - 719468
}

// loCivil is the date of day z counted from 1970-01-01.
func loCivil(z0 float64) (float64, float64, float64) {
	z := z0 + 719468
	era := loFloor(z, 146097)
	doe := z - era*146097
	yoe := loFloor(doe-loFloor(doe, 1460)+loFloor(doe, 36524)-loFloor(doe, 146096), 365)
	doy := doe - (365*yoe + loFloor(yoe, 4) - loFloor(yoe, 100))
	mp := loFloor(5*doy+2, 153)
	m := mp - 9
	if mp < 10 {
		m = mp + 3
	}
	y := yoe + era*400
	if m <= 2 {
		y++
	}
	return y, m, doy - loFloor(153*mp+2, 5) + 1
}

// loTm is a broken-down time in lua's units (see loBreak in lua-strlib.js).
type loTm struct {
	year, month, day, hour, min, sec, wday, yday, zoff float64
	isdst                                              bool
	zone                                               string
}

// loBreak is the broken-down time t, UTC or local, or nil when its year leaves
// struct tm's int range.
func loBreak(t float64, utc bool) *loTm {
	y0, _, _ := loCivil(math.Floor(t / 86400))
	if y0-1900 > loIntMax || y0-1900 < -loIntMax-1 {
		return nil
	}
	zoff, lt := 0.0, t
	if utc {
		zoff = loStdOff()
	} else {
		zoff = osQuery(2, t).(float64)
		lt = t + zoff
	}
	days := math.Floor(lt / 86400)
	secs := lt - days*86400
	y, m, d := loCivil(days)
	f := &loTm{year: y, month: m, day: d, hour: math.Floor(secs / 3600), min: math.Mod(math.Floor(secs/60), 60),
		sec: math.Mod(secs, 60), wday: loMod(days+4, 7) + 1, yday: days - loDays(y, 1, 1) + 1, zoff: zoff, zone: "UTC"}
	if !utc {
		f.isdst = osQuery(4, t).(bool)
		f.zone = lu8FromHost(osQuery(3, t).(string))
	}
	return f
}

// loOffWhere is the local zone's offset in t's year at mid-January or mid-July,
// whichever has the daylight saving state dst, else mktime's guess (daylight
// saving time one hour ahead).
func loOffWhere(t float64, dst bool) float64 {
	y, _, _ := loCivil(math.Floor(t / 86400))
	for _, at := range []float64{loDays(y, 1, 15) * 86400, loDays(y, 7, 15) * 86400} {
		if osQuery(4, at).(bool) == dst {
			return osQuery(2, at).(float64)
		}
	}
	if dst {
		return osQuery(2, t).(float64) + 3600
	}
	return osQuery(2, t).(float64) - 3600
}

// loStdOff is the local zone's offset outside daylight saving time as of now:
// what darwin's strftime prints as %z for a UTC time.
func loStdOff() float64 { return loOffWhere(osQuery(0, 0).(float64), false) }

// loMkTime is mktime over lua's fields (see loMkTime in lua-strlib.js).
func loMkTime(y, mo, d, h, mi, s float64, isdst int) float64 {
	yy := y + loFloor(mo-1, 12)
	local := (loDays(yy, loMod(mo-1, 12)+1, 1)+d-1)*86400 + h*3600 + mi*60 + s
	off := func(t float64) float64 { return osQuery(2, t).(float64) }
	dst := func(t float64) bool { return osQuery(4, t).(bool) }
	t := local - off(local)
	t = local - off(t)
	if isdst >= 0 && (isdst > 0) != dst(t) {
		return local - loOffWhere(t, isdst > 0)
	}
	return t
}

func loPad(n float64, w int) string {
	s := luNumStr(n)
	for len(s) < w {
		s = "0" + s
	}
	return s
}

func loYear(y float64) string {
	if y >= 0 {
		return loPad(y, 4)
	}
	return luNumStr(y)
}

func loIsoP(y float64) float64 {
	return loMod(y+loFloor(y, 4)-loFloor(y, 100)+loFloor(y, 400), 7)
}

func loIsoWeeks(y float64) float64 {
	if loIsoP(y) == 4 || loIsoP(y-1) == 3 {
		return 53
	}
	return 52
}

// loIso is the ISO 8601 year and week of a broken-down time.
func loIso(f *loTm) (float64, float64) {
	wd := f.wday - 1
	if f.wday == 1 {
		wd = 7
	}
	wk := loFloor(f.yday-wd+10, 7)
	if wk < 1 {
		return f.year - 1, loIsoWeeks(f.year - 1)
	}
	if wk > loIsoWeeks(f.year) {
		return f.year + 1, 1
	}
	return f.year, wk
}

// loConv is one conversion of strftime in the C locale.
func loConv(c byte, f *loTm) string {
	switch c {
	case 'a':
		return loDayNames[int(f.wday)-1][:3]
	case 'A':
		return loDayNames[int(f.wday)-1]
	case 'b', 'h':
		return loMonNames[int(f.month)-1][:3]
	case 'B':
		return loMonNames[int(f.month)-1]
	case 'c':
		return loConv('a', f) + " " + loConv('b', f) + " " + loConv('e', f) + " " + loConv('T', f) + " " + loConv('Y', f)
	case 'C':
		return loPad(loFloor(f.year, 100), 2)
	case 'd':
		return loPad(f.day, 2)
	case 'D', 'x':
		return loConv('m', f) + "/" + loConv('d', f) + "/" + loConv('y', f)
	case 'e':
		if f.day < 10 {
			return " " + luNumStr(f.day)
		}
		return luNumStr(f.day)
	case 'F':
		return loConv('Y', f) + "-" + loConv('m', f) + "-" + loConv('d', f)
	case 'g':
		y, _ := loIso(f)
		return loPad(loMod(y, 100), 2)
	case 'G':
		y, _ := loIso(f)
		return loYear(y)
	case 'H':
		return loPad(f.hour, 2)
	case 'I':
		h := math.Mod(f.hour, 12)
		if h == 0 {
			h = 12
		}
		return loPad(h, 2)
	case 'j':
		return loPad(f.yday, 3)
	case 'm':
		return loPad(f.month, 2)
	case 'M':
		return loPad(f.min, 2)
	case 'n':
		return "\n"
	case 'p':
		if f.hour < 12 {
			return "AM"
		}
		return "PM"
	case 'r':
		return loConv('I', f) + ":" + loConv('M', f) + ":" + loConv('S', f) + " " + loConv('p', f)
	case 'R':
		return loConv('H', f) + ":" + loConv('M', f)
	case 'S':
		return loPad(f.sec, 2)
	case 't':
		return "\t"
	case 'T', 'X':
		return loConv('H', f) + ":" + loConv('M', f) + ":" + loConv('S', f)
	case 'u':
		if f.wday == 1 {
			return "7"
		}
		return luNumStr(f.wday - 1)
	case 'U':
		return loPad(loFloor(f.yday+7-f.wday, 7), 2)
	case 'V':
		_, w := loIso(f)
		return loPad(w, 2)
	case 'w':
		return luNumStr(f.wday - 1)
	case 'W':
		return loPad(loFloor(f.yday+6-loMod(f.wday+5, 7), 7), 2)
	case 'y':
		return loPad(loMod(f.year, 100), 2)
	case 'Y':
		return loYear(f.year)
	case 'z':
		sign, a := "+", f.zoff
		if f.zoff < 0 {
			sign, a = "-", -f.zoff
		}
		return sign + loPad(loFloor(a, 3600), 2) + loPad(math.Mod(loFloor(a, 60), 60), 2)
	case 'Z':
		return f.zone
	}
	return "%"
}

const (
	loOpt1 = "aAbBcCdDeFgGhHIjmMnprRStTuUVwWxXyYzZ%"
	loOptE = "cCxXyY"
	loOptO = "deHImMSuUVwWy"
)

// loFormat is os.date's format over a broken-down time: the text, or false and
// the rest of the format from a conversion lua 5.5 does not accept.
func loFormat(fmt string, f *loTm) (string, bool) {
	var out strings.Builder
	for i := 0; i < len(fmt); {
		if fmt[i] != '%' {
			out.WriteByte(fmt[i])
			i++
			continue
		}
		if i+1 >= len(fmt) {
			return fmt[i+1:], false
		}
		cc := fmt[i+1]
		conv, used := cc, 2
		if cc == 'E' || cc == 'O' {
			allowed := loOptE
			if cc == 'O' {
				allowed = loOptO
			}
			if i+2 >= len(fmt) || strings.IndexByte(allowed, fmt[i+2]) < 0 {
				return fmt[i+1:], false
			}
			conv, used = fmt[i+2], 3
		} else if strings.IndexByte(loOpt1, cc) < 0 {
			return fmt[i+1:], false
		}
		out.WriteString(loConv(conv, f))
		i += used
	}
	return out.String(), true
}

// lu8FromHost is host text as the Lua byte string of its UTF-8. Twin of
// lu8FromHost in lua-strlib.js.
func lu8FromHost(s string) string {
	rs := make([]rune, len(s))
	for i := 0; i < len(s); i++ {
		rs[i] = rune(s[i])
	}
	return string(rs)
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		argsOf := func(h uint64) []interface{} {
			if arr, ok := u(h).(*jsArray); ok {
				return arr.elems
			}
			return nil
		}
		// getfield: an integer field of os.time's table, through its __index.
		field := func(t interface{}, key string, d, delta float64) float64 {
			v := rt.getMember(t, luKey(key))
			if luIsNil(v) && rt.luMetaOf(t) != nil {
				v = rt.luIndexMeta(t, key)
			}
			n, ok := luToNum(v)
			r := luNumOf(n)
			if !ok || !luIsInt(n) && (r-r != 0 || r != math.Trunc(r) || r >= luMinIntF || r < -luMinIntF) {
				if !luIsNil(v) {
					rt.luFail("field '%s' is not an integer", key)
				}
				if d < 0 {
					rt.luFail("field '%s' missing in date table", key)
				}
				return d
			}
			if r >= 0 && r-delta > loIntMax || r < 0 && -loIntMax-1+delta > r {
				rt.luFail("field '%s' is out-of-bound", key)
			}
			return r
		}
		// setallfields, in loslib.c's order.
		fill := func(t interface{}, f *loTm) {
			set := func(k string, v interface{}) {
				if rt.luMetaOf(t) != nil {
					rt.luSetIndexMeta(t, k, v)
				} else {
					rt.setMember(t, luKey(k), v)
				}
			}
			set("year", f.year)
			set("month", f.month)
			set("day", f.day)
			set("hour", f.hour)
			set("min", f.min)
			set("sec", f.sec)
			set("yday", f.yday)
			set("wday", f.wday)
			set("isdst", f.isdst)
		}
		m["js_luostime"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			if luIsNil(luArgAt(args, 1)) {
				return w(osQuery(0, 0))
			}
			t := rt.luCheckTab("os.time", 1, args)
			y := field(t, "year", -1, 1900)
			mo := field(t, "month", -1, 1)
			d := field(t, "day", -1, 0)
			h := field(t, "hour", 12, 0)
			mi := field(t, "min", 0, 0)
			s := field(t, "sec", 0, 0)
			dst := rt.getMember(t, luKey("isdst"))
			if luIsNil(dst) && rt.luMetaOf(t) != nil {
				dst = rt.luIndexMeta(t, "isdst")
			}
			hint := -1
			if !luIsNil(dst) {
				hint = 0
				if luTruthy(dst) {
					hint = 1
				}
			}
			r := loMkTime(y, mo, d, h, mi, s, hint)
			f := loBreak(r, false)
			if f == nil {
				rt.luFail("time result cannot be represented in this installation")
			}
			fill(t, f)
			return w(r)
		}
		m["js_luosclock"] = func(a []uint64) uint64 { return w(luMkFlo(osQuery(1, 0).(float64))) }
		m["js_luosdate"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			s := rt.luOptStr("os.date", 1, args, "%c")
			tv := osQuery(0, 0).(float64)
			if !luIsNil(luArgAt(args, 2)) {
				tv = rt.luCheckInt("os.date", 2, args)
			}
			utc := strings.HasPrefix(s, "!")
			if utc {
				s = s[1:]
			}
			f := loBreak(tv, utc)
			if f == nil {
				rt.luFail("date result cannot be represented in this installation")
			}
			if s == "*t" {
				out := newJSObject()
				fill(out, f)
				return w(out)
			}
			text, ok := loFormat(s, f)
			if !ok {
				rt.luArgErr("os.date", 1, "invalid conversion specifier '%"+text+"'")
			}
			return rt.wrapStr(text)
		}
		m["js_luosdifftime"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t1 := rt.luCheckInt("os.difftime", 1, args)
			return w(luMkFlo(t1 - rt.luCheckInt("os.difftime", 2, args)))
		}
		m["js_luosgetenv"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			if v, ok := os.LookupEnv(luOutText(rt.luCheckStr("os.getenv", 1, args))); ok {
				return rt.wrapStr(lu8FromHost(v))
			}
			return w(jsUndef)
		}
		// exit([code]): true or absent is EXIT_SUCCESS, false EXIT_FAILURE.
		m["js_luosexit"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			c := luArgAt(args, 1)
			switch {
			case luIsNil(c) || c == true:
				os.Exit(0)
			case c == false:
				os.Exit(1)
			}
			os.Exit(int(int32(rt.luCheckInt("os.exit", 1, args))))
			return w(jsUndef)
		}
	})
}
