package abnf

// A binary form of the compiled script modules for the frozen script cache.
//
// The cache used to keep each module as LLVM assembly TEXT and rebuild it with
// the llir parser. That parser is the slowest thing in a -frozen run that has a
// warm cache: one Kotlin run reloads 104 modules and spends 57 ms of its 253 ms
// in there, most of it on the two big shared includes of languages/lib (777 KB
// and 272 KB of .ll). The text is a general purpose interchange format; nothing
// here needs one, because the only consumer is the IR interpreter in llvmmap.go.
//
// So a module is written as the subset of the IR that the interpreter actually
// executes: types, constants, globals, functions, blocks, the ~30 instructions
// exec() understands and the four terminators. Everything is index based (one
// table for strings, one for types, dense numbering for values), which is why
// loading is a linear pass with no lexing, no name resolution and no
// backtracking.
//
// The format is a CACHE format, so it may refuse work: encodeModule returns an
// error for anything it does not know, and the caller then stores .ll as before.
// decodeModule likewise returns an error rather than guessing, and a failed
// decode just recompiles the script. scriptCacheFormat (scriptcache.go) is part
// of every cache key, so changing anything here invalidates old entries.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// scriptBinMagic starts every binary cache entry. Its last byte is the format
// version: an old entry then fails to decode and is recompiled.
var scriptBinMagic = []byte("MECIR\x01")

// Type kinds.
const (
	tkVoid = iota
	tkInt
	tkPointer
	tkArray
	tkStruct
	tkFunc
	tkLabel
)

// Value reference kinds.
const (
	vkConst = iota
	vkGlobal
	vkFunc
	vkLocal // A parameter or an instruction result of the current function.
	vkBlock
)

// Constant kinds.
const (
	ckInt = iota
	ckNull
	ckZero
	ckCharArray
	ckArray
	ckStruct
	ckGlobal
	ckIndex
	ckGEP
)

// Instruction opcodes. Only what machine.exec executes.
const (
	opAlloca = iota
	opLoad
	opStore
	opGEP
	opAdd
	opSub
	opMul
	opUDiv
	opSDiv
	opURem
	opSRem
	opAnd
	opOr
	opXor
	opShl
	opLShr
	opAShr
	opICmp
	opZExt
	opSExt
	opTrunc
	opIntToPtr
	opPtrToInt
	opBitCast
	opSelect
	opPhi
	opCall
)

// Terminator opcodes.
const (
	tmBr = iota
	tmCondBr
	tmRet
	tmUnreachable
)

var errUnsupportedIR = errors.New("script cache: the module uses IR this codec does not cover")

// ----------------------------------------------------------------------------
// Encoding

type irEncoder struct {
	buf      []byte
	strIdx   map[string]int
	typeIdx  map[types.Type]int
	globIdx  map[*ir.Global]int
	funcIdx  map[*ir.Func]int
	localIdx map[value.Value]int // Per function: parameters, then instructions.
	blockIdx map[*ir.Block]int   // Per function.
	strs     []string
	types    [][]byte // Already encoded type entries, in dependency order.
}

func (e *irEncoder) uvar(v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	e.buf = append(e.buf, tmp[:binary.PutUvarint(tmp[:], v)]...)
}

func (e *irEncoder) i64(v int64) {
	var tmp [binary.MaxVarintLen64]byte
	e.buf = append(e.buf, tmp[:binary.PutVarint(tmp[:], v)]...)
}

func (e *irEncoder) bytes(b []byte) {
	e.uvar(uint64(len(b)))
	e.buf = append(e.buf, b...)
}

// str interns a string and writes its index.
func (e *irEncoder) str(s string) {
	i, ok := e.strIdx[s]
	if !ok {
		i = len(e.strs)
		e.strs = append(e.strs, s)
		e.strIdx[s] = i
	}
	e.uvar(uint64(i))
}

// typ interns a type (children first) and writes its index.
func (e *irEncoder) typ(t types.Type) error {
	if _, ok := e.typeIdx[t]; ok {
		e.uvar(uint64(e.typeIdx[t]))
		return nil
	}
	// Encode the entry into a scratch encoder that shares the tables, so a
	// nested type ends up in front of the one that uses it.
	sub := &irEncoder{strIdx: e.strIdx, typeIdx: e.typeIdx, strs: e.strs, types: e.types}
	if err := sub.typeEntry(t); err != nil {
		return err
	}
	e.strs, e.types, e.typeIdx = sub.strs, sub.types, sub.typeIdx
	idx := len(e.types)
	e.types = append(e.types, sub.buf)
	e.typeIdx[t] = idx
	e.uvar(uint64(idx))
	return nil
}

