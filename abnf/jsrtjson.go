package abnf

// JSON.stringify and JSON.parse for the js and typescript compiler halves under
// llvm.Run: a port of languages/lib/json.js, which the interpreter halves include
// and layer 2 imports. Keep the two in step.

import (
	"strconv"
	"strings"
)

// The program's JSON global, declared by the emitters in the program's own scope.
func (rt *jsrt) addJSONExterns(m map[string]func(args []uint64) uint64) {
	m["js_jsjson"] = func(a []uint64) uint64 {
		// The two members are UNLISTED props, so Object.keys and for-in skip them:
		// node's are not enumerable (js_jsmath's members are kept the same way).
		o := newJSObject()
		o.props["stringify"] = jsHostFunc("stringify", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			s, ok := rt.jsonStringify(argAt(args, 0), argAt(args, 1), argAt(args, 2))
			if !ok {
				return jsUndef
			}
			return s
		})
		o.props["parse"] = jsHostFunc("parse", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jsonParse(argAt(args, 0), argAt(args, 1))
		})
		o.props["__strtag"] = "JSON" // Object.prototype.toString's tag; see jsvProtoTag.
		return rt.wrap(o)
	}
}

type jsonState struct {
	gap, indent string
	stack       []interface{}
	fn          interface{}
	allow       []string
}

func (rt *jsrt) jsonStringify(value, replacer, space interface{}) (string, bool) {
	st := &jsonState{gap: jsonGap(space)}
	if rt.jsonIsFn(replacer) {
		st.fn = replacer
	} else if arr, ok := replacer.(*jsArray); ok {
		for _, e := range arr.elems {
			switch e.(type) {
			case string, float64:
				k := rt.jsvString(e)
				dup := false
				for _, x := range st.allow {
					dup = dup || x == k
				}
				if !dup {
					st.allow = append(st.allow, k)
				}
			}
		}
		if st.allow == nil {
			st.allow = []string{}
		}
	}
	wrapper := newJSObject()
	wrapper.set("", value)
	return rt.jsonSer(st, wrapper, "", value)
}

// jsonGap: a number is that many spaces (at most 10), a string its first ten
// characters, anything else none.
func jsonGap(space interface{}) string {
	switch s := space.(type) {
	case float64:
		n := 0
		for n < 10 && float64(n+1) <= s {
			n++
		}
		return strings.Repeat(" ", n)
	case string:
		u := strUnits(s)
		if len(u) > 10 {
			return strFromUnits(u[:10])
		}
		return s
	}
	return ""
}

func (rt *jsrt) jsonIsFn(v interface{}) bool { return isCallable(v) || isClassDesc(v) }

// jsonIsObj: an object that serializes by its keys.
func jsonIsObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok || isClassDesc(o) {
		return false
	}
	_, isSym := o.props["__symkey"]
	return !isSym
}

// jsonToJSON is v.toJSON(key); a class method takes its receiver PREPENDED (see
// jsvCallMethod).
func (rt *jsrt) jsonToJSON(v interface{}, key string) (interface{}, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if m, found := jsvFindClassMethod(o, "toJSON"); found {
		return rt.call(m, jsUndef, []interface{}{o, key}), true
	}
	if m, has := o.props["toJSON"]; has && isCallable(m) {
		return rt.call(m, o, []interface{}{key}), true
	}
	return nil, false
}

// jsonSer is SerializeJSONProperty; false means the property is skipped.
func (rt *jsrt) jsonSer(st *jsonState, holder interface{}, key string, v interface{}) (string, bool) {
	if _, isArr := v.(*jsArray); isArr || jsonIsObj(v) {
		if r, ok := rt.jsonToJSON(v, key); ok {
			v = r
		}
	}
	if st.fn != nil {
		v = rt.call(st.fn, holder, []interface{}{key, v})
	}
	switch x := v.(type) {
	case jsNullT:
		return "null", true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case string:
		return jsonQuote(x), true
	case float64:
		if x != x || x-x != 0 {
			return "null", true
		}
		return rt.jsvString(x), true
	case *jsArray:
		return rt.jsonArr(st, x), true
	}
	if _, isBig := bigOf(v); isBig {
		rt.bigRaise("%s", "TypeError: Do not know how to serialize a BigInt")
	}
	if jsonIsObj(v) {
		return rt.jsonObj(st, v.(*jsObject)), true
	}
	return "", false
}

