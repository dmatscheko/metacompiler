package ebnf

import (
	"fmt"
	"strings"

	"./r"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/metadata"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

//
// See:
// https://pkg.go.dev/github.com/llir/llvm/
//

var llvmFuncMap = map[string]r.Object{ // The LLVM functions.
	// See https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/types
	"types": map[string]r.Object{
		// Funcs.
		"Equal":      types.Equal,
		"IsArray":    types.IsArray,
		"IsFloat":    types.IsFloat,
		"IsFunc":     types.IsFunc,
		"IsInt":      types.IsInt,
		"IsLabel":    types.IsLabel,
		"IsMMX":      types.IsMMX,
		"IsMetadata": types.IsMetadata,
		"IsPointer":  types.IsPointer,
		"IsStruct":   types.IsStruct,
		"IsToken":    types.IsToken,
		"IsVector":   types.IsVector,
		"IsVoid":     types.IsVoid,
		"NewArray":   types.NewArray,
		"NewFunc":    types.NewFunc,
		"NewInt":     types.NewInt,
		"NewPointer": types.NewPointer,
		"NewStruct":  types.NewStruct,
		"NewVector":  types.NewVector,
		// Basic types.
		"Void":     types.Void,
		"MMX":      types.MMX,
		"Label":    types.Label,
		"Token":    types.Token,
		"Metadata": types.Metadata,
		// Integer types.
		"I1":    types.I1,
		"I2":    types.I2,
		"I3":    types.I3,
		"I4":    types.I4,
		"I5":    types.I5,
		"I6":    types.I6,
		"I7":    types.I7,
		"I8":    types.I8,
		"I16":   types.I16,
		"I32":   types.I32,
		"I64":   types.I64,
		"I128":  types.I128,
		"I256":  types.I256,
		"I512":  types.I512,
		"I1024": types.I1024,
		// Floating-point types.
		"Half":      types.Half,
		"Float":     types.Float,
		"Double":    types.Double,
		"X86_FP80":  types.X86_FP80,
		"FP128":     types.FP128,
		"PPC_FP128": types.PPC_FP128,
		// Integer pointer types.
		"I1Ptr":   types.I1Ptr,
		"I8Ptr":   types.I8Ptr,
		"I16Ptr":  types.I16Ptr,
		"I32Ptr":  types.I32Ptr,
		"I64Ptr":  types.I64Ptr,
		"I128Ptr": types.I128Ptr,
	},

	// See https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/metadata
	"metadata": map[string]r.Object{
		"Null": metadata.Null,
	},

	// See https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/constant
	"constant": map[string]r.Object{
		// Consts.
		"None":  constant.None,
		"True":  constant.True,
		"False": constant.False,
		// Funcs.
		"NewArray":               constant.NewArray,
		"NewBlockAddress":        constant.NewBlockAddress,
		"NewCharArray":           constant.NewCharArray,
		"NewCharArrayFromString": constant.NewCharArrayFromString,
		"NewAShr":                constant.NewAShr,
		"NewAdd":                 constant.NewAdd,
		"NewAddrSpaceCast":       constant.NewAddrSpaceCast,
		"NewAnd":                 constant.NewAnd,
		"NewBitCast":             constant.NewBitCast,
		"NewExtractElement":      constant.NewExtractElement,
		"NewExtractValue":        constant.NewExtractValue,
		"NewFAdd":                constant.NewFAdd,
		"NewFCmp":                constant.NewFCmp,
		"NewFDiv":                constant.NewFDiv,
		"NewFMul":                constant.NewFMul,
		"NewFNeg":                constant.NewFNeg,
		"NewFPExt":               constant.NewFPExt,
		"NewFPToSI":              constant.NewFPToSI,
		"NewFPToUI":              constant.NewFPToUI,
		"NewFPTrunc":             constant.NewFPTrunc,
		"NewFRem":                constant.NewFRem,
		"NewFSub":                constant.NewFSub,
		"NewGetElementPtr":       constant.NewGetElementPtr,
		"NewICmp":                constant.NewICmp,
		"NewInsertElement":       constant.NewInsertElement,
		"NewInsertValue":         constant.NewInsertValue,
		"NewIntToPtr":            constant.NewIntToPtr,
		"NewLShr":                constant.NewLShr,
		"NewMul":                 constant.NewMul,
		"NewOr":                  constant.NewOr,
		"NewPtrToInt":            constant.NewPtrToInt,
		"NewSDiv":                constant.NewSDiv,
		"NewSExt":                constant.NewSExt,
		"NewSIToFP":              constant.NewSIToFP,
		"NewSRem":                constant.NewSRem,
		"NewSelect":              constant.NewSelect,
		"NewShl":                 constant.NewShl,
		"NewShuffleVector":       constant.NewShuffleVector,
		"NewSub":                 constant.NewSub,
		"NewTrunc":               constant.NewTrunc,
		"NewUDiv":                constant.NewUDiv,
		"NewUIToFP":              constant.NewUIToFP,
		"NewURem":                constant.NewURem,
		"NewXor":                 constant.NewXor,
		"NewZExt":                constant.NewZExt,
		"NewFloat":               constant.NewFloat,
		"NewFloatFromString":     constant.NewFloatFromString,
		"NewIndex":               constant.NewIndex,
		"NewBool":                constant.NewBool,
		"NewInt":                 constant.NewInt,
		"NewIntFromString":       constant.NewIntFromString,
		"NewNull":                constant.NewNull,
		"NewStruct":              constant.NewStruct,
		"NewUndef":               constant.NewUndef,
		"NewVector":              constant.NewVector,
		"NewZeroInitializer":     constant.NewZeroInitializer,
	},

	// See https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir
	"ir": map[string]r.Object{
		// Funcs.
		"NewAlias":          ir.NewAlias,
		"NewArg":            ir.NewArg,
		"NewBlock":          ir.NewBlock,
		"NewCase":           ir.NewCase,
		"NewClause":         ir.NewClause,
		"NewFunc":           ir.NewFunc,
		"NewGlobal":         ir.NewGlobal,
		"NewGlobalDef":      ir.NewGlobalDef,
		"NewIFunc":          ir.NewIFunc,
		"NewIncoming":       ir.NewIncoming,
		"NewInlineAsm":      ir.NewInlineAsm,
		"NewAShr":           ir.NewAShr,
		"NewAdd":            ir.NewAdd,
		"NewAddrSpaceCast":  ir.NewAddrSpaceCast,
		"NewAlloca":         ir.NewAlloca,
		"NewAnd":            ir.NewAnd,
		"NewAtomicRMW":      ir.NewAtomicRMW,
		"NewBitCast":        ir.NewBitCast,
		"NewCall":           ir.NewCall,
		"NewCatchPad":       ir.NewCatchPad,
		"NewCleanupPad":     ir.NewCleanupPad,
		"NewCmpXchg":        ir.NewCmpXchg,
		"NewExtractElement": ir.NewExtractElement,
		"NewExtractValue":   ir.NewExtractValue,
		"NewFAdd":           ir.NewFAdd,
		"NewFCmp":           ir.NewFCmp,
		"NewFDiv":           ir.NewFDiv,
		"NewFMul":           ir.NewFMul,
		"NewFNeg":           ir.NewFNeg,
		"NewFPExt":          ir.NewFPExt,
		"NewFPToSI":         ir.NewFPToSI,
		"NewFPToUI":         ir.NewFPToUI,
		"NewFPTrunc":        ir.NewFPTrunc,
		"NewFRem":           ir.NewFRem,
		"NewFSub":           ir.NewFSub,
		"NewFence":          ir.NewFence,
		"NewInstFreeze":     ir.NewInstFreeze,
		"NewGetElementPtr":  ir.NewGetElementPtr,
		"NewICmp":           ir.NewICmp,
		"NewInsertElement":  ir.NewInsertElement,
		"NewInsertValue":    ir.NewInsertValue,
		"NewIntToPtr":       ir.NewIntToPtr,
		"NewLShr":           ir.NewLShr,
		"NewLandingPad":     ir.NewLandingPad,
		"NewLoad":           ir.NewLoad,
		"NewMul":            ir.NewMul,
		"NewOr":             ir.NewOr,
		"NewPhi":            ir.NewPhi,
		"NewPtrToInt":       ir.NewPtrToInt,
		"NewSDiv":           ir.NewSDiv,
		"NewSExt":           ir.NewSExt,
		"NewSIToFP":         ir.NewSIToFP,
		"NewSRem":           ir.NewSRem,
		"NewSelect":         ir.NewSelect,
		"NewShl":            ir.NewShl,
		"NewShuffleVector":  ir.NewShuffleVector,
		"NewStore":          ir.NewStore,
		"NewSub":            ir.NewSub,
		"NewTrunc":          ir.NewTrunc,
		"NewUDiv":           ir.NewUDiv,
		"NewUIToFP":         ir.NewUIToFP,
		"NewURem":           ir.NewURem,
		"NewVAArg":          ir.NewVAArg,
		"NewXor":            ir.NewXor,
		"NewZExt":           ir.NewZExt,
		"NewLocalIdent":     ir.NewLocalIdent,
		"NewModule":         ir.NewModule,
		"NewOperandBundle":  ir.NewOperandBundle,
		"NewParam":          ir.NewParam,
		"NewBr":             ir.NewBr,
		"NewCallBr":         ir.NewCallBr,
		"NewCatchRet":       ir.NewCatchRet,
		"NewCatchSwitch":    ir.NewCatchSwitch,
		"NewCleanupRet":     ir.NewCleanupRet,
		"NewCondBr":         ir.NewCondBr,
		"NewIndirectBr":     ir.NewIndirectBr,
		"NewInvoke":         ir.NewInvoke,
		"NewResume":         ir.NewResume,
		"NewRet":            ir.NewRet,
		"NewSwitch":         ir.NewSwitch,
		"NewUnreachable":    ir.NewUnreachable,
	},
	"Callgraph": callgraph,
	"Eval":      eval,
}

// callgraph returns the callgraph in Graphviz DOT format of the given LLVM IR module.
// Code taken from: https://github.com/llir/llvm#analysis-example---process-llvm-ir
// DOT output is viewable online e.g. with: http://magjac.com/graphviz-visual-editor/
func callgraph(m *ir.Module) string {
	buf := &strings.Builder{}
	buf.WriteString("digraph {\n")
	// For each function of the module.
	for _, f := range m.Funcs {
		// Add caller node.
		caller := f.Ident()
		fmt.Fprintf(buf, "\t%q\n", caller)
		// For each basic block of the function.
		for _, block := range f.Blocks {
			// For each non-branching instruction of the basic block.
			for _, inst := range block.Insts {
				// Type switch on instruction to find call instructions.
				switch inst := inst.(type) {
				case *ir.InstCall:
					callee := inst.Callee.Ident()
					// Add edges from caller to callee.
					fmt.Fprintf(buf, "\t%q -> %q\n", caller, callee)
				}
			}
			// Terminator of basic block.
			switch term := block.Term.(type) {
			case *ir.TermRet:
				// do something.
				_ = term
			}
		}
	}
	buf.WriteString("}")
	return buf.String()
}

// ----------------------------------------------------------------------------
// IR function evaluator
// Code taken from https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir#example-package-Evaluator

func eval(m *ir.Module, start string) uint32 {
	// Evalute and print the return value of the given function.
	for _, f := range m.Funcs {
		if f.Name() == start {
			e := newEvaluator(f)
			return e.evalFunc()
		}
	}
	return 0
}

// evaluator is a function evaluator.
type evaluator struct {
	// Function being evaluated.
	f *ir.Func
	// Function arguments.
	args []value.Value
}

// newEvaluator returns a new function evaluator, for evaluating the result of
// invoking f with args.
func newEvaluator(f *ir.Func, args ...value.Value) *evaluator {
	return &evaluator{f: f, args: args}
}

// evalFunc evalutes f and returns the corresponding 32-bit integer.
func (e *evaluator) evalFunc() uint32 {
	f := e.f
	if !types.Equal(f.Sig.RetType, types.I32) {
		panic(fmt.Errorf("support for function return type %s not yet implemented", f.Sig.RetType))
	}
	for _, block := range f.Blocks {
		switch term := block.Term.(type) {
		case *ir.TermRet:
			// Note: support for functions with more than one ret terminator not
			// yet implemented.
			if term.X != nil {
				// The result of the first return value of a function is evaluated.
				return e.evalValue(term.X)
			}
		}
	}
	panic(fmt.Errorf("unable to locate ret terminator in function %q", f.Ident()))
}

// evalInst evaluates inst and returns the corresponding 32-bit integer.
func (e *evaluator) evalInst(inst ir.Instruction) uint32 {
	switch inst := inst.(type) {
	// Binary instructions.
	case *ir.InstAdd:
		return e.evalValue(inst.X) + e.evalValue(inst.Y)
	case *ir.InstSub:
		return e.evalValue(inst.X) - e.evalValue(inst.Y)
	case *ir.InstMul:
		return e.evalValue(inst.X) * e.evalValue(inst.Y)
	case *ir.InstUDiv:
		return e.evalValue(inst.X) / e.evalValue(inst.Y)
	case *ir.InstSDiv:
		return e.evalValue(inst.X) / e.evalValue(inst.Y)
	case *ir.InstURem:
		return e.evalValue(inst.X) % e.evalValue(inst.Y)
	case *ir.InstSRem:
		return e.evalValue(inst.X) % e.evalValue(inst.Y)
	// Bitwise instructions.
	case *ir.InstShl:
		return e.evalValue(inst.X) << e.evalValue(inst.Y)
	case *ir.InstLShr:
		return e.evalValue(inst.X) >> e.evalValue(inst.Y)
	case *ir.InstAShr:
		x, y := e.evalValue(inst.X), e.evalValue(inst.Y)
		result := x >> y
		// sign extend.
		if x&0x80000000 != 0 {
			result = signExt(result)
		}
		return result
	case *ir.InstAnd:
		return e.evalValue(inst.X) & e.evalValue(inst.Y)
	case *ir.InstOr:
		return e.evalValue(inst.X) | e.evalValue(inst.Y)
	case *ir.InstXor:
		return e.evalValue(inst.X) ^ e.evalValue(inst.Y)
	// Other instructions.
	case *ir.InstCall:
		callee, ok := inst.Callee.(*ir.Func)
		if !ok {
			panic(fmt.Errorf("support for callee type %T not yet implemented", inst.Callee))
		}
		ee := newEvaluator(callee, inst.Args...)
		return ee.evalFunc()
	default:
		panic(fmt.Errorf("support for instruction type %T not yet implemented", inst))
	}
}

// evalValue evalutes v and returns the corresponding 32-bit integer.
func (e *evaluator) evalValue(v value.Value) uint32 {
	switch v := v.(type) {
	case ir.Instruction:
		return e.evalInst(v)
	case *constant.Int:
		return uint32(v.X.Int64())
	case *ir.Param:
		f := e.f
		for i, param := range f.Params {
			if v.Ident() == param.Ident() {
				return e.evalValue(e.args[i])
			}
		}
		panic(fmt.Errorf("unable to locate paramater %q of function %q", v.Ident(), f.Ident()))
	default:
		panic(fmt.Errorf("support for value type %T not yet implemented", v))
	}
}

// signExt sign extends x.
func signExt(x uint32) uint32 {
	for i := uint32(31); i >= 0; i-- {
		mask := uint32(1 << i)
		if x&mask != 0 {
			break
		}
		x |= mask
	}
	return x
}
