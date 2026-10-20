// Package time: Go's time package WRITTEN IN GO, compiled with a program that
// imports "time" (core.importFile in both go grammars). A Time is an instant in
// UTC - there is no zone database, and Local is UTC - kept as seconds since
// January 1 of year 1 and a nanosecond, so the zero Time is that instant as in Go.
// This runtime has no clock: Now answers the zero Time and Sleep returns at once,
// so only the arithmetic and the formatting of given instants are Go's.
package time

import "errors"

type Duration int64

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

// "1h2m3.5s", "1.5ms", "0s": the largest unit first, a fraction without its
// trailing zeros.
func (d Duration) String() string {
	if d == 0 {
		return "0s"
	}
	neg := d < 0
	u := uint64(d)
	if neg {
		u = uint64(-(d + 1)) + 1
	}
	s := ""
	switch {
	case u < uint64(Microsecond):
		s = utoa(u) + "ns"
	case u < uint64(Millisecond):
		s = fracText(u, 3) + "µs"
	case u < uint64(Second):
		s = fracText(u, 6) + "ms"
	default:
		s = fracText(u%uint64(Minute), 9) + "s"
		u /= uint64(Minute)
		if u > 0 {
			s = utoa(u%60) + "m" + s
			u /= 60
			if u > 0 {
				s = utoa(u) + "h" + s
			}
		}
	}
	if neg {
		return "-" + s
	}
	return s
}

func utoa(u uint64) string {
	if u == 0 {
		return "0"
	}
	s := ""
	for u > 0 {
		s = string(rune('0'+u%10)) + s
		u /= 10
	}
	return s
}

// v / 10^prec, with the digits of the remainder after a point when it has any.
func fracText(v uint64, prec int) string {
	p := uint64(1)
	for i := 0; i < prec; i++ {
		p *= 10
	}
	f := v % p
	if f == 0 {
		return utoa(v / p)
	}
	digits := ""
	for i := 0; i < prec; i++ {
		digits = string(rune('0'+f%10)) + digits
		f /= 10
	}
	end := len(digits)
	for end > 0 && digits[end-1] == '0' {
		end--
	}
	return utoa(v/p) + "." + digits[:end]
}

func (d Duration) Nanoseconds() int64  { return int64(d) }
func (d Duration) Microseconds() int64 { return int64(d) / 1000 }
func (d Duration) Milliseconds() int64 { return int64(d) / 1000000 }
func (d Duration) Seconds() float64 {
	return float64(d/Second) + float64(d%Second)/1e9
}
func (d Duration) Minutes() float64 {
	return float64(d/Minute) + float64(d%Minute)/(60*1e9)
}
func (d Duration) Hours() float64 {
	return float64(d/Hour) + float64(d%Hour)/(60*60*1e9)
}
func (d Duration) Abs() Duration {
	if d >= 0 {
		return d
	}
	return -d
}

// d rounded toward zero to a multiple of m (d itself when m <= 0).
func (d Duration) Truncate(m Duration) Duration {
	if m <= 0 {
		return d
	}
	return d - d%m
}

// d rounded to the nearest multiple of m, halfway values away from zero.
func (d Duration) Round(m Duration) Duration {
	if m <= 0 {
		return d
	}
	r := d % m
	if d < 0 {
		r = -r
		if r+r < m {
			return d + r
		}
		if d1 := d - m + r; d1 < d {
			return d1
		}
		return Duration(-1 << 63)
	}
	if r+r < m {
		return d - r
	}
	if d1 := d + m - r; d1 > d {
		return d1
	}
	return Duration(1<<63 - 1)
}

type Month int