// jsonQuote is QuoteJSONString, over UTF-16 code units: a lone surrogate is escaped.
func jsonQuote(s string) string {
	u := strUnits(s)
	out := make([]uint16, 0, len(u)+2)
	add := func(t string) {
		for i := 0; i < len(t); i++ {
			out = append(out, uint16(t[i]))
		}
	}
	const hex = "0123456789abcdef"
	esc := func(c uint16) {
		add("\\u")
		out = append(out, uint16(hex[c>>12&15]), uint16(hex[c>>8&15]), uint16(hex[c>>4&15]), uint16(hex[c&15]))
	}
	out = append(out, '"')
	for i := 0; i < len(u); i++ {
		c := u[i]
		switch {
		case c == '"':
			add("\\\"")
		case c == '\\':
			add("\\\\")
		case c == 8:
			add("\\b")
		case c == 12:
			add("\\f")
		case c == 10:
			add("\\n")
		case c == 13:
			add("\\r")
		case c == 9:
			add("\\t")
		case c < 32:
			esc(c)
		case c >= 0xD800 && c <= 0xDFFF:
			if c <= 0xDBFF && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] <= 0xDFFF {
				out = append(out, c, u[i+1])
				i++
			} else {
				esc(c)
			}
		default:
			out = append(out, c)
		}
	}
	out = append(out, '"')
	return strFromUnits(out)
}

// jsonEnter: a cycle is node's TypeError (node's message goes on to describe the
// cycle, which no engine here builds).
func (rt *jsrt) jsonEnter(st *jsonState, v interface{}) {
	for _, s := range st.stack {
		if s == v {
			rt.bigRaise("%s", "TypeError: Converting circular structure to JSON")
		}
	}
	st.stack = append(st.stack, v)
}

func (rt *jsrt) jsonArr(st *jsonState, a *jsArray) string {
	rt.jsonEnter(st, a)
	stepback := st.indent
	st.indent = stepback + st.gap
	parts := make([]string, 0, len(a.elems))
	for i := 0; i < len(a.elems); i++ {
		s, ok := rt.jsonSer(st, a, strconv.Itoa(i), a.elems[i])
		if !ok {
			s = "null"
		}
		parts = append(parts, s)
	}
	out := jsonWrap(st, parts, "[", "]", stepback)
	st.indent = stepback
	st.stack = st.stack[:len(st.stack)-1]
	return out
}

func (rt *jsrt) jsonObj(st *jsonState, o *jsObject) string {
	rt.jsonEnter(st, o)
	stepback := st.indent
	st.indent = stepback + st.gap
	keys := st.allow
	if keys == nil {
		keys = jsvOwnKeys(o)
	}
	colon := ":"
	if st.gap != "" {
		colon = ": "
	}
	parts := []string{}
	for _, k := range keys {
		if s, ok := rt.jsonSer(st, o, k, rt.jsvGetKey(o, k)); ok {
			parts = append(parts, jsonQuote(k)+colon+s)
		}
	}
	out := jsonWrap(st, parts, "{", "}", stepback)
	st.indent = stepback
	st.stack = st.stack[:len(st.stack)-1]
	return out
}

func jsonWrap(st *jsonState, parts []string, open, close, stepback string) string {
	if len(parts) == 0 {
		return open + close
	}
	if st.gap == "" {
		return open + strJoin(parts, ",") + close
	}
	return open + "\n" + st.indent + strJoin(parts, ",\n"+st.indent) + "\n" + stepback + close
}

// ----- JSON.parse -----

type jsonParser struct {
	rt *jsrt
	s  []uint16
	i  int
}

func (rt *jsrt) jsonParse(text, reviver interface{}) interface{} {
	p := &jsonParser{rt: rt, s: strUnits(rt.jsvString(text))}
	v := p.value()
	p.ws()
	if p.i < len(p.s) {
		p.bad()
	}
	if rt.jsonIsFn(reviver) {
		root := newJSObject()
		root.set("", v)
		return rt.jsonRevive(root, "", v, reviver)
	}
	return v
}

func (p *jsonParser) bad() {
	if p.i >= len(p.s) {
		p.rt.bigRaise("%s", "SyntaxError: Unexpected end of JSON input")
	}
	p.rt.bigRaise("SyntaxError: Unexpected token '%s' in JSON at position %d", strFromUnits(p.s[p.i:p.i+1]), p.i)
}

func (p *jsonParser) ws() {
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return
		}
		p.i++
	}
}

func (p *jsonParser) word(w string) {
	for j := 0; j < len(w); j++ {
		if p.i+j >= len(p.s) || p.s[p.i+j] != uint16(w[j]) {
			p.i += j
			p.bad()
		}
	}
	p.i += len(w)
}