func (e *irEncoder) typeEntry(t types.Type) error {
	switch t := t.(type) {
	case *types.VoidType:
		e.buf = append(e.buf, tkVoid)
	case *types.IntType:
		e.buf = append(e.buf, tkInt)
		e.uvar(t.BitSize)
	case *types.PointerType:
		e.buf = append(e.buf, tkPointer)
		return e.typ(t.ElemType)
	case *types.ArrayType:
		e.buf = append(e.buf, tkArray)
		e.uvar(t.Len)
		return e.typ(t.ElemType)
	case *types.StructType:
		e.buf = append(e.buf, tkStruct)
		e.str(t.Name())
		if t.Packed {
			e.buf = append(e.buf, 1)
		} else {
			e.buf = append(e.buf, 0)
		}
		e.uvar(uint64(len(t.Fields)))
		for _, f := range t.Fields {
			if err := e.typ(f); err != nil {
				return err
			}
		}
	case *types.FuncType:
		e.buf = append(e.buf, tkFunc)
		if t.Variadic {
			e.buf = append(e.buf, 1)
		} else {
			e.buf = append(e.buf, 0)
		}
		if err := e.typ(t.RetType); err != nil {
			return err
		}
		e.uvar(uint64(len(t.Params)))
		for _, p := range t.Params {
			if err := e.typ(p); err != nil {
				return err
			}
		}
	case *types.LabelType:
		e.buf = append(e.buf, tkLabel)
	default:
		return fmt.Errorf("%w: type %T", errUnsupportedIR, t)
	}
	return nil
}

// val writes an operand reference.
func (e *irEncoder) val(v value.Value) error {
	switch v := v.(type) {
	case *ir.Global:
		if i, ok := e.globIdx[v]; ok {
			e.buf = append(e.buf, vkGlobal)
			e.uvar(uint64(i))
			return nil
		}
		return fmt.Errorf("%w: unknown global %s", errUnsupportedIR, v.Ident())
	case *ir.Func:
		if i, ok := e.funcIdx[v]; ok {
			e.buf = append(e.buf, vkFunc)
			e.uvar(uint64(i))
			return nil
		}
		return fmt.Errorf("%w: unknown function %s", errUnsupportedIR, v.Ident())
	case *ir.Block:
		if i, ok := e.blockIdx[v]; ok {
			e.buf = append(e.buf, vkBlock)
			e.uvar(uint64(i))
			return nil
		}
		return fmt.Errorf("%w: unknown block", errUnsupportedIR)
	case constant.Constant:
		e.buf = append(e.buf, vkConst)
		return e.constant(v)
	default:
		if i, ok := e.localIdx[v]; ok {
			e.buf = append(e.buf, vkLocal)
			e.uvar(uint64(i))
			return nil
		}
		return fmt.Errorf("%w: value %T", errUnsupportedIR, v)
	}
}