const (
	January Month = 1 + iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

var monthNames = []string{"January", "February", "March", "April", "May", "June", "July",
	"August", "September", "October", "November", "December"}

func (m Month) String() string {
	if m >= January && m <= December {
		return monthNames[m-1]
	}
	return "%!Month(" + itoa(int(m)) + ")"
}

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var dayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

func (d Weekday) String() string {
	if d >= Sunday && d <= Saturday {
		return dayNames[d]
	}
	return "%!Weekday(" + itoa(int(d)) + ")"
}

func itoa(n int) string {
	if n < 0 {
		return "-" + utoa(uint64(-n))
	}
	return utoa(uint64(n))
}

type Location struct{ name string }

func (l *Location) String() string { return l.name }

var UTC = &Location{"UTC"}
var Local = UTC

// The seconds from January 1 of year 1 to the Unix epoch.
const unixToInternal int64 = 62135596800

type Time struct {
	sec  int64 // since January 1, year 1, 00:00:00 UTC
	nsec int64 // [0, 1e9)
}

// The day number of a civil date, counted from January 1 of year 1 (day 0), in the
// proleptic Gregorian calendar.
func daysFrom(y int64, m int64, d int64) int64 {
	if m <= 2 {
		y--
	}
	era := floorDiv(y, 400)
	yoe := y - era*400
	mp := (m + 9) % 12
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 306
}

// The civil date of a day number (daysFrom's inverse).
func civil(days int64) (int64, int64, int64) {
	z := days + 306
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if m > 12 {
		m -= 12
	}
	if m <= 2 {
		y++
	}
	return y, m, d
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// Date normalizes as Go does: October 32 is November 1, -1 seconds the minute before.
func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time {
	m := int64(month) - 1
	y := int64(year) + floorDiv(m, 12)
	m = m - floorDiv(m, 12)*12 + 1
	ns := int64(nsec)
	s := int64(sec) + floorDiv(ns, 1000000000)
	ns = ns - floorDiv(ns, 1000000000)*1000000000
	total := (daysFrom(y, m, 1)+int64(day)-1)*86400 + int64(hour)*3600 + int64(min)*60 + s
	return Time{total, ns}
}

func Unix(sec int64, nsec int64) Time {
	sec += floorDiv(nsec, 1000000000)
	nsec -= floorDiv(nsec, 1000000000) * 1000000000
	return Time{sec + unixToInternal, nsec}
}

func UnixMilli(msec int64) Time { return Unix(floorDiv(msec, 1000), (msec-floorDiv(msec, 1000)*1000)*1000000) }

func Now() Time { return Time{} }

func Since(t Time) Duration { return Now().Sub(t) }
func Until(t Time) Duration { return t.Sub(Now()) }
func Sleep(d Duration)      {}

func (t Time) Unix() int64      { return t.sec - unixToInternal }
func (t Time) UnixMilli() int64 { return (t.sec-unixToInternal)*1000 + t.nsec/1000000 }
func (t Time) UnixNano() int64  { return (t.sec-unixToInternal)*1000000000 + t.nsec }
func (t Time) IsZero() bool     { return t.sec == 0 && t.nsec == 0 }
func (t Time) UTC() Time        { return t }
func (t Time) Local() Time      { return t }
func (t Time) In(loc *Location) Time { return t }
func (t Time) Location() *Location   { return UTC }

func (t Time) Date() (year int, month Month, day int) {
	y, m, d := civil(floorDiv(t.sec, 86400))
	return int(y), Month(m), int(d)
}

func (t Time) Clock() (hour, min, sec int) {
	s := t.sec - floorDiv(t.sec, 86400)*86400
	return int(s / 3600), int(s % 3600 / 60), int(s % 60)
}

func (t Time) Year() int {
	y, _, _ := t.Date()
	return y
}
func (t Time) Month() Month {
	_, m, _ := t.Date()
	return m
}
func (t Time) Day() int {
	_, _, d := t.Date()
	return d
}
func (t Time) Hour() int {
	h, _, _ := t.Clock()
	return h
}
func (t Time) Minute() int {
	_, m, _ := t.Clock()
	return m
}
func (t Time) Second() int {
	_, _, s := t.Clock()
	return s
}
func (t Time) Nanosecond() int { return int(t.nsec) }

// January 1 of year 1 was a Monday.
func (t Time) Weekday() Weekday {
	return Weekday((floorDiv(t.sec, 86400) + 1) % 7)
}

func (t Time) YearDay() int {
	y, _, _ := t.Date()
	return int(floorDiv(t.sec, 86400)-daysFrom(int64(y), 1, 1)) + 1
}

func (t Time) Add(d Duration) Time {
	ns := t.nsec + int64(d)%1000000000
	s := t.sec + int64(d)/1000000000 + floorDiv(ns, 1000000000)
	return Time{s, ns - floorDiv(ns, 1000000000)*1000000000}
}

func (t Time) Sub(u Time) Duration {
	return Duration((t.sec-u.sec)*1000000000 + (t.nsec - u.nsec))
}

func (t Time) AddDate(years int, months int, days int) Time {
	y, m, d := t.Date()
	h, mi, s := t.Clock()
	return Date(y+years, m+Month(months), d+days, h, mi, s, int(t.nsec), UTC)
}

func (t Time) Before(u Time) bool {
	return t.sec < u.sec || t.sec == u.sec && t.nsec < u.nsec
}
func (t Time) After(u Time) bool {
	return t.sec > u.sec || t.sec == u.sec && t.nsec > u.nsec
}
func (t Time) Equal(u Time) bool { return t.sec == u.sec && t.nsec == u.nsec }
func (t Time) Compare(u Time) int {
	if t.Before(u) {
		return -1
	}
	if t.After(u) {
		return 1
	}
	return 0
}

// t rounded down to a multiple of d since the zero Time.
func (t Time) Truncate(d Duration) Time {
	if d <= 0 {
		return t
	}
	return t.Add(-Duration(t.mod(d)))
}

func (t Time) Round(d Duration) Time {
	if d <= 0 {
		return t
	}
	r := t.mod(d)
	if r+r < int64(d) {
		return t.Add(-Duration(r))
	}
	return t.Add(Duration(int64(d) - r))
}

// The nanoseconds since the zero Time, modulo d, without overflowing int64.
func (t Time) mod(d Duration) int64 {
	dd := int64(d)
	return (mulMod(t.sec, 1000000000, dd) + t.nsec) % dd
}

// a * b modulo m, in [0, m), for b >= 0, by doubling.
func mulMod(a, b, m int64) int64 {
	r := int64(0)
	a %= m
	if a < 0 {
		a += m
	}
	for b > 0 {
		if b&1 == 1 {
			r = (r + a) % m
		}
		a = (a + a) % m
		b >>= 1
	}
	return r
}

func (t Time) String() string { return t.Format("2006-01-02 15:04:05.999999999 -0700 MST") }

const (
	Layout      = "01/02 03:04:05PM '06 -0700"
	ANSIC       = "Mon Jan _2 15:04:05 2006"
	UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
	RFC822      = "02 Jan 06 15:04 MST"
	RFC822Z     = "02 Jan 06 15:04 -0700"
	RFC850      = "Monday, 02-Jan-06 15:04:05 MST"
	RFC1123     = "Mon, 02 Jan 2006 15:04:05 MST"
	RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700"
	RFC3339     = "2006-01-02T15:04:05Z07:00"
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
	Kitchen     = "3:04PM"
	Stamp       = "Jan _2 15:04:05"
	StampMilli  = "Jan _2 15:04:05.000"
	StampMicro  = "Jan _2 15:04:05.000000"
	StampNano   = "Jan _2 15:04:05.000000000"
	DateTime    = "2006-01-02 15:04:05"
	DateOnly    = "2006-01-02"
	TimeOnly    = "15:04:05"
)

// The layout's elements, recognized as Go's nextStdChunk does: the reference time
// Mon Jan 2 15:04:05 MST 2006 spelled in any of its forms, anything else verbatim.
func (t Time) Format(layout string) string {
	y, mo, d := t.Date()
	h, mi, s := t.Clock()
	out := ""
	i := 0
	for i < len(layout) {
		rest := layout[i:]
		n, text := t.element(rest, y, mo, d, h, mi, s)
		if n == 0 {
			out += layout[i : i+1]
			i++
			continue
		}
		out += text
		i += n
	}
	return out
}

func pad(n int, w int) string {
	if n < 0 {
		return "-" + pad(-n, w)
	}
	s := itoa(n)
	for len(s) < w {
		s = "0" + s
	}
	return s
}

func has(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }

// The length of the element rest starts with (0 for none) and its text for t.
func (t Time) element(rest string, y int, mo Month, d, h, mi, s int) (int, string) {
	switch rest[0] {
	case 'J':
		if has(rest, "January") {
			return 7, mo.String()
		}
		if has(rest, "Jan") {
			return 3, mo.String()[:3]
		}
	case 'M':
		if has(rest, "Monday") {
			return 6, t.Weekday().String()
		}
		if has(rest, "Mon") {
			return 3, t.Weekday().String()[:3]
		}
		if has(rest, "MST") {
			return 3, "UTC"
		}
	case '0':
		if has(rest, "002") {
			return 3, pad(t.YearDay(), 3)
		}
		if len(rest) >= 2 && rest[1] >= '1' && rest[1] <= '6' {
			switch rest[1] {
			case '1':
				return 2, pad(int(mo), 2)
			case '2':
				return 2, pad(d, 2)
			case '3':
				return 2, pad(hour12(h), 2)
			case '4':
				return 2, pad(mi, 2)
			case '5':
				return 2, pad(s, 2)
			case '6':
				return 2, pad(y%100, 2)
			}
		}
	case '1':
		if has(rest, "15") {
			return 2, pad(h, 2)
		}
		return 1, itoa(int(mo))
	case '2':
		if has(rest, "2006") {
			return 4, pad(y, 4)
		}
		return 1, itoa(d)
	case '_':
		if has(rest, "_2") {
			if has(rest, "_2006") {
				return 0, ""
			}
			if d < 10 {
				return 2, " " + itoa(d)
			}
			return 2, itoa(d)
		}
		if has(rest, "__2") {
			yd := itoa(t.YearDay())
			for len(yd) < 3 {
				yd = " " + yd
			}
			return 3, yd
		}
	case '3':
		return 1, itoa(hour12(h))
	case '4':
		return 1, itoa(mi)
	case '5':
		return 1, itoa(s)
	case 'P':
		if has(rest, "PM") {
			if h >= 12 {
				return 2, "PM"
			}
			return 2, "AM"
		}
	case 'p':
		if has(rest, "pm") {
			if h >= 12 {
				return 2, "pm"
			}
			return 2, "am"
		}
	case '-':
		for _, z := range []string{"-07:00:00", "-070000", "-07:00", "-0700", "-07"} {
			if has(rest, z) {
				return len(z), zoneText(z[1:], "+")
			}
		}
	case 'Z':
		for _, z := range []string{"Z07:00:00", "Z070000", "Z07:00", "Z0700", "Z07"} {
			if has(rest, z) {
				return len(z), "Z"
			}
		}
	case '.', ',':
		if len(rest) >= 2 && (rest[1] == '0' || rest[1] == '9') {
			n := 1
			for n < len(rest) && rest[n] == rest[1] {
				n++
			}
			if n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
				return 0, ""
			}
			return n, fracNanos(t.nsec, n-1, rest[1] == '9', rest[:1])
		}
	}
	return 0, ""
}

// UTC's offset as an element spells it ("07:00" -> "+00:00").
func zoneText(z string, sign string) string {
	out := sign
	for i := 0; i < len(z); i++ {
		if z[i] == ':' {
			out += ":"
		} else {
			out += "0"
		}
	}
	return out
}

func hour12(h int) int {
	h = h % 12
	if h == 0 {
		return 12
	}
	return h
}

// The first digits of nsec after point: all of them for zeros (".000"), without
// their trailing zeros - and without the point when none is left - for nines.
func fracNanos(nsec int64, digits int, trim bool, point string) string {
	s := pad(int(nsec), 9)[:digits]
	if trim {
		end := len(s)
		for end > 0 && s[end-1] == '0' {
			end--
		}
		if end == 0 {
			return ""
		}
		s = s[:end]
	}
	return point + s
}

// ParseDuration: a sequence of decimal numbers, each with an optional fraction and a
// unit ("ns", "us", "µs", "ms", "s", "m", "h"), signed as a whole.
func ParseDuration(s string) (Duration, error) {
	orig := s
	neg := false
	if s != "" && (s[0] == '-' || s[0] == '+') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, errors.New("time: invalid duration " + quote(orig))
	}
	var total float64
	for s != "" {
		i := 0
		for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
			i++
		}
		if i == 0 {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		num := s[:i]
		s = s[i:]
		j := 0
		for j < len(s) && !(s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
			j++
		}
		unit := s[:j]
		s = s[j:]
		var scale float64
		switch unit {
		case "ns":
			scale = 1
		case "us", "µs", "μs":
			scale = 1e3
		case "ms":
			scale = 1e6
		case "s":
			scale = 1e9
		case "m":
			scale = 60e9
		case "h":
			scale = 3600e9
		case "":
			return 0, errors.New("time: missing unit in duration " + quote(orig))
		default:
			return 0, errors.New("time: unknown unit " + quote(unit) + " in duration " + quote(orig))
		}
		total += decimal(num) * scale
	}
	if neg {
		total = -total
	}
	return Duration(total), nil
}

func decimal(s string) float64 {
	v := 0.0
	frac := 0.0
	scale := 1.0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			frac = 1
			continue
		}
		if frac > 0 {
			scale /= 10
			v += float64(s[i]-'0') * scale
		} else {
			v = v*10 + float64(s[i]-'0')
		}
	}
	return v
}

func quote(s string) string { return "\"" + s + "\"" }