func (p *jsonParser) value() interface{} {
	p.ws()
	if p.i >= len(p.s) {
		p.bad()
	}
	switch c := p.s[p.i]; {
	case c == '{':
		return p.object()
	case c == '[':
		return p.array()
	case c == '"':
		return p.str()
	case c == 't':
		p.word("true")
		return true
	case c == 'f':
		p.word("false")
		return false
	case c == 'n':
		p.word("null")
		return jsNull
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	}
	p.bad()
	return jsUndef
}

func (p *jsonParser) digits() {
	start := p.i
	for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
	}
	if p.i == start {
		p.bad()
	}
}

func (p *jsonParser) at(c uint16) bool { return p.i < len(p.s) && p.s[p.i] == c }

// '-'? ('0' | [1-9][0-9]*) ('.' [0-9]+)? ([eE] [+-]? [0-9]+)?
func (p *jsonParser) number() interface{} {
	start := p.i
	if p.at('-') {
		p.i++
	}
	if p.at('0') {
		p.i++
	} else {
		p.digits()
	}
	if p.at('.') {
		p.i++
		p.digits()
	}
	if p.at('e') || p.at('E') {
		p.i++
		if p.at('+') || p.at('-') {
			p.i++
		}
		p.digits()
	}
	f, _ := strconv.ParseFloat(strFromUnits(p.s[start:p.i]), 64)
	return f
}

func (p *jsonParser) str() string {
	p.i++
	out := []uint16{}
	for {
		if p.i >= len(p.s) {
			p.bad()
		}
		c := p.s[p.i]
		if c == '"' {
			break
		}
		if c < 32 {
			p.bad()
		}
		if c != '\\' {
			out = append(out, c)
			p.i++
			continue
		}
		if p.i+1 >= len(p.s) {
			p.i++
			p.bad()
		}
		e := p.s[p.i+1]
		p.i += 2
		switch e {
		case '"', '\\', '/':
			out = append(out, e)
		case 'b':
			out = append(out, 8)
		case 'f':
			out = append(out, 12)
		case 'n':
			out = append(out, 10)
		case 'r':
			out = append(out, 13)
		case 't':
			out = append(out, 9)
		case 'u':
			cu := uint16(0)
			for j := 0; j < 4; j++ {
				h := -1
				if p.i < len(p.s) {
					h = jsonHexVal(p.s[p.i])
				}
				if h < 0 {
					p.bad()
				}
				cu = cu*16 + uint16(h)
				p.i++
			}
			out = append(out, cu)
		default:
			p.i--
			p.bad()
		}
	}
	p.i++
	return strFromUnits(out)
}

func jsonHexVal(c uint16) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

func (p *jsonParser) array() interface{} {
	p.i++
	out := &jsArray{}
	p.ws()
	if p.at(']') {
		p.i++
		return out
	}
	for {
		out.elems = append(out.elems, p.value())
		p.ws()
		if p.i >= len(p.s) {
			p.bad()
		}
		c := p.s[p.i]
		p.i++
		if c == ']' {
			return out
		}
		if c != ',' {
			p.i--
			p.bad()
		}
	}
}

func (p *jsonParser) object() interface{} {
	p.i++
	o := newJSObject()
	p.ws()
	if p.at('}') {
		p.i++
		return o
	}
	for {
		p.ws()
		if !p.at('"') {
			p.bad()
		}
		k := p.str()
		p.ws()
		if !p.at(':') {
			p.bad()
		}
		p.i++
		o.set(k, p.value())
		p.ws()
		if p.i >= len(p.s) {
			p.bad()
		}
		c := p.s[p.i]
		p.i++
		if c == '}' {
			return o
		}
		if c != ',' {
			p.i--
			p.bad()
		}
	}
}

// jsonRevive is InternalizeJSONProperty: the reviver sees every value bottom-up with
// the holder as its receiver, and answering undefined deletes the property.
func (rt *jsrt) jsonRevive(holder interface{}, key string, val interface{}, reviver interface{}) interface{} {
	switch x := val.(type) {
	case *jsArray:
		for i := range x.elems {
			x.elems[i] = rt.jsonRevive(x, strconv.Itoa(i), x.elems[i], reviver)
		}
	case *jsObject:
		if jsonIsObj(x) {
			for _, k := range jsvOwnKeys(x) {
				nv := rt.jsonRevive(x, k, rt.jsvGetKey(x, k), reviver)
				if _, isU := nv.(jsUndefT); isU {
					delete(x.props, k)
					for i, kk := range x.keys {
						if kk == k {
							x.keys = append(x.keys[:i:i], x.keys[i+1:]...)
							break
						}
					}
				} else {
					x.set(k, nv)
				}
			}
		}
	}
	return rt.call(reviver, holder, []interface{}{key, val})
}