func (e *irEncoder) constant(c constant.Constant) error {
	switch c := c.(type) {
	case *constant.Int:
		e.buf = append(e.buf, ckInt)
		if err := e.typ(c.Typ); err != nil {
			return err
		}
		if c.X.Sign() < 0 {
			e.buf = append(e.buf, 1)
		} else {
			e.buf = append(e.buf, 0)
		}
		e.bytes(c.X.Bytes())
	case *constant.Null:
		e.buf = append(e.buf, ckNull)
		return e.typ(c.Typ)
	case *constant.ZeroInitializer:
		e.buf = append(e.buf, ckZero)
		return e.typ(c.Typ)
	case *constant.CharArray:
		e.buf = append(e.buf, ckCharArray)
		if err := e.typ(c.Typ); err != nil {
			return err
		}
		e.bytes(c.X)
	case *constant.Array:
		e.buf = append(e.buf, ckArray)
		if err := e.typ(c.Typ); err != nil {
			return err
		}
		e.uvar(uint64(len(c.Elems)))
		for _, elem := range c.Elems {
			if err := e.constant(elem); err != nil {
				return err
			}
		}
	case *constant.Struct:
		e.buf = append(e.buf, ckStruct)
		if err := e.typ(c.Typ); err != nil {
			return err
		}
		e.uvar(uint64(len(c.Fields)))
		for _, f := range c.Fields {
			if err := e.constant(f); err != nil {
				return err
			}
		}
	case *ir.Global:
		i, ok := e.globIdx[c]
		if !ok {
			return fmt.Errorf("%w: unknown global %s", errUnsupportedIR, c.Ident())
		}
		e.buf = append(e.buf, ckGlobal)
		e.uvar(uint64(i))
	case *constant.Index:
		e.buf = append(e.buf, ckIndex)
		return e.constant(c.Constant)
	case *constant.ExprGetElementPtr:
		e.buf = append(e.buf, ckGEP)
		if err := e.typ(c.ElemType); err != nil {
			return err
		}
		if err := e.constant(c.Src); err != nil {
			return err
		}
		e.uvar(uint64(len(c.Indices)))
		for _, idx := range c.Indices {
			if err := e.constant(idx); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: constant %T", errUnsupportedIR, c)
	}
	return nil
}

// ident writes the local name / id of a value, so idents in error messages stay
// what they were.
func (e *irEncoder) ident(id ir.LocalIdent) {
	e.str(id.LocalName)
	e.i64(id.LocalID)
}

// encodeModule renders a module in the binary cache format. An error means the
// module uses something the codec does not cover; the caller falls back to .ll.
func encodeModule(m *ir.Module) (res []byte, err error) {
	defer func() { // llir panics on some malformed input; a cache write must not.
		if p := recover(); p != nil {
			res, err = nil, fmt.Errorf("%w: %v", errUnsupportedIR, p)
		}
	}()

	e := &irEncoder{
		strIdx:  map[string]int{},
		typeIdx: map[types.Type]int{},
		globIdx: map[*ir.Global]int{},
		funcIdx: map[*ir.Func]int{},
	}
	for i, g := range m.Globals {
		e.globIdx[g] = i
	}
	for i, f := range m.Funcs {
		e.funcIdx[f] = i
	}

	// Globals.
	e.uvar(uint64(len(m.Globals)))
	for _, g := range m.Globals {
		e.str(g.GlobalName)
		if err := e.typ(g.ContentType); err != nil {
			return nil, err
		}
		if g.Init == nil {
			e.buf = append(e.buf, 0)
		} else {
			e.buf = append(e.buf, 1)
			if err := e.constant(g.Init); err != nil {
				return nil, err
			}
		}
	}

	// Functions: every signature first, then the bodies. A body may call a
	// function that is declared further down, so the decoder needs them all
	// before it reads the first one.
	e.uvar(uint64(len(m.Funcs)))
	for _, f := range m.Funcs {
		e.str(f.GlobalName)
		if err := e.typ(f.Sig.RetType); err != nil {
			return nil, err
		}
		if f.Sig.Variadic {
			e.buf = append(e.buf, 1)
		} else {
			e.buf = append(e.buf, 0)
		}
		e.uvar(uint64(len(f.Params)))
		for _, p := range f.Params {
			e.ident(p.LocalIdent)
			if err := e.typ(p.Typ); err != nil {
				return nil, err
			}
		}
	}
	for _, f := range m.Funcs {
		if err := e.function(f); err != nil {
			return nil, err
		}
	}

	// The tables go in front, so the decoder has them before it needs them.
	head := &irEncoder{}
	head.buf = append(head.buf, scriptBinMagic...)
	head.uvar(uint64(len(e.strs)))
	for _, s := range e.strs {
		head.bytes([]byte(s))
	}
	head.uvar(uint64(len(e.types)))
	for _, t := range e.types {
		head.bytes(t)
	}
	return append(head.buf, e.buf...), nil
}

// function encodes the body of one function: the blocks with their instructions
// and terminators. Values are numbered parameters first, then the instructions
// of the blocks in order - the same dense numbering the interpreter's frames use.
func (e *irEncoder) function(f *ir.Func) error {
	e.localIdx = map[value.Value]int{}
	e.blockIdx = map[*ir.Block]int{}
	n := 0
	for _, p := range f.Params {
		e.localIdx[p] = n
		n++
	}
	for i, b := range f.Blocks {
		e.blockIdx[b] = i
		for _, inst := range b.Insts {
			if v, ok := inst.(value.Value); ok {
				e.localIdx[v] = n
			}
			n++
		}
	}

	e.uvar(uint64(len(f.Blocks)))
	for _, b := range f.Blocks {
		e.ident(b.LocalIdent)
		e.uvar(uint64(len(b.Insts)))
		for _, inst := range b.Insts {
			if err := e.inst(inst); err != nil {
				return err
			}
		}
		if err := e.term(b.Term); err != nil {
			return err
		}
	}
	return nil
}

func (e *irEncoder) binary(op byte, id ir.LocalIdent, x, y value.Value) error {
	e.buf = append(e.buf, op)
	e.ident(id)
	if err := e.val(x); err != nil {
		return err
	}
	return e.val(y)
}

func (e *irEncoder) cast(op byte, id ir.LocalIdent, from value.Value, to types.Type) error {
	e.buf = append(e.buf, op)
	e.ident(id)
	if err := e.val(from); err != nil {
		return err
	}
	return e.typ(to)
}

func (e *irEncoder) inst(inst ir.Instruction) error {
	switch inst := inst.(type) {
	case *ir.InstAlloca:
		e.buf = append(e.buf, opAlloca)
		e.ident(inst.LocalIdent)
		if err := e.typ(inst.ElemType); err != nil {
			return err
		}
		if inst.NElems == nil {
			e.buf = append(e.buf, 0)
			return nil
		}
		e.buf = append(e.buf, 1)
		return e.val(inst.NElems)
	case *ir.InstLoad:
		e.buf = append(e.buf, opLoad)
		e.ident(inst.LocalIdent)
		if err := e.typ(inst.ElemType); err != nil {
			return err
		}
		return e.val(inst.Src)
	case *ir.InstStore:
		e.buf = append(e.buf, opStore)
		if err := e.val(inst.Src); err != nil {
			return err
		}
		return e.val(inst.Dst)
	case *ir.InstGetElementPtr:
		e.buf = append(e.buf, opGEP)
		e.ident(inst.LocalIdent)
		if err := e.typ(inst.ElemType); err != nil {
			return err
		}
		if err := e.val(inst.Src); err != nil {
			return err
		}
		e.uvar(uint64(len(inst.Indices)))
		for _, idx := range inst.Indices {
			if err := e.val(idx); err != nil {
				return err
			}
		}
	case *ir.InstAdd:
		return e.binary(opAdd, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstSub:
		return e.binary(opSub, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstMul:
		return e.binary(opMul, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstUDiv:
		return e.binary(opUDiv, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstSDiv:
		return e.binary(opSDiv, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstURem:
		return e.binary(opURem, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstSRem:
		return e.binary(opSRem, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstAnd:
		return e.binary(opAnd, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstOr:
		return e.binary(opOr, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstXor:
		return e.binary(opXor, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstShl:
		return e.binary(opShl, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstLShr:
		return e.binary(opLShr, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstAShr:
		return e.binary(opAShr, inst.LocalIdent, inst.X, inst.Y)
	case *ir.InstICmp:
		e.buf = append(e.buf, opICmp)
		e.ident(inst.LocalIdent)
		e.uvar(uint64(inst.Pred))
		if err := e.val(inst.X); err != nil {
			return err
		}
		return e.val(inst.Y)
	case *ir.InstZExt:
		return e.cast(opZExt, inst.LocalIdent, inst.From, inst.To)
	case *ir.InstSExt:
		return e.cast(opSExt, inst.LocalIdent, inst.From, inst.To)
	case *ir.InstTrunc:
		return e.cast(opTrunc, inst.LocalIdent, inst.From, inst.To)
	case *ir.InstIntToPtr:
		return e.cast(opIntToPtr, inst.LocalIdent, inst.From, inst.To)
	case *ir.InstPtrToInt:
		return e.cast(opPtrToInt, inst.LocalIdent, inst.From, inst.To)
	case *ir.InstBitCast:
		return e.cast(opBitCast, inst.LocalIdent, inst.From, inst.To)
	case *ir.InstSelect:
		e.buf = append(e.buf, opSelect)
		e.ident(inst.LocalIdent)
		if err := e.val(inst.Cond); err != nil {
			return err
		}
		if err := e.val(inst.ValueTrue); err != nil {
			return err
		}
		return e.val(inst.ValueFalse)
	case *ir.InstPhi:
		e.buf = append(e.buf, opPhi)
		e.ident(inst.LocalIdent)
		if err := e.typ(inst.Type()); err != nil {
			return err
		}
		e.uvar(uint64(len(inst.Incs)))
		for _, inc := range inst.Incs {
			if err := e.val(inc.X); err != nil {
				return err
			}
			if err := e.val(inc.Pred); err != nil {
				return err
			}
		}
	case *ir.InstCall:
		callee, ok := inst.Callee.(*ir.Func)
		if !ok {
			return fmt.Errorf("%w: callee %T", errUnsupportedIR, inst.Callee)
		}
		e.buf = append(e.buf, opCall)
		e.ident(inst.LocalIdent)
		if err := e.val(callee); err != nil {
			return err
		}
		e.uvar(uint64(len(inst.Args)))
		for _, a := range inst.Args {
			if err := e.val(a); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: instruction %T", errUnsupportedIR, inst)
	}
	return nil
}

func (e *irEncoder) term(t ir.Terminator) error {
	switch t := t.(type) {
	case *ir.TermBr:
		e.buf = append(e.buf, tmBr)
		return e.val(t.Target)
	case *ir.TermCondBr:
		e.buf = append(e.buf, tmCondBr)
		if err := e.val(t.Cond); err != nil {
			return err
		}
		if err := e.val(t.TargetTrue); err != nil {
			return err
		}
		return e.val(t.TargetFalse)
	case *ir.TermRet:
		e.buf = append(e.buf, tmRet)
		if t.X == nil {
			e.buf = append(e.buf, 0)
			return nil
		}
		e.buf = append(e.buf, 1)
		return e.val(t.X)
	case *ir.TermUnreachable:
		e.buf = append(e.buf, tmUnreachable)
	default:
		return fmt.Errorf("%w: terminator %T", errUnsupportedIR, t)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Decoding

type irDecoder struct {
	buf    []byte
	pos    int
	strs   []string
	types  []types.Type
	mod    *ir.Module
	locals []value.Value // Per function, in the encoder's numbering.
	blocks []*ir.Block   // Per function.
	fixups []func()      // Operand bindings, run once the function is complete.
}

// valRef is an operand as it comes off the stream: a constant is complete, a
// local, block, global or function is an index that resolve() looks up.
type valRef struct {
	kind byte
	idx  int
	c    constant.Constant
}

var errCorruptCache = errors.New("script cache: corrupt entry")

func (d *irDecoder) uvar() uint64 {
	v, n := binary.Uvarint(d.buf[d.pos:])
	if n <= 0 {
		panic(errCorruptCache)
	}
	d.pos += n
	return v
}

func (d *irDecoder) i64() int64 {
	v, n := binary.Varint(d.buf[d.pos:])
	if n <= 0 {
		panic(errCorruptCache)
	}
	d.pos += n
	return v
}

func (d *irDecoder) byte() byte {
	if d.pos >= len(d.buf) {
		panic(errCorruptCache)

	}
	b := d.buf[d.pos]
	d.pos++
	return b
}

func (d *irDecoder) bytes() []byte {
	n := int(d.uvar())
	if d.pos+n > len(d.buf) {
		panic(errCorruptCache)
	}
	b := d.buf[d.pos : d.pos+n]
	d.pos += n
	return b
}

func (d *irDecoder) str() string {
	i := int(d.uvar())
	if i >= len(d.strs) {
		panic(errCorruptCache)
	}
	return d.strs[i]
}

func (d *irDecoder) typ() types.Type {
	i := int(d.uvar())
	if i >= len(d.types) {
		panic(errCorruptCache)
	}
	return d.types[i]
}

func (d *irDecoder) ident() ir.LocalIdent {
	return ir.LocalIdent{LocalName: d.str(), LocalID: d.i64()}
}

// decodeModule rebuilds a module from the binary cache format.
func decodeModule(data []byte) (m *ir.Module, err error) {
	defer func() {
		if p := recover(); p != nil {
			m, err = nil, fmt.Errorf("%w: %v", errCorruptCache, p)
		}
	}()
	if len(data) < len(scriptBinMagic) || string(data[:len(scriptBinMagic)]) != string(scriptBinMagic) {
		return nil, errCorruptCache
	}
	d := &irDecoder{buf: data, pos: len(scriptBinMagic)}

	n := int(d.uvar())
	d.strs = make([]string, n)
	for i := range d.strs {
		d.strs[i] = string(d.bytes())
	}
	n = int(d.uvar())
	d.types = make([]types.Type, 0, n)
	for i := 0; i < n; i++ {
		d.types = append(d.types, d.typeEntry(d.bytes()))
	}

	d.mod = ir.NewModule()

	nglobals := int(d.uvar())
	for i := 0; i < nglobals; i++ {
		name := d.str()
		contentType := d.typ()
		g := d.mod.NewGlobal(name, contentType)
		if d.byte() == 1 {
			g.Init = d.constant()
		}
	}

	nfuncs := int(d.uvar())
	for i := 0; i < nfuncs; i++ {
		name := d.str()
		retType := d.typ()
		variadic := d.byte() == 1
		params := make([]*ir.Param, int(d.uvar()))
		for j := range params {
			id := d.ident()
			p := ir.NewParam("", d.typ())
			p.LocalIdent = id
			params[j] = p
		}
		f := d.mod.NewFunc(name, retType, params...)
		f.Sig.Variadic = variadic
	}
	for _, f := range d.mod.Funcs {
		d.function(f)
	}
	return d.mod, nil
}

// function reads the body of one function. Operands are read as references and
// bound afterwards: a phi (and a branch) may name a value or a block that comes
// further down, so nothing can be resolved on the spot.
func (d *irDecoder) function(f *ir.Func) {
	d.locals = make([]value.Value, 0, len(f.Params)+32)
	for _, p := range f.Params {
		d.locals = append(d.locals, p)
	}
	d.fixups = d.fixups[:0]

	nblocks := int(d.uvar())
	d.blocks = make([]*ir.Block, nblocks)
	type blockRec struct {
		id     ir.LocalIdent
		ninsts int
	}
	recs := make([]blockRec, nblocks)
	// The blocks exist before any instruction, so a branch target resolves.
	for i := range d.blocks {
		d.blocks[i] = &ir.Block{}
	}
	for i := 0; i < nblocks; i++ {
		b := d.blocks[i]
		b.LocalIdent = d.ident()
		recs[i].ninsts = int(d.uvar())
		for j := 0; j < recs[i].ninsts; j++ {
			inst := d.inst()
			b.Insts = append(b.Insts, inst)
			if v, ok := inst.(value.Value); ok {
				d.locals = append(d.locals, v)
			} else {
				d.locals = append(d.locals, nil) // A store still takes a slot.
			}
		}
		b.Term = d.term()
		b.Parent = f
	}
	for _, fix := range d.fixups {
		fix()
	}
	f.Blocks = d.blocks
}

// fix defers binding an operand until the whole function has been read.
func (d *irDecoder) fix(bind func()) {
	d.fixups = append(d.fixups, bind)
}

// ref reads an operand reference; resolve turns it into the value later.
func (d *irDecoder) ref() valRef {
	k := d.byte()
	if k == vkConst {
		return valRef{kind: k, c: d.constant()}
	}
	return valRef{kind: k, idx: int(d.uvar())}
}

func (d *irDecoder) resolve(r valRef) value.Value {
	switch r.kind {
	case vkConst:
		return r.c
	case vkGlobal:
		if r.idx >= len(d.mod.Globals) {
			panic(errCorruptCache)
		}
		return d.mod.Globals[r.idx]
	case vkFunc:
		if r.idx >= len(d.mod.Funcs) {
			panic(errCorruptCache)
		}
		return d.mod.Funcs[r.idx]
	case vkLocal:
		if r.idx >= len(d.locals) || d.locals[r.idx] == nil {
			panic(errCorruptCache)
		}
		return d.locals[r.idx]
	case vkBlock:
		if r.idx >= len(d.blocks) {
			panic(errCorruptCache)
		}
		return d.blocks[r.idx]
	}
	panic(errCorruptCache)
}

func (d *irDecoder) block(r valRef) *ir.Block {
	b, ok := d.resolve(r).(*ir.Block)
	if !ok {
		panic(errCorruptCache)
	}
	return b
}

func (d *irDecoder) inst() ir.Instruction {
	switch d.byte() {
	case opAlloca:
		inst := &ir.InstAlloca{}
		inst.LocalIdent = d.ident()
		inst.ElemType = d.typ()
		if d.byte() == 1 {
			r := d.ref()
			d.fix(func() { inst.NElems = d.resolve(r) })
		}
		return inst
	case opLoad:
		inst := &ir.InstLoad{}
		inst.LocalIdent = d.ident()
		inst.ElemType = d.typ()
		r := d.ref()
		d.fix(func() { inst.Src = d.resolve(r) })
		return inst
	case opStore:
		inst := &ir.InstStore{}
		src, dst := d.ref(), d.ref()
		d.fix(func() { inst.Src, inst.Dst = d.resolve(src), d.resolve(dst) })
		return inst
	case opGEP:
		inst := &ir.InstGetElementPtr{}
		inst.LocalIdent = d.ident()
		inst.ElemType = d.typ()
		src := d.ref()
		refs := make([]valRef, int(d.uvar()))
		for i := range refs {
			refs[i] = d.ref()
		}
		d.fix(func() {
			inst.Src = d.resolve(src)
			inst.Indices = make([]value.Value, len(refs))
			for i, r := range refs {
				inst.Indices[i] = d.resolve(r)
			}
			inst.Typ = types.NewPointer(inst.ElemType)
		})
		return inst
	case opAdd:
		inst := &ir.InstAdd{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opSub:
		inst := &ir.InstSub{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opMul:
		inst := &ir.InstMul{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opUDiv:
		inst := &ir.InstUDiv{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opSDiv:
		inst := &ir.InstSDiv{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opURem:
		inst := &ir.InstURem{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opSRem:
		inst := &ir.InstSRem{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opAnd:
		inst := &ir.InstAnd{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opOr:
		inst := &ir.InstOr{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opXor:
		inst := &ir.InstXor{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opShl:
		inst := &ir.InstShl{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opLShr:
		inst := &ir.InstLShr{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opAShr:
		inst := &ir.InstAShr{}
		d.binary(&inst.LocalIdent, func(x, y value.Value) { inst.X, inst.Y = x, y })
		return inst
	case opICmp:
		inst := &ir.InstICmp{}
		inst.LocalIdent = d.ident()
		inst.Pred = enum.IPred(d.uvar())
		x, y := d.ref(), d.ref()
		d.fix(func() { inst.X, inst.Y = d.resolve(x), d.resolve(y) })
		return inst
	case opZExt:
		inst := &ir.InstZExt{}
		inst.LocalIdent = d.ident()
		from := d.ref()
		inst.To = d.typ()
		d.fix(func() { inst.From = d.resolve(from) })
		return inst
	case opSExt:
		inst := &ir.InstSExt{}
		inst.LocalIdent = d.ident()
		from := d.ref()
		inst.To = d.typ()
		d.fix(func() { inst.From = d.resolve(from) })
		return inst
	case opTrunc:
		inst := &ir.InstTrunc{}
		inst.LocalIdent = d.ident()
		from := d.ref()
		inst.To = d.typ()
		d.fix(func() { inst.From = d.resolve(from) })
		return inst
	case opIntToPtr:
		inst := &ir.InstIntToPtr{}
		inst.LocalIdent = d.ident()
		from := d.ref()
		inst.To = d.typ()
		d.fix(func() { inst.From = d.resolve(from) })
		return inst
	case opPtrToInt:
		inst := &ir.InstPtrToInt{}
		inst.LocalIdent = d.ident()
		from := d.ref()
		inst.To = d.typ()
		d.fix(func() { inst.From = d.resolve(from) })
		return inst
	case opBitCast:
		inst := &ir.InstBitCast{}
		inst.LocalIdent = d.ident()
		from := d.ref()
		inst.To = d.typ()
		d.fix(func() { inst.From = d.resolve(from) })
		return inst
	case opSelect:
		inst := &ir.InstSelect{}
		inst.LocalIdent = d.ident()
		cond, vt, vf := d.ref(), d.ref(), d.ref()
		d.fix(func() {
			inst.Cond, inst.ValueTrue, inst.ValueFalse = d.resolve(cond), d.resolve(vt), d.resolve(vf)
		})
		return inst
	case opPhi:
		inst := &ir.InstPhi{}
		inst.LocalIdent = d.ident()
		inst.Typ = d.typ()
		n := int(d.uvar())
		xs := make([]valRef, n)
		preds := make([]valRef, n)
		for i := 0; i < n; i++ {
			xs[i], preds[i] = d.ref(), d.ref()
		}
		d.fix(func() {
			inst.Incs = make([]*ir.Incoming, n)
			for i := range xs {
				inst.Incs[i] = &ir.Incoming{X: d.resolve(xs[i]), Pred: d.block(preds[i])}
			}
		})
		return inst
	case opCall:
		inst := &ir.InstCall{}
		inst.LocalIdent = d.ident()
		callee := d.ref()
		refs := make([]valRef, int(d.uvar()))
		for i := range refs {
			refs[i] = d.ref()
		}
		d.fix(func() {
			f, ok := d.resolve(callee).(*ir.Func)
			if !ok {
				panic(errCorruptCache)
			}
			inst.Callee = f
			inst.Typ = f.Sig.RetType
			inst.Args = make([]value.Value, len(refs))
			for i, r := range refs {
				inst.Args[i] = d.resolve(r)
			}
		})
		return inst
	}
	panic(errCorruptCache)
}

// binary reads the shared shape of the integer binary instructions.
func (d *irDecoder) binary(id *ir.LocalIdent, bind func(x, y value.Value)) {
	*id = d.ident()
	x, y := d.ref(), d.ref()
	d.fix(func() { bind(d.resolve(x), d.resolve(y)) })
}

func (d *irDecoder) term() ir.Terminator {
	switch d.byte() {
	case tmBr:
		t := &ir.TermBr{}
		r := d.ref()
		d.fix(func() { t.Target = d.block(r) })
		return t
	case tmCondBr:
		t := &ir.TermCondBr{}
		cond, tt, tf := d.ref(), d.ref(), d.ref()
		d.fix(func() {
			t.Cond = d.resolve(cond)
			t.TargetTrue, t.TargetFalse = d.block(tt), d.block(tf)
		})
		return t
	case tmRet:
		t := &ir.TermRet{}
		if d.byte() == 1 {
			r := d.ref()
			d.fix(func() { t.X = d.resolve(r) })
		}
		return t
	case tmUnreachable:
		return &ir.TermUnreachable{}
	}
	panic(errCorruptCache)
}

func (d *irDecoder) typeEntry(entry []byte) types.Type {
	sub := &irDecoder{buf: entry, strs: d.strs, types: d.types}
	switch sub.byte() {
	case tkVoid:
		return types.Void
	case tkInt:
		return types.NewInt(sub.uvar())
	case tkPointer:
		return types.NewPointer(sub.typ())
	case tkArray:
		length := sub.uvar()
		return types.NewArray(length, sub.typ())
	case tkStruct:
		name := sub.str()
		packed := sub.byte() == 1
		fields := make([]types.Type, int(sub.uvar()))
		for i := range fields {
			fields[i] = sub.typ()
		}
		t := types.NewStruct(fields...)
		t.TypeName = name
		t.Packed = packed
		return t
	case tkFunc:
		variadic := sub.byte() == 1
		ret := sub.typ()
		params := make([]types.Type, int(sub.uvar()))
		for i := range params {
			params[i] = sub.typ()
		}
		t := types.NewFunc(ret, params...)
		t.Variadic = variadic
		return t
	case tkLabel:
		return types.Label
	}
	panic(errCorruptCache)
}

func (d *irDecoder) constant() constant.Constant {
	switch d.byte() {
	case ckInt:
		t, ok := d.typ().(*types.IntType)
		if !ok {
			panic(errCorruptCache)
		}
		neg := d.byte() == 1
		x := new(big.Int).SetBytes(d.bytes())
		if neg {
			x.Neg(x)
		}
		return &constant.Int{Typ: t, X: x}
	case ckNull:
		t, ok := d.typ().(*types.PointerType)
		if !ok {
			panic(errCorruptCache)
		}
		return constant.NewNull(t)
	case ckZero:
		return constant.NewZeroInitializer(d.typ())
	case ckCharArray:
		t, ok := d.typ().(*types.ArrayType)
		if !ok {
			panic(errCorruptCache)
		}
		return &constant.CharArray{Typ: t, X: append([]byte{}, d.bytes()...)}
	case ckArray:
		t, ok := d.typ().(*types.ArrayType)
		if !ok {
			panic(errCorruptCache)
		}
		elems := make([]constant.Constant, int(d.uvar()))
		for i := range elems {
			elems[i] = d.constant()
		}
		return &constant.Array{Typ: t, Elems: elems}
	case ckStruct:
		t, ok := d.typ().(*types.StructType)
		if !ok {
			panic(errCorruptCache)
		}
		fields := make([]constant.Constant, int(d.uvar()))
		for i := range fields {
			fields[i] = d.constant()
		}
		return &constant.Struct{Typ: t, Fields: fields}
	case ckGlobal:
		i := int(d.uvar())
		if i >= len(d.mod.Globals) {
			panic(errCorruptCache)
		}
		return d.mod.Globals[i]
	case ckIndex:
		return &constant.Index{Constant: d.constant()}
	case ckGEP:
		elemType := d.typ()
		src := d.constant()
		indices := make([]constant.Constant, int(d.uvar()))
		for i := range indices {
			indices[i] = d.constant()
		}
		return constant.NewGetElementPtr(elemType, src, indices...)
	}
	panic(errCorruptCache)
}
