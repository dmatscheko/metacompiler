package abnf

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"14.gy/mec/abnf/r"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/metadata"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// ----------------------------------------------------------------------------
// Scripting subsystem mapping for LLVM IR

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
		// Wrapped so the call graph can attribute each function to its own
		// source file: record the module being built (see beginCompileModule).
		"NewModule": func() *ir.Module {
			m := ir.NewModule()
			beginCompileModule(m)
			return m
		},
		"NewOperandBundle": ir.NewOperandBundle,
		"NewParam":         ir.NewParam,
		"NewBr":            ir.NewBr,
		"NewCallBr":        ir.NewCallBr,
		"NewCatchRet":      ir.NewCatchRet,
		"NewCatchSwitch":   ir.NewCatchSwitch,
		"NewCleanupRet":    ir.NewCleanupRet,
		"NewCondBr":        ir.NewCondBr,
		"NewIndirectBr":    ir.NewIndirectBr,
		"NewInvoke":        ir.NewInvoke,
		"NewResume":        ir.NewResume,
		"NewRet":           ir.NewRet,
		"NewSwitch":        ir.NewSwitch,
		"NewUnreachable":   ir.NewUnreachable,
	},

	// See https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/enum
	"enum": map[string]r.Object{
		// AtomicOp is an AtomicRMW binary operation.
		// AtomicRMW binary operations.
		"AtomicOpAdd":  enum.AtomicOpAdd,  // add
		"AtomicOpAnd":  enum.AtomicOpAnd,  // and
		"AtomicOpFAdd": enum.AtomicOpFAdd, // fadd
		"AtomicOpFSub": enum.AtomicOpFSub, // fsub
		"AtomicOpMax":  enum.AtomicOpMax,  // max
		"AtomicOpMin":  enum.AtomicOpMin,  // min
		"AtomicOpNAnd": enum.AtomicOpNAnd, // nand
		"AtomicOpOr":   enum.AtomicOpOr,   // or
		"AtomicOpSub":  enum.AtomicOpSub,  // sub
		"AtomicOpUMax": enum.AtomicOpUMax, // umax
		"AtomicOpUMin": enum.AtomicOpUMin, // umin
		"AtomicOpXChg": enum.AtomicOpXChg, // xchg
		"AtomicOpXor":  enum.AtomicOpXor,  // xor
		// AtomicOrdering is an atomic ordering attribute.
		// Atomic ordering attributes.
		"AtomicOrderingNone":      enum.AtomicOrderingNone,      // none
		"AtomicOrderingAcqRel":    enum.AtomicOrderingAcqRel,    // acq_rel
		"AtomicOrderingAcquire":   enum.AtomicOrderingAcquire,   // acquire
		"AtomicOrderingMonotonic": enum.AtomicOrderingMonotonic, // monotonic
		"AtomicOrderingRelease":   enum.AtomicOrderingRelease,   // release
		"AtomicOrderingSeqCst":    enum.AtomicOrderingSeqCst,    // seq_cst
		"AtomicOrderingUnordered": enum.AtomicOrderingUnordered, // unordered
		// CallingConv is a calling convention.
		// Calling conventions.
		//
		// From include/llvm/IR/CallingConv.h
		"CallingConvNone": enum.CallingConvNone, // none
		// Note, C calling convention is defined as 0 in LLVM. To have the zero-value
		// calling convention mean no calling convention, re-define C calling
		// convention as 1, and use 0 for none.
		"CallingConvC":            enum.CallingConvC,            // ccc
		"CallingConvFast":         enum.CallingConvFast,         // fastcc
		"CallingConvCold":         enum.CallingConvCold,         // coldcc
		"CallingConvGHC":          enum.CallingConvGHC,          // ghccc
		"CallingConvHiPE":         enum.CallingConvHiPE,         // cc 11
		"CallingConvWebKitJS":     enum.CallingConvWebKitJS,     // webkit_jscc
		"CallingConvAnyReg":       enum.CallingConvAnyReg,       // anyregcc
		"CallingConvPreserveMost": enum.CallingConvPreserveMost, // preserve_mostcc
		"CallingConvPreserveAll":  enum.CallingConvPreserveAll,  // preserve_allcc
		"CallingConvSwift":        enum.CallingConvSwift,        // swiftcc
		"CallingConvCXXFastTLS":   enum.CallingConvCXXFastTLS,   // cxx_fast_tlscc
		"CallingConvTail":         enum.CallingConvTail,         // tailcc
		"CallingConvCFGuardCheck": enum.CallingConvCFGuardCheck, // cfguard_checkcc
		// Start of target-specific calling conventions.
		"CallingConvFirstTarget":          enum.CallingConvFirstTarget,          // CallingConvX86StdCall
		"CallingConvX86StdCall":           enum.CallingConvX86StdCall,           // x86_stdcallcc
		"CallingConvX86FastCall":          enum.CallingConvX86FastCall,          // x86_fastcallcc
		"CallingConvARM_APCS":             enum.CallingConvARM_APCS,             // arm_apcscc
		"CallingConvARM_AAPCS":            enum.CallingConvARM_AAPCS,            // arm_aapcscc
		"CallingConvARM_AAPCS_VFP":        enum.CallingConvARM_AAPCS_VFP,        // arm_aapcs_vfpcc
		"CallingConvMSP430Intr":           enum.CallingConvMSP430Intr,           // msp430_intrcc
		"CallingConvX86ThisCall":          enum.CallingConvX86ThisCall,          // x86_thiscallcc
		"CallingConvPTXKernel":            enum.CallingConvPTXKernel,            // ptx_kernel
		"CallingConvPTXDevice":            enum.CallingConvPTXDevice,            // ptx_device
		"CallingConvSPIRFunc":             enum.CallingConvSPIRFunc,             // spir_func
		"CallingConvSPIRKernel":           enum.CallingConvSPIRKernel,           // spir_kernel
		"CallingConvIntelOCL_BI":          enum.CallingConvIntelOCL_BI,          // intel_ocl_bicc
		"CallingConvX86_64SysV":           enum.CallingConvX86_64SysV,           // x86_64_sysvcc
		"CallingConvWin64":                enum.CallingConvWin64,                // win64cc
		"CallingConvX86VectorCall":        enum.CallingConvX86VectorCall,        // x86_vectorcallcc
		"CallingConvHHVM":                 enum.CallingConvHHVM,                 // hhvmcc
		"CallingConvHHVM_C":               enum.CallingConvHHVM_C,               // hhvm_ccc
		"CallingConvX86Intr":              enum.CallingConvX86Intr,              // x86_intrcc
		"CallingConvAVRIntr":              enum.CallingConvAVRIntr,              // avr_intrcc
		"CallingConvAVRSignal":            enum.CallingConvAVRSignal,            // avr_signalcc
		"CallingConvAVRBuiltin":           enum.CallingConvAVRBuiltin,           // cc 86
		"CallingConvAMDGPU_VS":            enum.CallingConvAMDGPU_VS,            // amdgpu_vs
		"CallingConvAMDGPU_GS":            enum.CallingConvAMDGPU_GS,            // amdgpu_gs
		"CallingConvAMDGPU_PS":            enum.CallingConvAMDGPU_PS,            // amdgpu_ps
		"CallingConvAMDGPU_CS":            enum.CallingConvAMDGPU_CS,            // amdgpu_cs
		"CallingConvAMDGPUKernel":         enum.CallingConvAMDGPUKernel,         // amdgpu_kernel
		"CallingConvX86RegCall":           enum.CallingConvX86RegCall,           // x86_regcallcc
		"CallingConvAMDGPU_HS":            enum.CallingConvAMDGPU_HS,            // amdgpu_hs
		"CallingConvMSP430Builtin":        enum.CallingConvMSP430Builtin,        // cc 94
		"CallingConvAMDGPU_LS":            enum.CallingConvAMDGPU_LS,            // amdgpu_ls
		"CallingConvAMDGPU_ES":            enum.CallingConvAMDGPU_ES,            // amdgpu_es
		"CallingConvAArch64VectorCall":    enum.CallingConvAArch64VectorCall,    // aarch64_vector_pcs
		"CallingConvAArch64SVEVectorCall": enum.CallingConvAArch64SVEVectorCall, // aarch64_sve_vector_pcs
		// ChecksumKind is a checksum algorithm.
		// Checksum algorithms.
		//
		// From include/llvm/IR/DebugInfoMetadata.h
		"ChecksumKindMD5":  enum.ChecksumKindMD5,  // CSK_MD5
		"ChecksumKindSHA1": enum.ChecksumKindSHA1, // CSK_SHA1
		// ClauseType specifies the clause type of a landingpad clause.
		// Clause types.
		"ClauseTypeCatch":  enum.ClauseTypeCatch,  // catch
		"ClauseTypeFilter": enum.ClauseTypeFilter, // filter
		// DIFlag is a debug info flag bitfield.
		// Debug info flags.
		//
		// From include/llvm/IR/DebugInfoFlags.def (LLVM 9.0
		"DIFlagZero":                enum.DIFlagZero,
		"DIFlagPrivate":             enum.DIFlagPrivate,
		"DIFlagProtected":           enum.DIFlagProtected,
		"DIFlagPublic":              enum.DIFlagPublic,
		"DIFlagFwdDecl":             enum.DIFlagFwdDecl,
		"DIFlagAppleBlock":          enum.DIFlagAppleBlock,
		"DIFlagBlockByrefStruct":    enum.DIFlagBlockByrefStruct,
		"DIFlagVirtual":             enum.DIFlagVirtual,
		"DIFlagArtificial":          enum.DIFlagArtificial,
		"DIFlagExplicit":            enum.DIFlagExplicit,
		"DIFlagPrototyped":          enum.DIFlagPrototyped,
		"DIFlagObjcClassComplete":   enum.DIFlagObjcClassComplete,
		"DIFlagObjectPointer":       enum.DIFlagObjectPointer,
		"DIFlagVector":              enum.DIFlagVector,
		"DIFlagStaticMember":        enum.DIFlagStaticMember,
		"DIFlagLValueReference":     enum.DIFlagLValueReference,
		"DIFlagRValueReference":     enum.DIFlagRValueReference,
		"DIFlagReserved":            enum.DIFlagReserved,
		"DIFlagSingleInheritance":   enum.DIFlagSingleInheritance,
		"DIFlagMultipleInheritance": enum.DIFlagMultipleInheritance,
		"DIFlagVirtualInheritance":  enum.DIFlagVirtualInheritance,
		"DIFlagIntroducedVirtual":   enum.DIFlagIntroducedVirtual,
		"DIFlagBitField":            enum.DIFlagBitField,
		"DIFlagNoReturn":            enum.DIFlagNoReturn,
		"DIFlagArgumentNotModified": enum.DIFlagArgumentNotModified,
		"DIFlagTypePassByValue":     enum.DIFlagTypePassByValue,
		"DIFlagTypePassByReference": enum.DIFlagTypePassByReference,
		"DIFlagEnumClass":           enum.DIFlagEnumClass,
		"DIFlagThunk":               enum.DIFlagThunk,
		"DIFlagNonTrivial":          enum.DIFlagNonTrivial,
		"DIFlagBigEndian":           enum.DIFlagBigEndian,
		"DIFlagLittleEndian":        enum.DIFlagLittleEndian,
		"DIFlagAllCallsDescribed":   enum.DIFlagAllCallsDescribed,
		"DIFlagIndirectVirtualBase": enum.DIFlagIndirectVirtualBase,
		// Mask for accessibility.
		"DIFlagAccessibility": enum.DIFlagAccessibility,
		// Mask for inheritance.
		"DIFlagPtrToMemberRep": enum.DIFlagPtrToMemberRep,
		// Track first and last debug info flag, used by diFlagsString in
		// ir/metadata/helper.go.
		"DIFlagFirst": enum.DIFlagFirst, // DIFlagFwdDecl
		"DIFlagLast":  enum.DIFlagLast,  // DIFlagAllCallsDescribed
		// DISPFlag is a subprogram specific flag bitfield.
		// Subprogram specific flags.
		//
		// From include/llvm/IR/DebugInfoFlags.def (LLVM 9.0
		"DISPFlagZero":           enum.DISPFlagZero,
		"DISPFlagVirtual":        enum.DISPFlagVirtual,
		"DISPFlagPureVirtual":    enum.DISPFlagPureVirtual,
		"DISPFlagLocalToUnit":    enum.DISPFlagLocalToUnit,
		"DISPFlagDefinition":     enum.DISPFlagDefinition,
		"DISPFlagOptimized":      enum.DISPFlagOptimized,
		"DISPFlagPure":           enum.DISPFlagPure,
		"DISPFlagElemental":      enum.DISPFlagElemental,
		"DISPFlagRecursive":      enum.DISPFlagRecursive,
		"DISPFlagMainSubprogram": enum.DISPFlagMainSubprogram,
		// Virtuality and non-virtuality.
		"DISPFlagNonvirtual": enum.DISPFlagNonvirtual,
		"DISPFlagVirtuality": enum.DISPFlagVirtuality,
		// Track first and last subprogram specific flag, used by diSPFlagsString in
		// ir/metadata/helper.go.
		"DISPFlagFirst": enum.DISPFlagFirst, // DISPFlagVirtual
		"DISPFlagLast":  enum.DISPFlagLast,  // DISPFlagMainSubprogram
		// DLLStorageClass specifies the DLL storage class of a global identifier.
		// DLL storage classes.
		"DLLStorageClassNone":      enum.DLLStorageClassNone,      // none
		"DLLStorageClassDLLExport": enum.DLLStorageClassDLLExport, // dllexport
		"DLLStorageClassDLLImport": enum.DLLStorageClassDLLImport, // dllimport
		// DwarfAttEncoding is a DWARF attribute type encoding.
		// DWARF attribute type encodings.
		//
		// From include/llvm/BinaryFormat/Dwarf.def
		// DWARF v2.
		"DwarfAttEncodingAddress":      enum.DwarfAttEncodingAddress,      // DW_ATE_address
		"DwarfAttEncodingBoolean":      enum.DwarfAttEncodingBoolean,      // DW_ATE_boolean
		"DwarfAttEncodingComplexFloat": enum.DwarfAttEncodingComplexFloat, // DW_ATE_complex_float
		"DwarfAttEncodingFloat":        enum.DwarfAttEncodingFloat,        // DW_ATE_float
		"DwarfAttEncodingSigned":       enum.DwarfAttEncodingSigned,       // DW_ATE_signed
		"DwarfAttEncodingSignedChar":   enum.DwarfAttEncodingSignedChar,   // DW_ATE_signed_char
		"DwarfAttEncodingUnsigned":     enum.DwarfAttEncodingUnsigned,     // DW_ATE_unsigned
		"DwarfAttEncodingUnsignedChar": enum.DwarfAttEncodingUnsignedChar, // DW_ATE_unsigned_char
		// DWARF v3.
		"DwarfAttEncodingImaginaryFloat": enum.DwarfAttEncodingImaginaryFloat, // DW_ATE_imaginary_float
		"DwarfAttEncodingPackedDecimal":  enum.DwarfAttEncodingPackedDecimal,  // DW_ATE_packed_decimal
		"DwarfAttEncodingNumericString":  enum.DwarfAttEncodingNumericString,  // DW_ATE_numeric_string
		"DwarfAttEncodingEdited":         enum.DwarfAttEncodingEdited,         // DW_ATE_edited
		"DwarfAttEncodingSignedFixed":    enum.DwarfAttEncodingSignedFixed,    // DW_ATE_signed_fixed
		"DwarfAttEncodingUnsignedFixed":  enum.DwarfAttEncodingUnsignedFixed,  // DW_ATE_unsigned_fixed
		"DwarfAttEncodingDecimalFloat":   enum.DwarfAttEncodingDecimalFloat,   // DW_ATE_decimal_float
		// DWARF v4.
		"DwarfAttEncodingUTF": enum.DwarfAttEncodingUTF, // DW_ATE_UTF
		// DWARF v5.
		"DwarfAttEncodingUCS":   enum.DwarfAttEncodingUCS,   // DW_ATE_UCS
		"DwarfAttEncodingASCII": enum.DwarfAttEncodingASCII, // DW_ATE_ASCII
		// DwarfCC is a DWARF calling convention.
		// DWARF calling conventions.
		"DwarfCCNormal":  enum.DwarfCCNormal,  // DW_CC_normal
		"DwarfCCProgram": enum.DwarfCCProgram, // DW_CC_program
		"DwarfCCNoCall":  enum.DwarfCCNoCall,  // DW_CC_nocall
		// DWARF v5.
		"DwarfCCPassByReference": enum.DwarfCCPassByReference, // DW_CC_pass_by_reference
		"DwarfCCPassByValue":     enum.DwarfCCPassByValue,     // DW_CC_pass_by_value
		// Vendor extensions.
		"DwarfCCGNUBorlandFastcallI386": enum.DwarfCCGNUBorlandFastcallI386, // DW_CC_GNU_borland_fastcall_i386
		"DwarfCCBORLANDSafecall":        enum.DwarfCCBORLANDSafecall,        // DW_CC_BORLAND_safecall
		"DwarfCCBORLANDStdcall":         enum.DwarfCCBORLANDStdcall,         // DW_CC_BORLAND_stdcall
		"DwarfCCBORLANDPascal":          enum.DwarfCCBORLANDPascal,          // DW_CC_BORLAND_pascal
		"DwarfCCBORLANDMSFastcall":      enum.DwarfCCBORLANDMSFastcall,      // DW_CC_BORLAND_msfastcall
		"DwarfCCBORLANDMSReturn":        enum.DwarfCCBORLANDMSReturn,        // DW_CC_BORLAND_msreturn
		"DwarfCCBORLANDThiscall":        enum.DwarfCCBORLANDThiscall,        // DW_CC_BORLAND_thiscall
		"DwarfCCBORLANDFastcall":        enum.DwarfCCBORLANDFastcall,        // DW_CC_BORLAND_fastcall
		"DwarfCCLLVMVectorcall":         enum.DwarfCCLLVMVectorcall,         // DW_CC_LLVM_vectorcall
		// DwarfLang is a DWARF language.
		// DWARF languages.
		//
		// From include/llvm/BinaryFormat/Dwarf.def
		// DWARF v2.
		"DwarfLangC89":       enum.DwarfLangC89,       // DW_LANG_C89
		"DwarfLangC":         enum.DwarfLangC,         // DW_LANG_C
		"DwarfLangAda83":     enum.DwarfLangAda83,     // DW_LANG_Ada83
		"DwarfLangCPlusPlus": enum.DwarfLangCPlusPlus, // DW_LANG_C_plus_plus
		"DwarfLangCobol74":   enum.DwarfLangCobol74,   // DW_LANG_Cobol74
		"DwarfLangCobol85":   enum.DwarfLangCobol85,   // DW_LANG_Cobol85
		"DwarfLangFortran77": enum.DwarfLangFortran77, // DW_LANG_Fortran77
		"DwarfLangFortran90": enum.DwarfLangFortran90, // DW_LANG_Fortran90
		"DwarfLangPascal83":  enum.DwarfLangPascal83,  // DW_LANG_Pascal83
		"DwarfLangModula2":   enum.DwarfLangModula2,   // DW_LANG_Modula2
		// DWARF v3.
		"DwarfLangJava":         enum.DwarfLangJava,         // DW_LANG_Java
		"DwarfLangC99":          enum.DwarfLangC99,          // DW_LANG_C99
		"DwarfLangAda95":        enum.DwarfLangAda95,        // DW_LANG_Ada95
		"DwarfLangFortran95":    enum.DwarfLangFortran95,    // DW_LANG_Fortran95
		"DwarfLangPLI":          enum.DwarfLangPLI,          // DW_LANG_PLI
		"DwarfLangObjC":         enum.DwarfLangObjC,         // DW_LANG_ObjC
		"DwarfLangObjCPlusPlus": enum.DwarfLangObjCPlusPlus, // DW_LANG_ObjC_plus_plus
		"DwarfLangUPC":          enum.DwarfLangUPC,          // DW_LANG_UPC
		"DwarfLangD":            enum.DwarfLangD,            // DW_LANG_D
		// DWARF v4.
		"DwarfLangPython": enum.DwarfLangPython, // DW_LANG_Python
		// DWARF v5.
		"DwarfLangOpenCL":       enum.DwarfLangOpenCL,       // DW_LANG_OpenCL
		"DwarfLangGo":           enum.DwarfLangGo,           // DW_LANG_Go
		"DwarfLangModula3":      enum.DwarfLangModula3,      // DW_LANG_Modula3
		"DwarfLangHaskell":      enum.DwarfLangHaskell,      // DW_LANG_Haskell
		"DwarfLangCPlusPlus03":  enum.DwarfLangCPlusPlus03,  // DW_LANG_C_plus_plus_03
		"DwarfLangCPlusPlus11":  enum.DwarfLangCPlusPlus11,  // DW_LANG_C_plus_plus_11
		"DwarfLangOCaml":        enum.DwarfLangOCaml,        // DW_LANG_OCaml
		"DwarfLangRust":         enum.DwarfLangRust,         // DW_LANG_Rust
		"DwarfLangC11":          enum.DwarfLangC11,          // DW_LANG_C11
		"DwarfLangSwift":        enum.DwarfLangSwift,        // DW_LANG_Swift
		"DwarfLangJulia":        enum.DwarfLangJulia,        // DW_LANG_Julia
		"DwarfLangDylan":        enum.DwarfLangDylan,        // DW_LANG_Dylan
		"DwarfLangCPlusPlus14":  enum.DwarfLangCPlusPlus14,  // DW_LANG_C_plus_plus_14
		"DwarfLangFortran03":    enum.DwarfLangFortran03,    // DW_LANG_Fortran03
		"DwarfLangFortran08":    enum.DwarfLangFortran08,    // DW_LANG_Fortran08
		"DwarfLangRenderScript": enum.DwarfLangRenderScript, // DW_LANG_RenderScript
		"DwarfLangBLISS":        enum.DwarfLangBLISS,        // DW_LANG_BLISS
		// Vendor extensions.
		"DwarfLangMipsAssembler":      enum.DwarfLangMipsAssembler,      // DW_LANG_Mips_Assembler
		"DwarfLangGoogleRenderScript": enum.DwarfLangGoogleRenderScript, // DW_LANG_GOOGLE_RenderScript
		"DwarfLangBorlandDelphi":      enum.DwarfLangBorlandDelphi,      // DW_LANG_BORLAND_Delphi
		// DwarfMacinfo is a macinfo type encoding.
		// Macinfo type encodings.
		//
		// From llvm/BinaryFormat/Dwarf.h
		"DwarfMacinfoDefine":    enum.DwarfMacinfoDefine,    // DW_MACINFO_define
		"DwarfMacinfoUndef":     enum.DwarfMacinfoUndef,     // DW_MACINFO_undef
		"DwarfMacinfoStartFile": enum.DwarfMacinfoStartFile, // DW_MACINFO_start_file
		"DwarfMacinfoEndFile":   enum.DwarfMacinfoEndFile,   // DW_MACINFO_end_file
		"DwarfMacinfoVendorExt": enum.DwarfMacinfoVendorExt, // DW_MACINFO_vendor_ext
		// DwarfOp is a DWARF expression operator.
		// DWARF expression operators.
		//
		// From include/llvm/BinaryFormat/Dwarf.def
		// DWARF v2.
		"DwarfOpAddr":       enum.DwarfOpAddr,       // DW_OP_addr
		"DwarfOpDeref":      enum.DwarfOpDeref,      // DW_OP_deref
		"DwarfOpConst1u":    enum.DwarfOpConst1u,    // DW_OP_const1u
		"DwarfOpConst1s":    enum.DwarfOpConst1s,    // DW_OP_const1s
		"DwarfOpConst2u":    enum.DwarfOpConst2u,    // DW_OP_const2u
		"DwarfOpConst2s":    enum.DwarfOpConst2s,    // DW_OP_const2s
		"DwarfOpConst4u":    enum.DwarfOpConst4u,    // DW_OP_const4u
		"DwarfOpConst4s":    enum.DwarfOpConst4s,    // DW_OP_const4s
		"DwarfOpConst8u":    enum.DwarfOpConst8u,    // DW_OP_const8u
		"DwarfOpConst8s":    enum.DwarfOpConst8s,    // DW_OP_const8s
		"DwarfOpConstu":     enum.DwarfOpConstu,     // DW_OP_constu
		"DwarfOpConsts":     enum.DwarfOpConsts,     // DW_OP_consts
		"DwarfOpDup":        enum.DwarfOpDup,        // DW_OP_dup
		"DwarfOpDrop":       enum.DwarfOpDrop,       // DW_OP_drop
		"DwarfOpOver":       enum.DwarfOpOver,       // DW_OP_over
		"DwarfOpPick":       enum.DwarfOpPick,       // DW_OP_pick
		"DwarfOpSwap":       enum.DwarfOpSwap,       // DW_OP_swap
		"DwarfOpRot":        enum.DwarfOpRot,        // DW_OP_rot
		"DwarfOpXderef":     enum.DwarfOpXderef,     // DW_OP_xderef
		"DwarfOpAbs":        enum.DwarfOpAbs,        // DW_OP_abs
		"DwarfOpAnd":        enum.DwarfOpAnd,        // DW_OP_and
		"DwarfOpDiv":        enum.DwarfOpDiv,        // DW_OP_div
		"DwarfOpMinus":      enum.DwarfOpMinus,      // DW_OP_minus
		"DwarfOpMod":        enum.DwarfOpMod,        // DW_OP_mod
		"DwarfOpMul":        enum.DwarfOpMul,        // DW_OP_mul
		"DwarfOpNeg":        enum.DwarfOpNeg,        // DW_OP_neg
		"DwarfOpNot":        enum.DwarfOpNot,        // DW_OP_not
		"DwarfOpOr":         enum.DwarfOpOr,         // DW_OP_or
		"DwarfOpPlus":       enum.DwarfOpPlus,       // DW_OP_plus
		"DwarfOpPlusUconst": enum.DwarfOpPlusUconst, // DW_OP_plus_uconst
		"DwarfOpShl":        enum.DwarfOpShl,        // DW_OP_shl
		"DwarfOpShr":        enum.DwarfOpShr,        // DW_OP_shr
		"DwarfOpShra":       enum.DwarfOpShra,       // DW_OP_shra
		"DwarfOpXor":        enum.DwarfOpXor,        // DW_OP_xor
		"DwarfOpBra":        enum.DwarfOpBra,        // DW_OP_bra
		"DwarfOpEq":         enum.DwarfOpEq,         // DW_OP_eq
		"DwarfOpGe":         enum.DwarfOpGe,         // DW_OP_ge
		"DwarfOpGt":         enum.DwarfOpGt,         // DW_OP_gt
		"DwarfOpLe":         enum.DwarfOpLe,         // DW_OP_le
		"DwarfOpLt":         enum.DwarfOpLt,         // DW_OP_lt
		"DwarfOpNe":         enum.DwarfOpNe,         // DW_OP_ne
		"DwarfOpSkip":       enum.DwarfOpSkip,       // DW_OP_skip
		"DwarfOpLit0":       enum.DwarfOpLit0,       // DW_OP_lit0
		"DwarfOpLit1":       enum.DwarfOpLit1,       // DW_OP_lit1
		"DwarfOpLit2":       enum.DwarfOpLit2,       // DW_OP_lit2
		"DwarfOpLit3":       enum.DwarfOpLit3,       // DW_OP_lit3
		"DwarfOpLit4":       enum.DwarfOpLit4,       // DW_OP_lit4
		"DwarfOpLit5":       enum.DwarfOpLit5,       // DW_OP_lit5
		"DwarfOpLit6":       enum.DwarfOpLit6,       // DW_OP_lit6
		"DwarfOpLit7":       enum.DwarfOpLit7,       // DW_OP_lit7
		"DwarfOpLit8":       enum.DwarfOpLit8,       // DW_OP_lit8
		"DwarfOpLit9":       enum.DwarfOpLit9,       // DW_OP_lit9
		"DwarfOpLit10":      enum.DwarfOpLit10,      // DW_OP_lit10
		"DwarfOpLit11":      enum.DwarfOpLit11,      // DW_OP_lit11
		"DwarfOpLit12":      enum.DwarfOpLit12,      // DW_OP_lit12
		"DwarfOpLit13":      enum.DwarfOpLit13,      // DW_OP_lit13
		"DwarfOpLit14":      enum.DwarfOpLit14,      // DW_OP_lit14
		"DwarfOpLit15":      enum.DwarfOpLit15,      // DW_OP_lit15
		"DwarfOpLit16":      enum.DwarfOpLit16,      // DW_OP_lit16
		"DwarfOpLit17":      enum.DwarfOpLit17,      // DW_OP_lit17
		"DwarfOpLit18":      enum.DwarfOpLit18,      // DW_OP_lit18
		"DwarfOpLit19":      enum.DwarfOpLit19,      // DW_OP_lit19
		"DwarfOpLit20":      enum.DwarfOpLit20,      // DW_OP_lit20
		"DwarfOpLit21":      enum.DwarfOpLit21,      // DW_OP_lit21
		"DwarfOpLit22":      enum.DwarfOpLit22,      // DW_OP_lit22
		"DwarfOpLit23":      enum.DwarfOpLit23,      // DW_OP_lit23
		"DwarfOpLit24":      enum.DwarfOpLit24,      // DW_OP_lit24
		"DwarfOpLit25":      enum.DwarfOpLit25,      // DW_OP_lit25
		"DwarfOpLit26":      enum.DwarfOpLit26,      // DW_OP_lit26
		"DwarfOpLit27":      enum.DwarfOpLit27,      // DW_OP_lit27
		"DwarfOpLit28":      enum.DwarfOpLit28,      // DW_OP_lit28
		"DwarfOpLit29":      enum.DwarfOpLit29,      // DW_OP_lit29
		"DwarfOpLit30":      enum.DwarfOpLit30,      // DW_OP_lit30
		"DwarfOpLit31":      enum.DwarfOpLit31,      // DW_OP_lit31
		"DwarfOpReg0":       enum.DwarfOpReg0,       // DW_OP_reg0
		"DwarfOpReg1":       enum.DwarfOpReg1,       // DW_OP_reg1
		"DwarfOpReg2":       enum.DwarfOpReg2,       // DW_OP_reg2
		"DwarfOpReg3":       enum.DwarfOpReg3,       // DW_OP_reg3
		"DwarfOpReg4":       enum.DwarfOpReg4,       // DW_OP_reg4
		"DwarfOpReg5":       enum.DwarfOpReg5,       // DW_OP_reg5
		"DwarfOpReg6":       enum.DwarfOpReg6,       // DW_OP_reg6
		"DwarfOpReg7":       enum.DwarfOpReg7,       // DW_OP_reg7
		"DwarfOpReg8":       enum.DwarfOpReg8,       // DW_OP_reg8
		"DwarfOpReg9":       enum.DwarfOpReg9,       // DW_OP_reg9
		"DwarfOpReg10":      enum.DwarfOpReg10,      // DW_OP_reg10
		"DwarfOpReg11":      enum.DwarfOpReg11,      // DW_OP_reg11
		"DwarfOpReg12":      enum.DwarfOpReg12,      // DW_OP_reg12
		"DwarfOpReg13":      enum.DwarfOpReg13,      // DW_OP_reg13
		"DwarfOpReg14":      enum.DwarfOpReg14,      // DW_OP_reg14
		"DwarfOpReg15":      enum.DwarfOpReg15,      // DW_OP_reg15
		"DwarfOpReg16":      enum.DwarfOpReg16,      // DW_OP_reg16
		"DwarfOpReg17":      enum.DwarfOpReg17,      // DW_OP_reg17
		"DwarfOpReg18":      enum.DwarfOpReg18,      // DW_OP_reg18
		"DwarfOpReg19":      enum.DwarfOpReg19,      // DW_OP_reg19
		"DwarfOpReg20":      enum.DwarfOpReg20,      // DW_OP_reg20
		"DwarfOpReg21":      enum.DwarfOpReg21,      // DW_OP_reg21
		"DwarfOpReg22":      enum.DwarfOpReg22,      // DW_OP_reg22
		"DwarfOpReg23":      enum.DwarfOpReg23,      // DW_OP_reg23
		"DwarfOpReg24":      enum.DwarfOpReg24,      // DW_OP_reg24
		"DwarfOpReg25":      enum.DwarfOpReg25,      // DW_OP_reg25
		"DwarfOpReg26":      enum.DwarfOpReg26,      // DW_OP_reg26
		"DwarfOpReg27":      enum.DwarfOpReg27,      // DW_OP_reg27
		"DwarfOpReg28":      enum.DwarfOpReg28,      // DW_OP_reg28
		"DwarfOpReg29":      enum.DwarfOpReg29,      // DW_OP_reg29
		"DwarfOpReg30":      enum.DwarfOpReg30,      // DW_OP_reg30
		"DwarfOpReg31":      enum.DwarfOpReg31,      // DW_OP_reg31
		"DwarfOpBreg0":      enum.DwarfOpBreg0,      // DW_OP_breg0
		"DwarfOpBreg1":      enum.DwarfOpBreg1,      // DW_OP_breg1
		"DwarfOpBreg2":      enum.DwarfOpBreg2,      // DW_OP_breg2
		"DwarfOpBreg3":      enum.DwarfOpBreg3,      // DW_OP_breg3
		"DwarfOpBreg4":      enum.DwarfOpBreg4,      // DW_OP_breg4
		"DwarfOpBreg5":      enum.DwarfOpBreg5,      // DW_OP_breg5
		"DwarfOpBreg6":      enum.DwarfOpBreg6,      // DW_OP_breg6
		"DwarfOpBreg7":      enum.DwarfOpBreg7,      // DW_OP_breg7
		"DwarfOpBreg8":      enum.DwarfOpBreg8,      // DW_OP_breg8
		"DwarfOpBreg9":      enum.DwarfOpBreg9,      // DW_OP_breg9
		"DwarfOpBreg10":     enum.DwarfOpBreg10,     // DW_OP_breg10
		"DwarfOpBreg11":     enum.DwarfOpBreg11,     // DW_OP_breg11
		"DwarfOpBreg12":     enum.DwarfOpBreg12,     // DW_OP_breg12
		"DwarfOpBreg13":     enum.DwarfOpBreg13,     // DW_OP_breg13
		"DwarfOpBreg14":     enum.DwarfOpBreg14,     // DW_OP_breg14
		"DwarfOpBreg15":     enum.DwarfOpBreg15,     // DW_OP_breg15
		"DwarfOpBreg16":     enum.DwarfOpBreg16,     // DW_OP_breg16
		"DwarfOpBreg17":     enum.DwarfOpBreg17,     // DW_OP_breg17
		"DwarfOpBreg18":     enum.DwarfOpBreg18,     // DW_OP_breg18
		"DwarfOpBreg19":     enum.DwarfOpBreg19,     // DW_OP_breg19
		"DwarfOpBreg20":     enum.DwarfOpBreg20,     // DW_OP_breg20
		"DwarfOpBreg21":     enum.DwarfOpBreg21,     // DW_OP_breg21
		"DwarfOpBreg22":     enum.DwarfOpBreg22,     // DW_OP_breg22
		"DwarfOpBreg23":     enum.DwarfOpBreg23,     // DW_OP_breg23
		"DwarfOpBreg24":     enum.DwarfOpBreg24,     // DW_OP_breg24
		"DwarfOpBreg25":     enum.DwarfOpBreg25,     // DW_OP_breg25
		"DwarfOpBreg26":     enum.DwarfOpBreg26,     // DW_OP_breg26
		"DwarfOpBreg27":     enum.DwarfOpBreg27,     // DW_OP_breg27
		"DwarfOpBreg28":     enum.DwarfOpBreg28,     // DW_OP_breg28
		"DwarfOpBreg29":     enum.DwarfOpBreg29,     // DW_OP_breg29
		"DwarfOpBreg30":     enum.DwarfOpBreg30,     // DW_OP_breg30
		"DwarfOpBreg31":     enum.DwarfOpBreg31,     // DW_OP_breg31
		"DwarfOpRegx":       enum.DwarfOpRegx,       // DW_OP_regx
		"DwarfOpFbreg":      enum.DwarfOpFbreg,      // DW_OP_fbreg
		"DwarfOpBregx":      enum.DwarfOpBregx,      // DW_OP_bregx
		"DwarfOpPiece":      enum.DwarfOpPiece,      // DW_OP_piece
		"DwarfOpDerefSize":  enum.DwarfOpDerefSize,  // DW_OP_deref_size
		"DwarfOpXderefSize": enum.DwarfOpXderefSize, // DW_OP_xderef_size
		"DwarfOpNop":        enum.DwarfOpNop,        // DW_OP_nop
		// DWARF v3.
		"DwarfOpPushObjectAddress": enum.DwarfOpPushObjectAddress, // DW_OP_push_object_address
		"DwarfOpCall2":             enum.DwarfOpCall2,             // DW_OP_call2
		"DwarfOpCall4":             enum.DwarfOpCall4,             // DW_OP_call4
		"DwarfOpCallRef":           enum.DwarfOpCallRef,           // DW_OP_call_ref
		"DwarfOpFormTLSAddress":    enum.DwarfOpFormTLSAddress,    // DW_OP_form_tls_address
		"DwarfOpCallFrameCFA":      enum.DwarfOpCallFrameCFA,      // DW_OP_call_frame_cfa
		"DwarfOpBitPiece":          enum.DwarfOpBitPiece,          // DW_OP_bit_piece
		// DWARF v4.
		"DwarfOpImplicitValue": enum.DwarfOpImplicitValue, // DW_OP_implicit_value
		"DwarfOpStackValue":    enum.DwarfOpStackValue,    // DW_OP_stack_value
		// DWARF v5.
		"DwarfOpImplicitPointer": enum.DwarfOpImplicitPointer, // DW_OP_implicit_pointer
		"DwarfOpAddrx":           enum.DwarfOpAddrx,           // DW_OP_addrx
		"DwarfOpConstx":          enum.DwarfOpConstx,          // DW_OP_constx
		"DwarfOpEntryValue":      enum.DwarfOpEntryValue,      // DW_OP_entry_value
		"DwarfOpConstType":       enum.DwarfOpConstType,       // DW_OP_const_type
		"DwarfOpRegvalType":      enum.DwarfOpRegvalType,      // DW_OP_regval_type
		"DwarfOpDerefType":       enum.DwarfOpDerefType,       // DW_OP_deref_type
		"DwarfOpXderefType":      enum.DwarfOpXderefType,      // DW_OP_xderef_type
		"DwarfOpConvert":         enum.DwarfOpConvert,         // DW_OP_convert
		"DwarfOpReinterpret":     enum.DwarfOpReinterpret,     // DW_OP_reinterpret
		// Vendor extensions.
		"DwarfOpGNUPushTLSAddress": enum.DwarfOpGNUPushTLSAddress, // DW_OP_GNU_push_tls_address
		"DwarfOpGNUEntryValue":     enum.DwarfOpGNUEntryValue,     // DW_OP_GNU_entry_value
		"DwarfOpGNUAddrIndex":      enum.DwarfOpGNUAddrIndex,      // DW_OP_GNU_addr_index
		"DwarfOpGNUConstIndex":     enum.DwarfOpGNUConstIndex,     // DW_OP_GNU_const_index
		// Only used in LLVM metadata.
		"DwarfOpLLVMFragment":  enum.DwarfOpLLVMFragment,  // DW_OP_LLVM_fragment
		"DwarfOpLLVMConvert":   enum.DwarfOpLLVMConvert,   // DW_OP_LLVM_convert
		"DwarfOpLLVMTagOffset": enum.DwarfOpLLVMTagOffset, // DW_OP_LLVM_tag_offset
		// DwarfTag is a DWARF tag.
		// DWARF tags.
		//
		// From include/llvm/BinaryFormat/Dwarf.def
		// DWARF v2.
		"DwarfTagNull":                   enum.DwarfTagNull,                   // DW_TAG_null
		"DwarfTagArrayType":              enum.DwarfTagArrayType,              // DW_TAG_array_type
		"DwarfTagClassType":              enum.DwarfTagClassType,              // DW_TAG_class_type
		"DwarfTagEntryPoint":             enum.DwarfTagEntryPoint,             // DW_TAG_entry_point
		"DwarfTagEnumerationType":        enum.DwarfTagEnumerationType,        // DW_TAG_enumeration_type
		"DwarfTagFormalParameter":        enum.DwarfTagFormalParameter,        // DW_TAG_formal_parameter
		"DwarfTagImportedDeclaration":    enum.DwarfTagImportedDeclaration,    // DW_TAG_imported_declaration
		"DwarfTagLabel":                  enum.DwarfTagLabel,                  // DW_TAG_label
		"DwarfTagLexicalBlock":           enum.DwarfTagLexicalBlock,           // DW_TAG_lexical_block
		"DwarfTagMember":                 enum.DwarfTagMember,                 // DW_TAG_member
		"DwarfTagPointerType":            enum.DwarfTagPointerType,            // DW_TAG_pointer_type
		"DwarfTagReferenceType":          enum.DwarfTagReferenceType,          // DW_TAG_reference_type
		"DwarfTagCompileUnit":            enum.DwarfTagCompileUnit,            // DW_TAG_compile_unit
		"DwarfTagStringType":             enum.DwarfTagStringType,             // DW_TAG_string_type
		"DwarfTagStructureType":          enum.DwarfTagStructureType,          // DW_TAG_structure_type
		"DwarfTagSubroutineType":         enum.DwarfTagSubroutineType,         // DW_TAG_subroutine_type
		"DwarfTagTypedef":                enum.DwarfTagTypedef,                // DW_TAG_typedef
		"DwarfTagUnionType":              enum.DwarfTagUnionType,              // DW_TAG_union_type
		"DwarfTagUnspecifiedParameters":  enum.DwarfTagUnspecifiedParameters,  // DW_TAG_unspecified_parameters
		"DwarfTagVariant":                enum.DwarfTagVariant,                // DW_TAG_variant
		"DwarfTagCommonBlock":            enum.DwarfTagCommonBlock,            // DW_TAG_common_block
		"DwarfTagCommonInclusion":        enum.DwarfTagCommonInclusion,        // DW_TAG_common_inclusion
		"DwarfTagInheritance":            enum.DwarfTagInheritance,            // DW_TAG_inheritance
		"DwarfTagInlinedSubroutine":      enum.DwarfTagInlinedSubroutine,      // DW_TAG_inlined_subroutine
		"DwarfTagModule":                 enum.DwarfTagModule,                 // DW_TAG_module
		"DwarfTagPtrToMemberType":        enum.DwarfTagPtrToMemberType,        // DW_TAG_ptr_to_member_type
		"DwarfTagSetType":                enum.DwarfTagSetType,                // DW_TAG_set_type
		"DwarfTagSubrangeType":           enum.DwarfTagSubrangeType,           // DW_TAG_subrange_type
		"DwarfTagWithStmt":               enum.DwarfTagWithStmt,               // DW_TAG_with_stmt
		"DwarfTagAccessDeclaration":      enum.DwarfTagAccessDeclaration,      // DW_TAG_access_declaration
		"DwarfTagBaseType":               enum.DwarfTagBaseType,               // DW_TAG_base_type
		"DwarfTagCatchBlock":             enum.DwarfTagCatchBlock,             // DW_TAG_catch_block
		"DwarfTagConstType":              enum.DwarfTagConstType,              // DW_TAG_const_type
		"DwarfTagConstant":               enum.DwarfTagConstant,               // DW_TAG_constant
		"DwarfTagEnumerator":             enum.DwarfTagEnumerator,             // DW_TAG_enumerator
		"DwarfTagFileType":               enum.DwarfTagFileType,               // DW_TAG_file_type
		"DwarfTagFriend":                 enum.DwarfTagFriend,                 // DW_TAG_friend
		"DwarfTagNamelist":               enum.DwarfTagNamelist,               // DW_TAG_namelist
		"DwarfTagNamelistItem":           enum.DwarfTagNamelistItem,           // DW_TAG_namelist_item
		"DwarfTagPackedType":             enum.DwarfTagPackedType,             // DW_TAG_packed_type
		"DwarfTagSubprogram":             enum.DwarfTagSubprogram,             // DW_TAG_subprogram
		"DwarfTagTemplateTypeParameter":  enum.DwarfTagTemplateTypeParameter,  // DW_TAG_template_type_parameter
		"DwarfTagTemplateValueParameter": enum.DwarfTagTemplateValueParameter, // DW_TAG_template_value_parameter
		"DwarfTagThrownType":             enum.DwarfTagThrownType,             // DW_TAG_thrown_type
		"DwarfTagTryBlock":               enum.DwarfTagTryBlock,               // DW_TAG_try_block
		"DwarfTagVariantPart":            enum.DwarfTagVariantPart,            // DW_TAG_variant_part
		"DwarfTagVariable":               enum.DwarfTagVariable,               // DW_TAG_variable
		"DwarfTagVolatileType":           enum.DwarfTagVolatileType,           // DW_TAG_volatile_type
		// DWARF v3.
		"DwarfTagDwarfProcedure":  enum.DwarfTagDwarfProcedure,  // DW_TAG_dwarf_procedure
		"DwarfTagRestrictType":    enum.DwarfTagRestrictType,    // DW_TAG_restrict_type
		"DwarfTagInterfaceType":   enum.DwarfTagInterfaceType,   // DW_TAG_interface_type
		"DwarfTagNamespace":       enum.DwarfTagNamespace,       // DW_TAG_namespace
		"DwarfTagImportedModule":  enum.DwarfTagImportedModule,  // DW_TAG_imported_module
		"DwarfTagUnspecifiedType": enum.DwarfTagUnspecifiedType, // DW_TAG_unspecified_type
		"DwarfTagPartialUnit":     enum.DwarfTagPartialUnit,     // DW_TAG_partial_unit
		"DwarfTagImportedUnit":    enum.DwarfTagImportedUnit,    // DW_TAG_imported_unit
		"DwarfTagCondition":       enum.DwarfTagCondition,       // DW_TAG_condition
		"DwarfTagSharedType":      enum.DwarfTagSharedType,      // DW_TAG_shared_type
		// DWARF v4.
		"DwarfTagTypeUnit":            enum.DwarfTagTypeUnit,            // DW_TAG_type_unit
		"DwarfTagRvalueReferenceType": enum.DwarfTagRvalueReferenceType, // DW_TAG_rvalue_reference_type
		"DwarfTagTemplateAlias":       enum.DwarfTagTemplateAlias,       // DW_TAG_template_alias
		// DWARF v5.
		"DwarfTagCoarrayType":       enum.DwarfTagCoarrayType,       // DW_TAG_coarray_type
		"DwarfTagGenericSubrange":   enum.DwarfTagGenericSubrange,   // DW_TAG_generic_subrange
		"DwarfTagDynamicType":       enum.DwarfTagDynamicType,       // DW_TAG_dynamic_type
		"DwarfTagAtomicType":        enum.DwarfTagAtomicType,        // DW_TAG_atomic_type
		"DwarfTagCallSite":          enum.DwarfTagCallSite,          // DW_TAG_call_site
		"DwarfTagCallSiteParameter": enum.DwarfTagCallSiteParameter, // DW_TAG_call_site_parameter
		"DwarfTagSkeletonUnit":      enum.DwarfTagSkeletonUnit,      // DW_TAG_skeleton_unit
		"DwarfTagImmutableType":     enum.DwarfTagImmutableType,     // DW_TAG_immutable_type
		// Vendor extensions.
		"DwarfTagMIPSLoop":                  enum.DwarfTagMIPSLoop,                  // DW_TAG_MIPS_loop
		"DwarfTagFormatLabel":               enum.DwarfTagFormatLabel,               // DW_TAG_format_label
		"DwarfTagFunctionTemplate":          enum.DwarfTagFunctionTemplate,          // DW_TAG_function_template
		"DwarfTagClassTemplate":             enum.DwarfTagClassTemplate,             // DW_TAG_class_template
		"DwarfTagGNUTemplateTemplateParam":  enum.DwarfTagGNUTemplateTemplateParam,  // DW_TAG_GNU_template_template_param
		"DwarfTagGNUTemplateParameterPack":  enum.DwarfTagGNUTemplateParameterPack,  // DW_TAG_GNU_template_parameter_pack
		"DwarfTagGNUFormalParameterPack":    enum.DwarfTagGNUFormalParameterPack,    // DW_TAG_GNU_formal_parameter_pack
		"DwarfTagGNUCallSite":               enum.DwarfTagGNUCallSite,               // DW_TAG_GNU_call_site
		"DwarfTagGNUCallSiteParameter":      enum.DwarfTagGNUCallSiteParameter,      // DW_TAG_GNU_call_site_parameter
		"DwarfTagAPPLEProperty":             enum.DwarfTagAPPLEProperty,             // DW_TAG_APPLE_property
		"DwarfTagBORLANDProperty":           enum.DwarfTagBORLANDProperty,           // DW_TAG_BORLAND_property
		"DwarfTagBORLANDDelphiString":       enum.DwarfTagBORLANDDelphiString,       // DW_TAG_BORLAND_Delphi_string
		"DwarfTagBORLANDDelphiDynamicArray": enum.DwarfTagBORLANDDelphiDynamicArray, // DW_TAG_BORLAND_Delphi_dynamic_array
		"DwarfTagBORLANDDelphiSet":          enum.DwarfTagBORLANDDelphiSet,          // DW_TAG_BORLAND_Delphi_set
		"DwarfTagBORLANDDelphiVariant":      enum.DwarfTagBORLANDDelphiVariant,      // DW_TAG_BORLAND_Delphi_variant
		// DwarfVirtuality is a DWARF virtuality code.
		// DWARF virtuality codes.
		"DwarfVirtualityNone":        enum.DwarfVirtualityNone,        // DW_VIRTUALITY_none
		"DwarfVirtualityVirtual":     enum.DwarfVirtualityVirtual,     // DW_VIRTUALITY_virtual
		"DwarfVirtualityPureVirtual": enum.DwarfVirtualityPureVirtual, // DW_VIRTUALITY_pure_virtual
		// EmissionKind specifies the debug emission kind.
		// Debug emission kinds.
		"EmissionKindNoDebug":             enum.EmissionKindNoDebug,             // NoDebug
		"EmissionKindFullDebug":           enum.EmissionKindFullDebug,           // FullDebug
		"EmissionKindLineTablesOnly":      enum.EmissionKindLineTablesOnly,      // LineTablesOnly
		"EmissionKindDebugDirectivesOnly": enum.EmissionKindDebugDirectivesOnly, // DebugDirectivesOnly
		// FastMathFlag is a fast-math flag.
		// Fast-math flags.
		"FastMathFlagAFn":      enum.FastMathFlagAFn,      // afn
		"FastMathFlagARcp":     enum.FastMathFlagARcp,     // arcp
		"FastMathFlagContract": enum.FastMathFlagContract, // contract
		"FastMathFlagFast":     enum.FastMathFlagFast,     // fast
		"FastMathFlagNInf":     enum.FastMathFlagNInf,     // ninf
		"FastMathFlagNNaN":     enum.FastMathFlagNNaN,     // nnan
		"FastMathFlagNSZ":      enum.FastMathFlagNSZ,      // nsz
		"FastMathFlagReassoc":  enum.FastMathFlagReassoc,  // reassoc
		// FPred is a floating-point comparison predicate.
		// Floating-point predicates.
		"FPredFalse": enum.FPredFalse, // false
		"FPredOEQ":   enum.FPredOEQ,   // oeq
		"FPredOGE":   enum.FPredOGE,   // oge
		"FPredOGT":   enum.FPredOGT,   // ogt
		"FPredOLE":   enum.FPredOLE,   // ole
		"FPredOLT":   enum.FPredOLT,   // olt
		"FPredONE":   enum.FPredONE,   // one
		"FPredORD":   enum.FPredORD,   // ord
		"FPredTrue":  enum.FPredTrue,  // true
		"FPredUEQ":   enum.FPredUEQ,   // ueq
		"FPredUGE":   enum.FPredUGE,   // uge
		"FPredUGT":   enum.FPredUGT,   // ugt
		"FPredULE":   enum.FPredULE,   // ule
		"FPredULT":   enum.FPredULT,   // ult
		"FPredUNE":   enum.FPredUNE,   // une
		"FPredUNO":   enum.FPredUNO,   // uno
		// FuncAttr is a function attribute.
		// Function attributes.
		"FuncAttrAlwaysInline":                enum.FuncAttrAlwaysInline,                // alwaysinline
		"FuncAttrArgMemOnly":                  enum.FuncAttrArgMemOnly,                  // argmemonly
		"FuncAttrBuiltin":                     enum.FuncAttrBuiltin,                     // builtin
		"FuncAttrCold":                        enum.FuncAttrCold,                        // cold
		"FuncAttrConvergent":                  enum.FuncAttrConvergent,                  // convergent
		"FuncAttrInaccessibleMemOnly":         enum.FuncAttrInaccessibleMemOnly,         // inaccessiblememonly
		"FuncAttrInaccessibleMemOrArgMemOnly": enum.FuncAttrInaccessibleMemOrArgMemOnly, // inaccessiblemem_or_argmemonly
		"FuncAttrInlineHint":                  enum.FuncAttrInlineHint,                  // inlinehint
		"FuncAttrJumpTable":                   enum.FuncAttrJumpTable,                   // jumptable
		"FuncAttrMinSize":                     enum.FuncAttrMinSize,                     // minsize
		"FuncAttrNaked":                       enum.FuncAttrNaked,                       // naked
		"FuncAttrNoBuiltin":                   enum.FuncAttrNoBuiltin,                   // nobuiltin
		"FuncAttrNoCFCheck":                   enum.FuncAttrNoCFCheck,                   // nocf_check
		"FuncAttrNoDuplicate":                 enum.FuncAttrNoDuplicate,                 // noduplicate
		"FuncAttrNoFree":                      enum.FuncAttrNoFree,                      // nofree
		"FuncAttrNoImplicitFloat":             enum.FuncAttrNoImplicitFloat,             // noimplicitfloat
		"FuncAttrNoInline":                    enum.FuncAttrNoInline,                    // noinline
		"FuncAttrNonLazyBind":                 enum.FuncAttrNonLazyBind,                 // nonlazybind
		"FuncAttrNoRecurse":                   enum.FuncAttrNoRecurse,                   // norecurse
		"FuncAttrNoRedZone":                   enum.FuncAttrNoRedZone,                   // noredzone
		"FuncAttrNoReturn":                    enum.FuncAttrNoReturn,                    // noreturn
		"FuncAttrNoSync":                      enum.FuncAttrNoSync,                      // nosync
		"FuncAttrNoUnwind":                    enum.FuncAttrNoUnwind,                    // nounwind
		"FuncAttrOptForFuzzing":               enum.FuncAttrOptForFuzzing,               // optforfuzzing
		"FuncAttrOptNone":                     enum.FuncAttrOptNone,                     // optnone
		"FuncAttrOptSize":                     enum.FuncAttrOptSize,                     // optsize
		"FuncAttrReadNone":                    enum.FuncAttrReadNone,                    // readnone
		"FuncAttrReadOnly":                    enum.FuncAttrReadOnly,                    // readonly
		"FuncAttrReturnsTwice":                enum.FuncAttrReturnsTwice,                // returns_twice
		"FuncAttrSafeStack":                   enum.FuncAttrSafeStack,                   // safestack
		"FuncAttrSanitizeAddress":             enum.FuncAttrSanitizeAddress,             // sanitize_address
		"FuncAttrSanitizeHWAddress":           enum.FuncAttrSanitizeHWAddress,           // sanitize_hwaddress
		"FuncAttrSanitizeMemory":              enum.FuncAttrSanitizeMemory,              // sanitize_memory
		"FuncAttrSanitizeMemTag":              enum.FuncAttrSanitizeMemTag,              // sanitize_memtag
		"FuncAttrSanitizeThread":              enum.FuncAttrSanitizeThread,              // sanitize_thread
		"FuncAttrShadowCallStack":             enum.FuncAttrShadowCallStack,             // shadowcallstack
		"FuncAttrSpeculatable":                enum.FuncAttrSpeculatable,                // speculatable
		"FuncAttrSpeculativeLoadHardening":    enum.FuncAttrSpeculativeLoadHardening,    // speculative_load_hardening
		"FuncAttrSSP":                         enum.FuncAttrSSP,                         // ssp
		"FuncAttrSSPReq":                      enum.FuncAttrSSPReq,                      // sspreq
		"FuncAttrSSPStrong":                   enum.FuncAttrSSPStrong,                   // sspstrong
		"FuncAttrStrictFP":                    enum.FuncAttrStrictFP,                    // strictfp
		"FuncAttrUwtable":                     enum.FuncAttrUwtable,                     // uwtable
		"FuncAttrWillReturn":                  enum.FuncAttrWillReturn,                  // willreturn
		"FuncAttrWriteOnly":                   enum.FuncAttrWriteOnly,                   // writeonly
		// IPred is an integer comparison predicate.
		// Integer predicates.
		"IPredEQ":  enum.IPredEQ,  // eq
		"IPredNE":  enum.IPredNE,  // ne
		"IPredSGE": enum.IPredSGE, // sge
		"IPredSGT": enum.IPredSGT, // sgt
		"IPredSLE": enum.IPredSLE, // sle
		"IPredSLT": enum.IPredSLT, // slt
		"IPredUGE": enum.IPredUGE, // uge
		"IPredUGT": enum.IPredUGT, // ugt
		"IPredULE": enum.IPredULE, // ule
		"IPredULT": enum.IPredULT, // ult
		// Linkage specifies the linkage of a global identifier.
		// Linkage kinds.
		"LinkageNone":                enum.LinkageNone,                // none
		"LinkageAppending":           enum.LinkageAppending,           // appending
		"LinkageAvailableExternally": enum.LinkageAvailableExternally, // available_externally
		"LinkageCommon":              enum.LinkageCommon,              // common
		"LinkageInternal":            enum.LinkageInternal,            // internal
		"LinkageLinkOnce":            enum.LinkageLinkOnce,            // linkonce
		"LinkageLinkOnceODR":         enum.LinkageLinkOnceODR,         // linkonce_odr
		"LinkagePrivate":             enum.LinkagePrivate,             // private
		"LinkageWeak":                enum.LinkageWeak,                // weak
		"LinkageWeakODR":             enum.LinkageWeakODR,             // weak_odr
		// External linkage.
		"LinkageExternal":   enum.LinkageExternal,   // external
		"LinkageExternWeak": enum.LinkageExternWeak, // extern_weak
		// NameTableKind is a name table specifier.
		// Name table kinds.
		//
		// From include/llvm/IR/DebugInfoMetadata.h
		"NameTableKindDefault": enum.NameTableKindDefault, // Default
		"NameTableKindGNU":     enum.NameTableKindGNU,     // GNU
		"NameTableKindNone":    enum.NameTableKindNone,    // None
		// OverflowFlag is an integer overflow flag.
		// Overflow flags.
		"OverflowFlagNSW": enum.OverflowFlagNSW, // nsw
		"OverflowFlagNUW": enum.OverflowFlagNUW, // nuw
		// ParamAttr is a parameter attribute.
		// Parameter attributes.
		"ParamAttrImmArg":     enum.ParamAttrImmArg,     // immarg
		"ParamAttrInAlloca":   enum.ParamAttrInAlloca,   // inalloca
		"ParamAttrInReg":      enum.ParamAttrInReg,      // inreg
		"ParamAttrNest":       enum.ParamAttrNest,       // nest
		"ParamAttrNoAlias":    enum.ParamAttrNoAlias,    // noalias
		"ParamAttrNoCapture":  enum.ParamAttrNoCapture,  // nocapture
		"ParamAttrNoFree":     enum.ParamAttrNoFree,     // nofree
		"ParamAttrNonNull":    enum.ParamAttrNonNull,    // nonnull
		"ParamAttrReadNone":   enum.ParamAttrReadNone,   // readnone
		"ParamAttrReadOnly":   enum.ParamAttrReadOnly,   // readonly
		"ParamAttrReturned":   enum.ParamAttrReturned,   // returned
		"ParamAttrSignExt":    enum.ParamAttrSignExt,    // signext
		"ParamAttrSRet":       enum.ParamAttrSRet,       // sret
		"ParamAttrSwiftError": enum.ParamAttrSwiftError, // swifterror
		"ParamAttrSwiftSelf":  enum.ParamAttrSwiftSelf,  // swiftself
		"ParamAttrWriteOnly":  enum.ParamAttrWriteOnly,  // writeonly
		"ParamAttrZeroExt":    enum.ParamAttrZeroExt,    // zeroext
		// Preemption specifies the preemtion of a global identifier.
		// Preemption kinds.
		"PreemptionNone":           enum.PreemptionNone,           // none
		"PreemptionDSOLocal":       enum.PreemptionDSOLocal,       // dso_local
		"PreemptionDSOPreemptable": enum.PreemptionDSOPreemptable, // dso_preemptable
		// ReturnAttr is a return argument attribute.
		// Return argument attributes.
		"ReturnAttrInReg":   enum.ReturnAttrInReg,   // inreg
		"ReturnAttrNoAlias": enum.ReturnAttrNoAlias, // noalias
		"ReturnAttrNonNull": enum.ReturnAttrNonNull, // nonnull
		"ReturnAttrSignExt": enum.ReturnAttrSignExt, // signext
		"ReturnAttrZeroExt": enum.ReturnAttrZeroExt, // zeroext
		// SelectionKind is a Comdat selection kind.
		// Comdat selection kinds.
		"SelectionKindAny":          enum.SelectionKindAny,          // any
		"SelectionKindExactMatch":   enum.SelectionKindExactMatch,   // exactmatch
		"SelectionKindLargest":      enum.SelectionKindLargest,      // largest
		"SelectionKindNoDuplicates": enum.SelectionKindNoDuplicates, // noduplicates
		"SelectionKindSameSize":     enum.SelectionKindSameSize,     // samesize
		// Tail is a tail call attribute.
		// Tail call attributes.
		"TailNone":     enum.TailNone,     // none
		"TailMustTail": enum.TailMustTail, // musttail
		"TailNoTail":   enum.TailNoTail,   // notail
		"TailTail":     enum.TailTail,     // tail
		// TLSModel is a thread local storage model.
		// Thread local storage models.
		"TLSModelNone": enum.TLSModelNone, // none
		// If no explicit model is given, the "general dynamic" model is used.
		"TLSModelGeneric":      enum.TLSModelGeneric,      // generic
		"TLSModelInitialExec":  enum.TLSModelInitialExec,  // initialexec
		"TLSModelLocalDynamic": enum.TLSModelLocalDynamic, // localdynamic
		"TLSModelLocalExec":    enum.TLSModelLocalExec,    // localexec
		// UnnamedAddr specifies whether the address is significant.
		// Unnamed address specifiers.
		"UnnamedAddrNone":             enum.UnnamedAddrNone,             // none
		"UnnamedAddrLocalUnnamedAddr": enum.UnnamedAddrLocalUnnamedAddr, // local_unnamed_addr
		"UnnamedAddrUnnamedAddr":      enum.UnnamedAddrUnnamedAddr,      // unnamed_addr
		// Visibility specifies the visibility of a global identifier.
		// Visibility kinds.
		"VisibilityNone":      enum.VisibilityNone,      // none
		"VisibilityDefault":   enum.VisibilityDefault,   // default
		"VisibilityHidden":    enum.VisibilityHidden,    // hidden
		"VisibilityProtected": enum.VisibilityProtected, // protected
	},

	"Callgraph": callgraph,
	// Eval executes the named function and returns its result (kept for compatibility).
	"Eval": func(m *ir.Module, start string) uint32 {
		return run(m, start, "").Ret
	},
	// Run executes the named function with the given stdin content for getchar() and
	// returns {Ret, Out}. The input parameter can be left out.
	"Run": run,
	// RunJS executes a MetaJS module (IR emitted by metajs-to-llvm-ir.abnf, where every
	// value is an i64 handle and the js_* externals implement the semantics). The
	// named function is the module entry (usually "jsmain"); its i64 handle result
	// is converted to an int32 and returned as Ret.
	"RunJS": runJSModule,
	// BuildExecutable writes the module as textual LLVM IR to a temp file and invokes
	// clang to link a native executable at outPath. Returns "" on success or a
	// human-readable error string. Driven by the -exe flag (c.exePath) from a compiler
	// grammar, so `mec <compiler> <source> -exe out` turns the source into a real binary.
	//
	// The optional third argument is the grammar's own runtime: extra files clang links
	// with the module (.c/.ll/.o/.a). It is merged with the -rt flag (c.runtime), and
	// -L/-l reach clang too. Supplying any of them switches OFF the zero-stubbing of
	// undefined symbols - see buildExecutable.
	"BuildExecutable": buildExecutable,
}

// libcExterns are the external functions clang resolves from the real C runtime, so
// they must NOT be stubbed; every OTHER declared-but-undefined function is a
// user/language function with no linkable symbol (e.g. a lisp form that references a
// never-defined name inside a short-circuit) and would make clang fail at link time.
//
// This list was `{putchar, getchar, puts, abs}` until 2026-08-02, which meant that
// `malloc` got a null-returning STUB: a C program that allocated built successfully
// under -exe and segfaulted on its first field write, while the identical module
// handed straight to clang printed the right answer. Same failure class as bash's
// rt_regex_search - a missing symbol became a wrong answer instead of a link error.
//
// THE BAR FOR ADDING A NAME HERE is that the C model can implement it CORRECTLY, not
// merely consistently. Two conditions, both required:
//
//  1. clang's real symbol must answer what cc answers for the same source, given the
//     way languages/c-to-llvm-ir.abnf lays memory out.
//  2. libcNative (below) must implement it too, so `llvm.Run` and the clang-built
//     binary agree. A name listed for clang ALONE reintroduces exactly the divergence
//     -exe exists to close.
//
// A name that fails either condition is deliberately left OFF, which makes it loud on
// both sides: `llvm.Run` panics naming the function (externHandler) and -exe warns on
// stderr naming the symbol before linking a zero stub (stubUndefined). Loud-and-absent
// beats listed-and-wrong every time; a silently wrong answer is the single defect class
// this project most wants surfaced, and byte-identity between the two engines is exactly
// the invariant that cannot see it.
//
// HISTORY, kept short so that nobody re-derives it. Until 2026-08-02 the whole str*/mem*
// family, atoi/atol and strdup were deliberately absent, because languages/c-to-llvm-ir.abnf
// gave `char` a FOUR-byte cell: "hello" was emitted as `[6 x i32]`, so a byte-oriented
// strlen stopped at the first padding NUL and answered 1 where cc answers 5, and atoi("42")
// read '4', 0 and answered 4 where cc answers 42. Listing them then made BOTH engines agree
// on a wrong answer, which is strictly worse than both failing loudly, so 43cf14e reverted
// it. Commit 93c814f removed that obstacle - a `char` is now a real i8 cell and a string
// literal a real `[n x i8]` - and the family is listed below, RE-MEASURED against cc rather
// than taken over from the earlier report.
//
// WHERE THE EVIDENCE IS. Condition 2 is pinned by abnf/libcnative_test.go, a differential
// test of all 18 against Go's own strings/bytes/strconv. Condition 1 was measured by hand
// against real cc (Apple clang 21.0.0, arm64-darwin) over a 49-line probe covering every
// name: cc, `llvm.Run` and the clang-built -exe binary printed the same 49 lines. It is
// NOT in tests/c-test-full.c, and that is deliberate - the ratchet file is run by BOTH
// halves of the language and languages/c-interpreter.abnf has no standard library at all,
// so a section calling these names takes that half from FULL to a reported gap and the
// two halves are then reported as disagreeing. The measurement is written down in the
// header of tests/c-test-full.c so nobody re-derives it.
//
// Still deliberately absent, and still for condition 2, NOT for the memory model: the
// varargs printf family (printf, fprintf, sprintf, snprintf, fwrite, fputs, putc, fputc),
// exit, abort, qsort and bsearch. See libcNative for why the interpreter cannot implement
// each of them; clang could link them, and that is precisely the divergence condition 2
// forbids. Re-confirmed 2026-08-02: the interpreter still has no varargs ABI (arguments
// reach a native already flattened to uint64 with no type tags), still no unwind path out
// of ma.call for exit/abort, and a C function pointer is still a funcId in an i32 rather
// than an address, so qsort/bsearch cannot call back through one.
var libcExterns = map[string]bool{
	// stdio / stdlib character and integer primitives, all pre-existing.
	"putchar": true, "getchar": true, "puts": true, "abs": true,
	// The allocator. Address-based, so even the old four-byte `char` cell did not reach
	// it: sizes come from sizeof over structs and wide integers, which ctBytes() and
	// machBytes() agree on. This is the family Phase 0 exists for.
	"malloc": true, "calloc": true, "realloc": true, "free": true,
	// Pure integer arithmetic, so it cannot be affected by the memory model at all.
	"labs": true,
	// The byte/address family, unlocked by the one-byte `char` of 93c814f. Every one is
	// pure work over the flat arena, so libcNative implements it exactly (condition 2)
	// and clang's real symbol sees the same bytes at the same addresses (condition 1).
	// All 18 are pinned by abnf/libcnative_test.go; see the note above for why they are
	// not in tests/c-test-full.c.
	"strlen": true, "strcmp": true, "strncmp": true,
	"strcpy": true, "strncpy": true, "strcat": true, "strncat": true,
	"strchr": true, "strrchr": true, "strstr": true,
	"memcpy": true, "memmove": true, "memset": true, "memcmp": true, "memchr": true,
	"atoi": true, "atol": true, "strdup": true,
	// CAVEAT, measured: for strcmp/strncmp/memcmp the magnitude is unspecified by C and
	// Apple's libc returns the UNSIGNED byte difference (strcmp("xy","xc") == 22), which
	// is what byteDiff reproduces. Only the SIGN is portable, so only the sign is
	// asserted. See byteDiff for the constant-folding trap that goes with it.
}

// undefinedSymbols lists, sorted, every function the module declares but neither
// defines nor expects from libc - i.e. exactly the set stubUndefined would stub and
// the set a linked runtime is supposed to provide.
func undefinedSymbols(m *ir.Module) []string {
	var names []string
	for _, f := range m.Funcs {
		if len(f.Blocks) > 0 || libcExterns[f.Name()] {
			continue
		}
		names = append(names, f.Name())
	}
	sort.Strings(names)
	return names
}

// stubUndefined gives every declared-but-undefined non-libc function a trivial body
// (return the zero value of its result type) so the module links, and WARNS on stderr
// naming each one. The stub exists for a language-level function that has no linkable
// symbol at all - a lisp form referencing a never-defined name inside a short-circuit
// is the case it was written for - and reaching one at run time yields a zero that no
// diagnostic would otherwise explain, so the warning is the only thing standing between
// a missing symbol and a silently wrong answer.
//
// It is skipped entirely when the build links a runtime (see buildExecutable): there
// the same symbol is a genuine link error, and papering over it with a zero would
// reintroduce the malloc-returns-null failure class phase 0 removed.
func stubUndefined(m *ir.Module) {
	for _, f := range m.Funcs {
		if len(f.Blocks) > 0 || libcExterns[f.Name()] {
			continue
		}
		fmt.Fprintln(warnWriter, "warning: no definition for "+f.Name()+
			"; linking a stub that returns zero, so a call to it answers 0/null")
		blk := f.NewBlock("")
		switch rt := f.Sig.RetType.(type) {
		case *types.IntType:
			blk.NewRet(constant.NewInt(rt, 0))
		case *types.PointerType:
			blk.NewRet(constant.NewNull(rt))
		case *types.VoidType:
			blk.NewRet(nil)
		default:
			blk.NewRet(constant.NewZeroInitializer(rt))
		}
	}
}

// linkInputs is the runtime the build links with the module: what the grammar passed
// to llvm.BuildExecutable, then what -rt added, in that order and without duplicates.
// Both sources compose on purpose - a grammar that knows where its own runtime lives
// should not stop a user from adding a second object file to the same link.
func linkInputs(fromGrammar []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]string{fromGrammar, RuntimeInputs} {
		for _, p := range list {
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// mentionsSymbol reports whether the linker's diagnostics name this symbol. Both
// spellings occur: ld64 prints `"_js_add", referenced from:` and GNU ld prints
// "undefined reference to `js_add'", so instead of parsing either format the check
// asks whether the name appears delimited by non-identifier characters - the C
// leading underscore included, which is why '_' does not count as a left delimiter
// only when it is the mangling prefix.
func mentionsSymbol(out, name string) bool {
	for i := 0; ; {
		j := strings.Index(out[i:], name)
		if j < 0 {
			return false
		}
		s := i + j
		e := s + len(name)
		leftOK := s == 0 || !isSymByte(out[s-1]) || (out[s-1] == '_' && (s == 1 || !isSymByte(out[s-2])))
		rightOK := e == len(out) || !isSymByte(out[e])
		if leftOK && rightOK {
			return true
		}
		i = s + 1
	}
}

func isSymByte(b byte) bool {
	return b == '_' || b == '$' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// duplicateSymbols returns the names a link failure blamed for being defined MORE
// THAN ONCE, sorted and deduplicated, or nil when the failure was not of that kind.
//
// It exists because the undefined-symbol report below is built by filtering the
// module's declared-but-undefined names by whether the linker text MENTIONS them -
// and a duplicate-symbol error mentions exactly the same names, since a name the
// module declares as an extern is a name some input defines. So a build whose real
// problem was two runtime inputs defining `mec_helper` was reported as
// "1 unresolved symbol(s) ... defined by neither the module nor any linked input",
// which is the precise opposite of the truth. PHP hit this; every language that
// links more than one runtime file can.
//
// Both linker spellings are recognised:
//
//	ld64/lld (Darwin):  duplicate symbol '_mec_helper' in:
//	GNU ld:             foo.o:(.text+0x0): multiple definition of `mec_helper'
//
// The names are read out of the LINKER TEXT rather than the module, because a
// duplicate need not appear in the module at all (two runtime inputs can collide
// over a name the program never calls). Only the quoted symbol is taken, never the
// object-file paths around it, so the report stays free of temp file names and this
// path can keep comparing bytes.
func duplicateSymbols(out string) []string {
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		var rest string
		switch {
		case strings.Contains(line, "duplicate symbol"):
			rest = line[strings.Index(line, "duplicate symbol")+len("duplicate symbol"):]
		case strings.Contains(line, "multiple definition of"):
			rest = line[strings.Index(line, "multiple definition of")+len("multiple definition of"):]
		default:
			continue
		}
		if n := quotedSymbol(rest); n != "" {
			seen[n] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// quotedSymbol reads the first 'name', `name' or "name" out of s and strips the
// platform's leading underscore, so both spellings answer the source-level name.
func quotedSymbol(s string) string {
	i := strings.IndexAny(s, "'`\"")
	if i < 0 {
		return ""
	}
	j := strings.IndexAny(s[i+1:], "'\"")
	if j < 0 {
		return ""
	}
	name := s[i+1 : i+1+j]
	if strings.HasPrefix(name, "_") && len(name) > 1 {
		name = name[1:]
	}
	for k := 0; k < len(name); k++ {
		if !isSymByte(name[k]) {
			return ""
		}
	}
	return name
}

// buildExecutable emits m as LLVM IR text and links it into a native executable with
// clang (override with the MEC_CLANG env var), together with the runtime files the
// grammar and -rt supply and the -L/-l libraries. It returns "" on success, else an
// error message the calling grammar prints before exiting non-zero.
//
// TWO MODES, and the difference is what an undefined symbol means:
//
//   - Nothing to link with (no runtime, no -l). The module must stand alone, so a
//     declared-but-undefined function can only be a language-level name that has no
//     symbol anywhere - the lisp form referencing a never-defined name inside a
//     short-circuit. stubUndefined links a zero body and warns. This is the historical
//     behaviour and it is preserved exactly.
//   - A runtime IS declared. Then every remaining declaration is a symbol the runtime
//     was supposed to define, and stubbing it would turn a missing runtime function
//     into an answer of 0/null at run time - the exact failure class phase 0 removed
//     from malloc. So nothing is stubbed, clang links for real, and if the link fails
//     the unresolved names are reported on stderr and the build fails.
//
// The failure report is built from the MODULE (sorted names, filtered by what the
// linker's output mentions), never from the raw linker text: that text carries the
// temp file names, and this path is in the test matrix, which compares bytes.
// rdynamicFlags answers the link flags a generator needs to find its own thread
// entry, and it is a PLATFORM question rather than a preference.
//
// languages/lib/runtime.c's gen_resume takes the address of coro_entry with
// dlsym(dlopen(0), "coro_entry") - see the comment at runtime.c:73, the reason
// being that c-to-llvm-ir.abnf compiles a function NAME to a call and has no way
// to spell a function POINTER, so the address has to come from the loader at run
// time. That lookup only succeeds if coro_entry is in the executable's DYNAMIC
// symbol table:
//
//   - Mach-O (darwin) puts every non-static global symbol of the executable in
//     the symbol table that dlsym searches, so the lookup works with no flag.
//     Passing -rdynamic here would still be ACCEPTED (Apple clang maps it to
//     ld -export_dynamic and warns about nothing - measured), but it is not a
//     no-op: it enlarges the exported symbol table, which changes the binary and
//     therefore its code layout, and this project measures native binaries down
//     to a percent (manual chapter 4). So darwin does not get it.
//   - ELF (linux and the BSDs) exports NOTHING from an executable by default.
//     Without -rdynamic the dlsym answers NULL and gen_resume dies with
//     "coroutines: dlsym of coro_entry failed" at the first generator - i.e. the
//     first native build of any language with a coroutine. That is what this
//     adds.
//   - Anything else (windows/PE) gets nothing: -rdynamic is not a flag clang
//     accepts for a PE target, and no coroutine build is claimed to work there.
//
// Not verifiable on this machine (nothing here builds on linux); the darwin half
// - that the flag is absent and the build is byte-for-byte what it was - is.
//
// Known and NOT handled here, because it is a separate question and would be a
// guess: on a glibc older than 2.34, dlopen/dlsym live in libdl and
// pthread_create in libpthread, so such a system also needs -ldl -lpthread. From
// 2.34 both are folded into libc and -rdynamic alone is enough.
func rdynamicFlags() []string {
	switch runtime.GOOS {
	case "darwin", "windows", "js", "plan9":
		return nil
	default:
		return []string{"-rdynamic"}
	}
}

func buildExecutable(m *ir.Module, outPath string, runtime []string) string {
	inputs := linkInputs(runtime)
	linkingForReal := len(inputs) > 0 || len(LinkLibs) > 0
	if !linkingForReal {
		stubUndefined(m)
	}
	tmp, err := os.CreateTemp("", "mec-*.ll")
	if err != nil {
		return "cannot create temporary .ll file: " + err.Error()
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(m.String()); err != nil {
		tmp.Close()
		return "cannot write .ll file: " + err.Error()
	}
	if err := tmp.Close(); err != nil {
		return "cannot finish .ll file: " + err.Error()
	}
	clangBin := os.Getenv("MEC_CLANG")
	if clangBin == "" {
		clangBin = "clang"
	}
	// -Wno-override-module silences the note that the IR carries no target triple.
	//
	// -O2 is not cosmetic. languages/c-to-llvm-ir.abnf gives every local and every
	// parameter an alloca/store/load and has no mem2reg of its own, and the MetaJS and
	// Lua floors are compiled by it, so the emitted module leans on the backend to
	// promote those slots to registers. Without any -O flag clang does none of it. The
	// effect is measured in docs/runtime-rework-plan.md.
	args := []string{"-Wno-override-module", "-O2", "-o", outPath, tmpName}
	args = append(args, rdynamicFlags()...)
	args = append(args, inputs...)
	for _, d := range LinkDirs {
		args = append(args, "-L"+d)
	}
	for _, l := range LinkLibs {
		args = append(args, "-l"+l)
	}
	out, err := exec.Command(clangBin, args...).CombinedOutput()
	if err != nil {
		// A DUPLICATE symbol is asked about first, and unconditionally: the
		// undefined-symbol report below cannot tell the two apart (see
		// duplicateSymbols), so asking it second would let a duplicate keep being
		// announced as its own opposite.
		if dup := duplicateSymbols(string(out)); len(dup) > 0 {
			fmt.Fprintf(warnWriter, "error: %d symbol(s) are defined MORE THAN ONCE"+
				" among the linked inputs:\n", len(dup))
			for _, name := range dup {
				fmt.Fprintln(warnWriter, "error:     "+name)
			}
			fmt.Fprintln(warnWriter, "error: each name above is defined by two or more of the emitted module and the"+
				"\nerror: linked inputs (c.runtime / -rt, -L, -l). Remove one definition; this is NOT"+
				"\nerror: an unresolved symbol, and adding a definition would make it worse.")
			return "clang could not link the executable: " + strconv.Itoa(len(dup)) +
				" duplicate symbol(s), named on stderr"
		}
		if linkingForReal {
			var missing []string
			for _, name := range undefinedSymbols(m) {
				if mentionsSymbol(string(out), name) {
					missing = append(missing, name)
				}
			}
			if len(missing) > 0 {
				fmt.Fprintf(warnWriter, "error: %d unresolved symbol(s), and this build links a runtime,"+
					" so they are NOT stubbed:\n", len(missing))
				for _, name := range missing {
					fmt.Fprintln(warnWriter, "error:     "+name)
				}
				fmt.Fprintln(warnWriter, "error: each name above is declared by the emitted module and defined by"+
					" neither the module\nerror: nor any linked input (c.runtime / -rt, -L, -l). Implement it in the"+
					" runtime;\nerror: a stub would answer 0/null at run time instead of failing here.")
				return "clang could not link the executable: " + strconv.Itoa(len(missing)) +
					" unresolved symbol(s), named on stderr"
			}
		}
		return "clang could not build the executable (" + err.Error() + "):\n" + string(out)
	}
	return ""
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
// IR interpreter
//
// A small LLVM IR interpreter, so that the IR modules generated by the compiler
// grammars can be executed and verified without an external LLVM installation.
// It supports the integer subset that the grammars in tests/ generate: alloca,
// load, store, getelementptr (into arrays), the integer arithmetic / logic /
// shift / compare instructions, zext / sext / trunc, select, phi, br, condbr,
// ret, direct calls, and the external functions putchar, getchar, puts and abs.
// Not supported: floats, structs, function pointers, and everything else.

// RunResult is what llvm.Run() returns to JS.
type RunResult struct {
	Ret uint32 // The return value of the started function.
	Out string // Everything the program has written via putchar / puts.
}

// machineMaxStepsDefault is the emergency brake against endless loops: the number
// of IR instructions ONE top-level call may execute before the interpreter gives
// up. MaxIRSteps carries the effective value (the -max-steps flag writes it), so a
// legitimately long running program can raise it instead of dying at 1e8.
const machineMaxStepsDefault = 100000000

// MaxIRSteps is the step budget of one top-level IR call, set from the -max-steps
// CLI flag. 0 means "no limit"; newMachine() turns that into a budget nothing can
// reach, so the check in run() stays a single comparison.
var MaxIRSteps = machineMaxStepsDefault

// machine holds the state of one IR program run.
type machine struct {
	mem      []byte                // One flat memory arena for globals and allocas. Offset 0 is reserved as null.
	globals  map[*ir.Global]uint64 // The memory offset of every global.
	funcs    map[string]*ir.Func   // All module functions by name (for host initiated calls).
	input    []byte                // The stdin content for getchar().
	inPos    int
	out      strings.Builder // The stdout content written by putchar() / puts().
	steps    int             // The instruction budget of the CURRENT top-level call (reset at depth 0).
	maxSteps int             // The budget's limit, from MaxIRSteps when the machine was made.
	depth    int             // The call nesting inside this machine, for the steps reset.

	// externs resolves calls to declared functions before the built-in ones
	// (putchar etc.) are tried. The JS runtime (jsrt.go) plugs its js_* host
	// functions in here.
	externs map[string]func(args []uint64) uint64

	// externBound is externs resolved per function OBJECT, filled by bindExterns()
	// as soon as the extern table is known and completed lazily for functions that
	// appear later. It exists because llir's f.Name() is not a field read: it runs
	// a strconv.ParseInt over the name to decide whether it needs quoting, and that
	// parse FAILS (allocating a *strconv.NumError plus a copy of the name) for every
	// name that is not a number - i.e. for all of them. With ~1.5 M extern calls per
	// frozen run that was 12 % of all allocated bytes.
	externBound map[*ir.Func]func(args []uint64) uint64

	layouts   map[*ir.Func]*funcLayout       // Decoded program per function, built on first call.
	sizes     map[types.Type]uint64          // Memoized sizeOf: the recursion is paid once per type.
	fieldOffs map[*types.StructType][]uint64 // Memoized field offsets, for gepStep.
	framePool []*frame                       // Frames of finished calls, ready to be used again.

	// Handle reclamation, installed by the handle runtime (jsrt.attach); nil
	// disables it, so the plain IR interpreter is unaffected. See recycle().
	relNew  map[string]bool   // Externs producing a reclaimable handle (js_scope_new, js_arr_new).
	relThru map[string]uint32 // Per extern, the argument positions whose handle it only READS (see jsThroughArgs).
	release func(h uint64)    // Drops the value behind a handle that provably went dead.
	pin     func(h uint64)    // Marks an argument array the callee can keep past the call.

	// heapSize records the byte size handed out by each malloc/calloc/realloc, which
	// is the one thing realloc needs and the flat arena does not otherwise remember.
	// free() is a no-op here on purpose: the arena never reuses memory, so freeing
	// cannot produce a dangling reuse that the native binary would not also produce.
	heapSize map[uint64]uint64
}

// funcLayout is the decoded program of one function: every value the function
// can define - its parameters and the results of its instructions - gets a dense
// slot, so one invocation is a flat register array instead of a map keyed by the
// instruction, and every instruction is decoded ONCE into a small fixed struct
// with an opcode, a destination slot and pre-resolved operands. Before that the
// interpreter walked the llir data structure itself: a type switch over an
// interface per instruction, a hash lookup per operand (valueOf), a big.Int read
// per constant use and a re-scan for the leading phis on every block entry -
// together ~20 % of a frozen mec run and ~40 % of the interpreter's own time.
//
// The slot numbering is parameters first, then the instructions of the blocks in
// order; instructions without a result (a store) still take a slot so the
// numbering stays positional. Operands are encoded in one int32: >= 0 is a
// register slot, < 0 is ^index into consts, the function's pool of values that
// are already known when the function is decoded (constants, global addresses).
type funcLayout struct {
	blocks  []dblock // The decoded blocks, in the function's own order.
	consts  []uint64 // Constant pool; operand ^i reads consts[i].
	size    int      // Registers per frame.
	maxArgs int      // Widest call instruction, for the frame's argument buffer.

	// The destination slots of the reclaimable calls (see recycle): their last
	// value dies with the frame, so it is released when the call returns.
	relSlots []int32

	// pinArgs says that this function can keep its argument array past its own
	// return (it declares 'arguments', stores it, returns it, ...), so the array
	// must be pinned when the frame starts. See recycle.
	pinArgs bool
}

// dblock is one decoded basic block: its leading phis (resolved simultaneously),
// its remaining instructions and its terminator.
type dblock struct {
	phis  []dphi
	insts []dinst
	term  dterm
}

// dphi is a decoded phi: one destination slot and one operand per predecessor.
type dphi struct {
	dst  int32
	incs []dphiInc
}

type dphiInc struct {
	pred int32 // Index of the predecessor block, or -2 if it is not a block of this function
	// (-1 is the "no predecessor yet" of the entry block, which must not match).
	x int32 // Operand to take when we came from that block.
}

// The opcodes of the decoded instruction stream.
const (
	dOpAlloca uint8 = iota
	dOpLoad
	dOpStore
	dOpStoreConst // Store of an aggregate constant (array / struct initializer).
	dOpGep
	dOpAdd
	dOpSub
	dOpMul
	dOpUDiv
	dOpSDiv
	dOpURem
	dOpSRem
	dOpAnd
	dOpOr
	dOpXor
	dOpShl
	dOpLShr
	dOpAShr
	dOpICmp
	dOpMask // zext / trunc / ptrtoint: mask the operand to w bits.
	dOpSExt
	dOpCopy // inttoptr / bitcast: the value passes through unchanged.
	dOpSelect
	dOpPhi // A phi behind a non-phi instruction; the block loop handles the leading ones.
	dOpCall
	dOpCallRel // A call whose result is reclaimable: see recycle.
)

// dinst is one decoded instruction. Which fields are used depends on op.
type dinst struct {
	op      uint8
	w, w2   uint8                      // Operand bit width; w2 is the target width of a sext.
	pred    enum.IPred                 // The comparison of an icmp.
	dst     int32                      // Destination register.
	x, y, z int32                      // Operands (z is the false value of a select).
	size    uint64                     // Byte size of a load / store, element size of an alloca or gep.
	args    []int32                    // Call arguments, or the indices of a gep.
	argw    []uint8                    // Bit width of each gep index, for sign extension.
	incs    []dphiInc                  // The incoming values of a non-leading phi.
	callee  *ir.Func                   // The called function.
	fn      func(args []uint64) uint64 // An extern callee's handler, bound on first execution.
	gep     *ir.InstGetElementPtr      // The gep whose index types this instruction walks.
	cst     constant.Constant          // The aggregate constant of an dOpStoreConst.
}

// The terminator kinds of a decoded block.
const (
	tBr uint8 = iota
	tCondBr
	tRet
	tRetVoid
	tUnreachable
)

// dterm is a decoded terminator: a branch to block indices, or a return.
type dterm struct {
	kind uint8
	cond int32 // The condition of a conditional branch, or the returned operand.
	t, f int32 // Target block indices.
}

// layoutOf returns the decoded program of a function, decoding it on first use.
func (ma *machine) layoutOf(f *ir.Func) *funcLayout {
	if l, ok := ma.layouts[f]; ok {
		return l
	}
	l := ma.decode(f)
	ma.layouts[f] = l
	return l
}

// decode numbers the values of a function and translates its instructions into
// the decoded form described at funcLayout.
func (ma *machine) decode(f *ir.Func) *funcLayout {
	l := &funcLayout{}
	slots := make(map[value.Value]int32, len(f.Params)+16)
	for _, p := range f.Params {
		slots[p] = int32(l.size)
		l.size++
	}
	blockIdx := make(map[*ir.Block]int32, len(f.Blocks))
	for i, b := range f.Blocks {
		blockIdx[b] = int32(i)
		for _, inst := range b.Insts {
			if v, ok := inst.(value.Value); ok {
				slots[v] = int32(l.size)
			}
			if call, ok := inst.(*ir.InstCall); ok && len(call.Args) > l.maxArgs {
				l.maxArgs = len(call.Args)
			}
			l.size++
		}
	}

	// pool interns a value that is already known here into the constant pool and
	// returns its operand encoding.
	pool := map[value.Value]int32{}
	opnd := func(v value.Value) int32 {
		if slot, ok := slots[v]; ok {
			return slot
		}
		if o, ok := pool[v]; ok {
			return o
		}
		var val uint64
		switch c := v.(type) {
		case *ir.Global:
			val = ma.globals[c]
		case constant.Constant:
			val = ma.constValue(c)
		default:
			panic("IR interpreter: use of an undefined value: " + v.Ident())
		}
		o := ^int32(len(l.consts))
		l.consts = append(l.consts, val)
		pool[v] = o
		return o
	}
	phiOf := func(phi *ir.InstPhi) []dphiInc {
		incs := make([]dphiInc, 0, len(phi.Incs))
		for _, inc := range phi.Incs {
			pred := int32(-2)
			if b, ok := inc.Pred.(*ir.Block); ok {
				if i, ok := blockIdx[b]; ok {
					pred = i
				}
			}
			incs = append(incs, dphiInc{pred: pred, x: opnd(inc.X)})
		}
		return incs
	}

	l.blocks = make([]dblock, len(f.Blocks))
	slot := int32(len(f.Params))
	for bi, b := range f.Blocks {
		db := &l.blocks[bi]
		// The phis at the top of a block resolve SIMULTANEOUSLY: the block loop
		// reads every incoming value against the predecessor's frame first and
		// assigns afterwards. Resolving them one after another let a phi that
		// reads an earlier phi of the same block see the NEW value, silently
		// miscompiling loop-carried swaps ("%a = phi ...,%b" next to
		// "%b = phi ...,%a").
		numPhis := 0
		for numPhis < len(b.Insts) {
			phi, ok := b.Insts[numPhis].(*ir.InstPhi)
			if !ok {
				break
			}
			db.phis = append(db.phis, dphi{dst: slot + int32(numPhis), incs: phiOf(phi)})
			numPhis++
		}
		slot += int32(numPhis)
		db.insts = make([]dinst, 0, len(b.Insts)-numPhis)
		for _, inst := range b.Insts[numPhis:] {
			db.insts = append(db.insts, ma.decodeInst(inst, slot, opnd, phiOf))
			slot++
		}
		db.term = ma.decodeTerm(b.Term, f, blockIdx, opnd)
	}
	ma.recycle(l, f)
	return l
}

// recycle marks the calls whose result may be reclaimed when the SAME call
// executes again in the SAME frame.
//
// The handle runtime (jsrt.go) never freed anything, and a loop body is a block,
// so a program that runs a loop N times leaves N scopes behind - the dominant
// memory cost of a long-running compiled program. There is no root set to trace:
// handles live in these register arrays and in the arena as bare integers. What
// IS decidable is where a handle can go, because the decoded program below holds
// every operand of the function:
//
//   - the result of a `js_scope_new` (or `js_arr_new`) lives in exactly one
//     register (slots are dense and never shared between two values), so when the
//     machine executes that instruction again in the same frame, whatever the
//     register still holds is the PREVIOUS value of that same site - one loop
//     iteration earlier;
//   - that previous handle is unreachable if every use of the value was an
//     argument position the extern only READS (relThru: declare/get/set on that
//     scope, being the parent of a nested scope - which keeps a Go POINTER and
//     never needs the parent's handle again - pushing onto that array, or being
//     the argument array of a call). Any other use - js_closure capturing it, a
//     store into the arena, a phi, a return, a value argument of any extern -
//     can outlive the frame, and then the instruction is left alone.
//
// The second kind of site is the per-call ARGUMENT ARRAY: every call site emits
// js_arr_new plus a js_arr_push per argument, and the array is dead the moment
// the call returns - with ONE exception, which is why the array's uses are not
// the whole question. js_call hands the array itself to a compiled callee (it is
// the callee's second parameter), and the callee may keep it: the JS, TypeScript
// and MetaJS grammars bind it to 'arguments' in the callee's scope, and a callee
// may return it. That is not visible at the call site, so it is decided in the
// CALLEE, by the same scan: pinArgs is set when parameter slot 1 has any use
// that is not a read-only position, and machine.call then pins the array for
// good (see jsrt.pinArray). The other call externs never leak the array itself -
// js_mcall/js_rmcall/js_supercall reach the callee through rt.call, which boxes
// a fresh array around the ELEMENTS - and host functions receive the element
// slice, never the array.
//
// Over-approximation is safe in both directions: a value wrongly treated as
// escaping is merely not reclaimed, and a wrongly pinned array is merely not
// reclaimed either. The scan below therefore reads x/y/z of EVERY instruction,
// even where a field is a flag rather than an operand.
//
// Reclaiming clears the table entry (the value is collected by Go) but never
// reuses the slot: a reused slot would turn a stale handle into a silently
// different value, a cleared one into a loud "is not a scope".
func (ma *machine) recycle(l *funcLayout, f *ir.Func) {
	if ma.release == nil {
		return
	}
	cand := map[int32]*dinst{}
	for bi := range l.blocks {
		for i := range l.blocks[bi].insts {
			in := &l.blocks[bi].insts[i]
			if in.op == dOpCall && in.callee != nil && len(in.callee.Blocks) == 0 && ma.relNew[in.callee.Name()] {
				cand[in.dst] = in
			}
		}
	}
	// Slot 1 of a two-parameter function is the argument array of the (env, args)
	// calling convention; it is not released here, only asked whether the caller
	// may release it (see pinArgs above).
	argSlot := int32(-1)
	if len(f.Params) == 2 {
		argSlot = 1
	}
	if len(cand) == 0 && argSlot < 0 {
		return
	}
	live := make(map[int32]bool, len(cand)+1)
	for d := range cand {
		live[d] = true
	}
	if argSlot >= 0 {
		live[argSlot] = true
	}
	use := func(o int32) { delete(live, o) }
	for bi := range l.blocks {
		db := &l.blocks[bi]
		for _, ph := range db.phis {
			for _, inc := range ph.incs {
				use(inc.x)
			}
		}
		for i := range db.insts {
			in := &db.insts[i]
			var thru uint32
			if in.op == dOpCall && in.callee != nil && len(in.callee.Blocks) == 0 {
				thru = ma.relThru[in.callee.Name()]
			}
			for ai, a := range in.args {
				if ai < 32 && thru&(1<<uint(ai)) != 0 {
					continue // A position the extern only reads.
				}
				use(a)
			}
			if in.op != dOpCall {
				use(in.x)
				use(in.y)
				use(in.z)
			}
			for _, inc := range in.incs {
				use(inc.x)
			}
		}
		use(db.term.cond)
	}
	l.pinArgs = argSlot >= 0 && !live[argSlot]
	// In instruction order, not map order: the release order at frame exit stays
	// the same from run to run.
	for bi := range l.blocks {
		for i := range l.blocks[bi].insts {
			in := &l.blocks[bi].insts[i]
			if live[in.dst] && cand[in.dst] == in {
				in.op = dOpCallRel
				l.relSlots = append(l.relSlots, in.dst)
			}
		}
	}
}

// decodeInst translates one non-leading instruction. dst is the register its
// result goes into.
func (ma *machine) decodeInst(inst ir.Instruction, dst int32, opnd func(value.Value) int32, phiOf func(*ir.InstPhi) []dphiInc) dinst {
	d := dinst{dst: dst}
	binary := func(op uint8, x, y value.Value) dinst {
		d.op, d.x, d.y, d.w = op, opnd(x), opnd(y), wbits(x.Type())
		return d
	}
	switch inst := inst.(type) {
	case *ir.InstAlloca:
		d.op, d.size = dOpAlloca, ma.sizeOf(inst.ElemType)
		if inst.NElems != nil {
			d.x, d.y = opnd(inst.NElems), 1 // y != 0 means "the element count is in x".
		}
	case *ir.InstLoad:
		d.op, d.x, d.size = dOpLoad, opnd(inst.Src), ma.sizeOf(inst.ElemType)
	case *ir.InstStore:
		// Aggregate constants (e.g. zeroinitializer of an array or struct) are written as a whole.
		if c, ok := inst.Src.(constant.Constant); ok {
			switch inst.Src.Type().(type) {
			case *types.ArrayType, *types.StructType:
				d.op, d.x, d.cst = dOpStoreConst, opnd(inst.Dst), c
				return d
			}
		}
		d.op, d.x, d.y, d.size = dOpStore, opnd(inst.Dst), opnd(inst.Src), ma.sizeOf(inst.Src.Type())
	case *ir.InstGetElementPtr:
		d.op, d.x, d.size, d.gep = dOpGep, opnd(inst.Src), ma.sizeOf(inst.ElemType), inst
		for _, index := range inst.Indices {
			d.args = append(d.args, opnd(index))
			// Remember how wide the index was computed, so the interpreter can
			// sign-extend it below: a gep index is SIGNED, and '*(e - 1)'
			// naturally emits a negative i32.
			w := uint8(64)
			if it, ok := index.Type().(*types.IntType); ok && it.BitSize < 64 {
				w = uint8(it.BitSize)
			}
			d.argw = append(d.argw, w)
		}
	case *ir.InstAdd:
		return binary(dOpAdd, inst.X, inst.Y)
	case *ir.InstSub:
		return binary(dOpSub, inst.X, inst.Y)
	case *ir.InstMul:
		return binary(dOpMul, inst.X, inst.Y)
	case *ir.InstUDiv:
		return binary(dOpUDiv, inst.X, inst.Y)
	case *ir.InstSDiv:
		return binary(dOpSDiv, inst.X, inst.Y)
	case *ir.InstURem:
		return binary(dOpURem, inst.X, inst.Y)
	case *ir.InstSRem:
		return binary(dOpSRem, inst.X, inst.Y)
	case *ir.InstAnd:
		return binary(dOpAnd, inst.X, inst.Y)
	case *ir.InstOr:
		return binary(dOpOr, inst.X, inst.Y)
	case *ir.InstXor:
		return binary(dOpXor, inst.X, inst.Y)
	case *ir.InstShl:
		return binary(dOpShl, inst.X, inst.Y)
	case *ir.InstLShr:
		return binary(dOpLShr, inst.X, inst.Y)
	case *ir.InstAShr:
		return binary(dOpAShr, inst.X, inst.Y)
	case *ir.InstICmp:
		d = binary(dOpICmp, inst.X, inst.Y)
		d.pred = inst.Pred
	case *ir.InstZExt:
		d.op, d.x, d.w = dOpMask, opnd(inst.From), wbits(inst.From.Type())
	case *ir.InstSExt:
		d.op, d.x = dOpSExt, opnd(inst.From)
		d.w, d.w2 = wbits(inst.From.Type()), wbits(inst.To)
	case *ir.InstTrunc:
		d.op, d.x, d.w = dOpMask, opnd(inst.From), wbits(inst.To)
	case *ir.InstIntToPtr:
		// Addresses are plain offsets into the arena; the width already fits.
		d.op, d.x = dOpCopy, opnd(inst.From)
	case *ir.InstPtrToInt:
		d.op, d.x, d.w = dOpMask, opnd(inst.From), wbits(inst.To)
	case *ir.InstBitCast:
		d.op, d.x = dOpCopy, opnd(inst.From)
	case *ir.InstSelect:
		d.op, d.x, d.y, d.z = dOpSelect, opnd(inst.Cond), opnd(inst.ValueTrue), opnd(inst.ValueFalse)
	case *ir.InstPhi:
		// Only reached for a phi BEHIND a non-phi instruction (invalid LLVM,
		// kept sequential for compatibility).
		d.op, d.incs = dOpPhi, phiOf(inst)
	case *ir.InstCall:
		callee, ok := inst.Callee.(*ir.Func)
		if !ok {
			panic(fmt.Sprintf("IR interpreter: unsupported callee type %T (function pointers are not supported)", inst.Callee))
		}
		d.op, d.callee = dOpCall, callee
		for _, a := range inst.Args {
			d.args = append(d.args, opnd(a))
		}
	default:
		panic(fmt.Sprintf("IR interpreter: unsupported instruction type %T", inst))
	}
	return d
}

// decodeTerm translates one terminator; branch targets become block indices.
func (ma *machine) decodeTerm(term ir.Terminator, f *ir.Func, blockIdx map[*ir.Block]int32, opnd func(value.Value) int32) dterm {
	switch term := term.(type) {
	case *ir.TermBr:
		return dterm{kind: tBr, t: blockIdx[term.Target.(*ir.Block)]}
	case *ir.TermCondBr:
		return dterm{kind: tCondBr, cond: opnd(term.Cond),
			t: blockIdx[term.TargetTrue.(*ir.Block)], f: blockIdx[term.TargetFalse.(*ir.Block)]}
	case *ir.TermRet:
		if term.X == nil {
			return dterm{kind: tRetVoid}
		}
		return dterm{kind: tRet, cond: opnd(term.X)}
	case *ir.TermUnreachable:
		return dterm{kind: tUnreachable}
	default:
		panic(fmt.Sprintf("IR interpreter: unsupported terminator %T in %s", term, f.Ident()))
	}
}

// frame holds the SSA values of one function invocation, one register per slot
// of the function's layout. Frames are pooled per machine: acquire clears the
// registers, so a value that the function never assigned reads as 0 instead of
// leaking the previous call's register.
type frame struct {
	regs   []uint64 // The registers, indexed by layout slot.
	args   []uint64 // Argument buffer shared by all call instructions of this frame.
	layout *funcLayout
}

// rd reads one decoded operand: a register, or a value of the constant pool.
func (fr *frame) rd(o int32) uint64 {
	if o >= 0 {
		return fr.regs[o]
	}
	return fr.layout.consts[^o]
}

func (ma *machine) acquireFrame(l *funcLayout) *frame {
	if n := len(ma.framePool); n > 0 {
		fr := ma.framePool[n-1]
		ma.framePool = ma.framePool[:n-1]
		if cap(fr.regs) >= l.size {
			fr.regs = fr.regs[:l.size]
			for i := range fr.regs {
				fr.regs[i] = 0
			}
		} else {
			fr.regs = make([]uint64, l.size)
		}
		if cap(fr.args) >= l.maxArgs {
			fr.args = fr.args[:l.maxArgs]
		} else {
			fr.args = make([]uint64, l.maxArgs)
		}
		fr.layout = l
		return fr
	}
	return &frame{regs: make([]uint64, l.size), args: make([]uint64, l.maxArgs), layout: l}
}

func (ma *machine) releaseFrame(fr *frame) {
	ma.framePool = append(ma.framePool, fr)
}

// stepLimitMsg names the safety valve that just tripped. The limit is not a
// property of the program, so the message says what it is and how to lift it -
// a long but finite run used to abort with a text that read like a grammar bug.
func (ma *machine) stepLimitMsg() string {
	return fmt.Sprintf("IR interpreter: step limit exceeded - one call ran more than %d instructions.\n"+
		"This is a safety valve against endless loops, not a limit of the language: if the program\n"+
		"legitimately runs that long, raise it with -max-steps N (or -max-steps 0 for no limit).", ma.maxSteps)
}

// newMachine loads a module into a fresh machine: it allocates and initializes
// the globals and indexes the functions.
func newMachine(m *ir.Module, input string) *machine {
	ma := &machine{
		mem:         make([]byte, 8), // Offset 0 stays unused, so 0 can act as null pointer.
		globals:     map[*ir.Global]uint64{},
		funcs:       map[string]*ir.Func{},
		externBound: map[*ir.Func]func(args []uint64) uint64{},
		layouts:     map[*ir.Func]*funcLayout{},
		sizes:       map[types.Type]uint64{},
		fieldOffs:   map[*types.StructType][]uint64{},
		input:       []byte(input),
		maxSteps:    MaxIRSteps,
	}
	if ma.maxSteps <= 0 { // -max-steps 0: no limit, expressed as one nothing can reach.
		ma.maxSteps = math.MaxInt64
	}
	for _, g := range m.Globals {
		off := ma.alloc(ma.sizeOf(g.ContentType))
		ma.globals[g] = off
		if g.Init != nil {
			ma.writeConst(off, g.Init)
		}
	}
	for _, f := range m.Funcs {
		ma.funcs[f.Name()] = f
	}
	ma.bindExterns() // Only the built-ins here; jsrt.attach() binds again once its host table is set.
	return ma
}

// callByName executes a module function with the given arguments.
func (ma *machine) callByName(name string, args []uint64) uint64 {
	f, ok := ma.funcs[name]
	if !ok {
		panic("IR interpreter: function not found in module: " + name)
	}
	return ma.call(f, args)
}

// run executes the function with the given name inside the module and returns
// its return value together with the produced output.
// The optional fourth argument is the grammar's runtime, the same list it hands to
// llvm.BuildExecutable: every .ll in it is parsed and linked into this run, so a
// language whose runtime lives in a separately compiled module (lib/batch-rt.ll)
// answers identically under the interpreter and as a clang-built binary. See
// abnf/llvmlink.go.
func run(m *ir.Module, start string, input string, runtime ...string) *RunResult {
	maybeDumpCFG(m)
	maybeDumpCallgraph(m)
	ma := newMachine(m, input)
	if len(runtime) > 0 {
		ma.linkRuntimeModules(m, runtime)
	}
	f, ok := ma.funcs[start]
	if !ok {
		panic("llvm.Run(): function not found in module: " + start)
	}
	// The start function is called with zero values for all of its parameters
	// (e.g. a main(int argc) still runs).
	ret := ma.call(f, make([]uint64, len(f.Params)))
	return &RunResult{Ret: uint32(ret), Out: ma.out.String()}
}

// alloc reserves size bytes of zeroed memory and returns their offset.
func (ma *machine) alloc(size uint64) uint64 {
	off := uint64(len(ma.mem))
	ma.mem = append(ma.mem, make([]byte, size)...)
	return off
}

// sizeOf returns the storage size of a type in bytes. Struct fields lie back to
// back (packed layout): the interpreter defines its own ABI, real alignment
// padding does not exist here.
func (ma *machine) sizeOf(t types.Type) uint64 {
	if n, ok := ma.sizes[t]; ok {
		return n
	}
	n := ma.sizeOfUncached(t)
	ma.sizes[t] = n
	return n
}

// sizeOfUncached is the recursion behind sizeOf; every step of it goes through
// the cache, so a nested type is measured once and not once per enclosing type.
func (ma *machine) sizeOfUncached(t types.Type) uint64 {
	switch t := t.(type) {
	case *types.IntType:
		return (t.BitSize + 7) / 8
	case *types.ArrayType:
		return t.Len * ma.sizeOf(t.ElemType)
	case *types.PointerType:
		return 8
	case *types.StructType:
		var size uint64
		for _, f := range t.Fields {
			size += ma.sizeOf(f)
		}
		return size
	default:
		panic("IR interpreter: unsupported type: " + t.LLString())
	}
}

// gepStep resolves one inner getelementptr index: it returns the byte offset of
// the selected element and the type to continue with. Arrays step by the element
// size, struct indices select a field (packed layout, see sizeOf).
// sextIdx sign-extends a gep index that was computed at bits wide to 64 bits.
// Registers hold values zero-extended, so a negative index arrives with its
// high bits clear; without this the subsequent unsigned multiply produces a
// wild offset rather than a step backwards.
func sextIdx(v uint64, bits uint8) uint64 {
	if bits == 0 || bits >= 64 {
		return v
	}
	if v&(1<<(bits-1)) != 0 {
		return v | ^uint64(0)<<bits
	}
	return v
}

func (ma *machine) gepStep(t types.Type, idx uint64) (uint64, types.Type) {
	switch t := t.(type) {
	case *types.ArrayType:
		return idx * ma.sizeOf(t.ElemType), t.ElemType
	case *types.StructType:
		offs, ok := ma.fieldOffs[t]
		if !ok {
			offs = make([]uint64, len(t.Fields))
			var off uint64
			for i, f := range t.Fields {
				offs[i] = off
				off += ma.sizeOf(f)
			}
			ma.fieldOffs[t] = offs
		}
		return offs[idx], t.Fields[idx]
	default:
		panic("IR interpreter: getelementptr only supports arrays and structs, not " + t.LLString())
	}
}

// wbits is widthOf as a decoded operand width. Everything at or above 64 bits is
// stored as 64: maskTo and signedOf treat those identically, so the clamp keeps a
// width of e.g. i128 from wrapping around the byte.
func wbits(t types.Type) uint8 {
	if w := widthOf(t); w < 64 {
		return uint8(w)
	}
	return 64
}

// widthOf returns the bit width used for calculations with values of that type.
func widthOf(t types.Type) uint64 {
	switch t := t.(type) {
	case *types.IntType:
		return t.BitSize
	case *types.PointerType:
		return 64
	default:
		panic("IR interpreter: not an integer type: " + t.LLString())
	}
}

// maskTo cuts a value down to the given bit width.
func maskTo(v uint64, bits uint64) uint64 {
	if bits >= 64 {
		return v
	}
	return v & (uint64(1)<<bits - 1)
}

// signedOf interprets the lower bits of v as a signed two's complement number.
func signedOf(v uint64, bits uint64) int64 {
	v = maskTo(v, bits)
	if bits < 64 && v&(uint64(1)<<(bits-1)) != 0 {
		v |= ^(uint64(1)<<bits - 1) // Extend the sign bit.
	}
	return int64(v)
}

// load reads a little endian value of the given byte size from memory.
func (ma *machine) load(addr uint64, size uint64) uint64 {
	if addr == 0 || addr+size > uint64(len(ma.mem)) {
		panic(fmt.Sprintf("IR interpreter: invalid load of %d bytes at address %d", size, addr))
	}
	b := ma.mem[addr : addr+size]
	switch size {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.LittleEndian.Uint16(b))
	case 4:
		return uint64(binary.LittleEndian.Uint32(b))
	case 8:
		return binary.LittleEndian.Uint64(b)
	default:
		panic(fmt.Sprintf("IR interpreter: unsupported load size %d", size))
	}
}

// store writes a little endian value of the given byte size into memory.
func (ma *machine) store(addr uint64, v uint64, size uint64) {
	if addr == 0 || addr+size > uint64(len(ma.mem)) {
		panic(fmt.Sprintf("IR interpreter: invalid store of %d bytes at address %d", size, addr))
	}
	b := ma.mem[addr : addr+size]
	switch size {
	case 1:
		b[0] = byte(v)
	case 2:
		binary.LittleEndian.PutUint16(b, uint16(v))
	case 4:
		binary.LittleEndian.PutUint32(b, uint32(v))
	case 8:
		binary.LittleEndian.PutUint64(b, v)
	default:
		panic(fmt.Sprintf("IR interpreter: unsupported store size %d", size))
	}
}

// constValue returns the numeric value of a scalar constant.
func (ma *machine) constValue(c constant.Constant) uint64 {
	switch c := c.(type) {
	case *constant.Int:
		if c.X.Sign() < 0 {
			return uint64(c.X.Int64())
		}
		return c.X.Uint64()
	case *constant.Null:
		return 0
	case *ir.Global:
		return ma.globals[c]
	case *constant.Index:
		// The asm parser wraps getelementptr indices into constant.Index.
		return ma.constValue(c.Constant)
	case *constant.ExprGetElementPtr:
		// The same address computation as the getelementptr instruction:
		// the first index scales by the whole element type, the following
		// ones step into arrays and structs.
		off := ma.constValue(c.Src) + ma.constValue(c.Indices[0])*ma.sizeOf(c.ElemType)
		t := c.ElemType
		for _, index := range c.Indices[1:] {
			d, next := ma.gepStep(t, ma.constValue(index))
			off += d
			t = next
		}
		return off
	default:
		panic(fmt.Sprintf("IR interpreter: unsupported constant (%T): %s", c, c.String()))
	}
}

// writeConst writes a (possibly aggregate) constant into memory, e.g. a global initializer.
func (ma *machine) writeConst(off uint64, c constant.Constant) {
	switch c := c.(type) {
	case *constant.Int:
		ma.store(off, ma.constValue(c), ma.sizeOf(c.Typ))
	case *constant.Null:
		ma.store(off, 0, 8)
	case *constant.ZeroInitializer:
		// Fresh arena memory is already zero, but a store of a zeroinitializer
		// must also clear memory that was written before.
		for i := uint64(0); i < ma.sizeOf(c.Typ); i++ {
			ma.mem[off+i] = 0
		}
	case *constant.CharArray:
		copy(ma.mem[off:off+uint64(len(c.X))], c.X)
	case *constant.Array:
		if len(c.Elems) > 0 {
			elemSize := ma.sizeOf(c.Elems[0].Type())
			for i, elem := range c.Elems {
				ma.writeConst(off+uint64(i)*elemSize, elem)
			}
		}
	default:
		panic("IR interpreter: unsupported initializer: " + c.String())
	}
}

// call executes a function with the given argument values and returns its return value.
// Functions without blocks are treated as external functions.
func (ma *machine) call(f *ir.Func, args []uint64) uint64 {
	if len(f.Blocks) == 0 {
		return ma.extern(f, args)
	}
	if len(args) != len(f.Params) {
		panic(fmt.Sprintf("IR interpreter: call of %s with %d arguments, but it has %d parameters", f.Ident(), len(args), len(f.Params)))
	}
	l := ma.layoutOf(f)
	fr := ma.acquireFrame(l)
	for i := range f.Params {
		fr.regs[i] = args[i] // The parameters are the first slots of the layout.
	}
	// This callee can keep its argument array past its own return (it binds it to
	// 'arguments', stores it, returns it, ...), so the array the CALLER built for
	// this call must survive the call site's reclamation. See recycle.
	if l.pinArgs {
		ma.pin(args[1])
	}

	// The step budget is per top-level call, not per machine lifetime:
	// machines are cached per (engine, module) for a whole compile run, so a
	// cumulative count tripped the emergency brake on big legitimate compiles
	// (thousands of runs of one hot tag script's module). The defer keeps the
	// depth right when a js_throw panic unwinds through this frame, and hands
	// the frame back for the next call.
	ma.depth++
	if ma.depth == 1 {
		ma.steps = 0
	}
	defer func() {
		ma.depth--
		// The frame dies here, and with it the last value of every reclaimable
		// call (see recycle) - on a js_throw unwind too, since the frame is gone
		// either way and a value that could have escaped it is not marked.
		for _, s := range l.relSlots {
			if h := fr.regs[s]; h != 0 {
				ma.release(h)
			}
		}
		ma.releaseFrame(fr)
	}()

	bi, prev := int32(0), int32(-1) // The current block and the one we came from (phis need it).
	var phiVals [8]uint64
	for {
		b := &l.blocks[bi]
		// The leading phis of a block resolve SIMULTANEOUSLY: read every incoming
		// value against the predecessor's frame first, then assign (see decode).
		if n := len(b.phis); n > 0 {
			vals := phiVals[:]
			if n > len(vals) {
				vals = make([]uint64, n)
			}
			for i := range b.phis {
				ma.steps++
				if ma.steps > ma.maxSteps {
					panic(ma.stepLimitMsg())
				}
				vals[i] = fr.rd(phiOperand(b.phis[i].incs, prev))
			}
			for i := range b.phis {
				fr.regs[b.phis[i].dst] = vals[i]
			}
		}
		for i := range b.insts {
			ma.steps++
			if ma.steps > ma.maxSteps {
				panic(ma.stepLimitMsg())
			}
			ma.exec(fr, &b.insts[i], prev)
		}
		switch b.term.kind {
		case tBr:
			prev, bi = bi, b.term.t
		case tCondBr:
			if fr.rd(b.term.cond)&1 != 0 {
				prev, bi = bi, b.term.t
			} else {
				prev, bi = bi, b.term.f
			}
		case tRet:
			return fr.rd(b.term.cond)
		case tRetVoid:
			return 0
		default:
			panic("IR interpreter: reached an unreachable terminator in " + f.Ident())
		}
	}
}

// phiOperand picks a phi's incoming operand for the block we came from.
func phiOperand(incs []dphiInc, prev int32) int32 {
	for i := range incs {
		if incs[i].pred == prev {
			return incs[i].x
		}
	}
	panic("IR interpreter: phi has no incoming value for the previous block")
}

// exec executes one decoded instruction inside the given frame. The destination
// register is part of the instruction, so there is no lookup here (see funcLayout).
func (ma *machine) exec(fr *frame, in *dinst, prev int32) {
	switch in.op {
	case dOpCall:
		// The frame's argument buffer is wide enough for every call of this
		// function, and the callee copies the values into its own registers
		// right away - so one buffer per frame does instead of a slice per call.
		args := fr.args[:len(in.args)]
		for i, a := range in.args {
			args[i] = fr.rd(a)
		}
		if len(in.callee.Blocks) == 0 {
			// An extern: its handler is bound to the instruction on first use, so
			// neither the name nor the per-machine handler table is touched again.
			if in.fn == nil {
				in.fn = ma.externHandler(in.callee)
			}
			fr.regs[in.dst] = in.fn(args)
			return
		}
		fr.regs[in.dst] = ma.call(in.callee, args)
	case dOpCallRel:
		// A reclaimable extern call (js_scope_new or js_arr_new, see recycle): the
		// destination register still holds the handle this very instruction
		// produced the last time round the loop, and nothing else can reach it.
		args := fr.args[:len(in.args)]
		for i, a := range in.args {
			args[i] = fr.rd(a)
		}
		if in.fn == nil {
			in.fn = ma.externHandler(in.callee)
		}
		if old := fr.regs[in.dst]; old != 0 {
			ma.release(old)
		}
		fr.regs[in.dst] = in.fn(args)
	case dOpAdd:
		fr.regs[in.dst] = maskTo(fr.rd(in.x)+fr.rd(in.y), uint64(in.w))
	case dOpSub:
		fr.regs[in.dst] = maskTo(fr.rd(in.x)-fr.rd(in.y), uint64(in.w))
	case dOpMul:
		fr.regs[in.dst] = maskTo(fr.rd(in.x)*fr.rd(in.y), uint64(in.w))
	case dOpICmp:
		bits := uint64(in.w)
		if ma.icmp(in.pred, fr.rd(in.x), fr.rd(in.y), bits) {
			fr.regs[in.dst] = 1
		} else {
			fr.regs[in.dst] = 0
		}
	case dOpMask:
		fr.regs[in.dst] = maskTo(fr.rd(in.x), uint64(in.w))
	case dOpCopy:
		fr.regs[in.dst] = fr.rd(in.x)
	case dOpSExt:
		fr.regs[in.dst] = maskTo(uint64(signedOf(fr.rd(in.x), uint64(in.w))), uint64(in.w2))
	case dOpLoad:
		fr.regs[in.dst] = ma.load(fr.rd(in.x), in.size)
	case dOpStore:
		ma.store(fr.rd(in.x), fr.rd(in.y), in.size)
	case dOpStoreConst:
		ma.writeConst(fr.rd(in.x), in.cst)
	case dOpAlloca:
		n := uint64(1)
		if in.y != 0 {
			n = fr.rd(in.x)
		}
		fr.regs[in.dst] = ma.alloc(in.size * n)
	case dOpGep:
		// The first index scales by the whole element type, the following ones step
		// into arrays and structs.
		//
		// Every index is SIGNED, and registers hold values zero-extended to 64 bits,
		// so an i32 -1 sits here as 0x00000000ffffffff. Scaling that unsigned walks
		// off into a wild address instead of stepping one element back - which is
		// what '*(e - 1)' does in C, and what any language with a negative index
		// does. Sign-extending first makes the uint64 arithmetic wrap correctly.
		off := fr.rd(in.x) + sextIdx(fr.rd(in.args[0]), in.argw[0])*in.size
		t := in.gep.ElemType
		for k, index := range in.args[1:] {
			d, next := ma.gepStep(t, sextIdx(fr.rd(index), in.argw[k+1]))
			off += d
			t = next
		}
		fr.regs[in.dst] = off
	case dOpSelect:
		if fr.rd(in.x)&1 != 0 {
			fr.regs[in.dst] = fr.rd(in.y)
		} else {
			fr.regs[in.dst] = fr.rd(in.z)
		}
	case dOpPhi:
		fr.regs[in.dst] = fr.rd(phiOperand(in.incs, prev))
	default:
		fr.regs[in.dst] = ma.binOp(fr, in)
	}
}

// binOp executes the binary integer instructions that are not inlined into exec.
// The result is masked to the width of the operands.
func (ma *machine) binOp(fr *frame, in *dinst) uint64 {
	bits := uint64(in.w)
	x := maskTo(fr.rd(in.x), bits)
	y := maskTo(fr.rd(in.y), bits)
	var r uint64
	switch in.op {
	case dOpUDiv:
		r = x / y
	case dOpSDiv:
		r = uint64(signedOf(x, bits) / signedOf(y, bits))
	case dOpURem:
		r = x % y
	case dOpSRem:
		r = uint64(signedOf(x, bits) % signedOf(y, bits))
	case dOpAnd:
		r = x & y
	case dOpOr:
		r = x | y
	case dOpXor:
		r = x ^ y
	case dOpShl:
		r = x << y
	case dOpLShr:
		r = x >> y
	case dOpAShr:
		r = uint64(signedOf(x, bits) >> y)
	default:
		panic(fmt.Sprintf("IR interpreter: unsupported opcode %d", in.op))
	}
	return maskTo(r, bits)
}

// icmp executes one integer comparison.
func (ma *machine) icmp(pred enum.IPred, x, y uint64, bits uint64) bool {
	x = maskTo(x, bits)
	y = maskTo(y, bits)
	switch pred {
	case enum.IPredEQ:
		return x == y
	case enum.IPredNE:
		return x != y
	case enum.IPredUGT:
		return x > y
	case enum.IPredUGE:
		return x >= y
	case enum.IPredULT:
		return x < y
	case enum.IPredULE:
		return x <= y
	case enum.IPredSGT:
		return signedOf(x, bits) > signedOf(y, bits)
	case enum.IPredSGE:
		return signedOf(x, bits) >= signedOf(y, bits)
	case enum.IPredSLT:
		return signedOf(x, bits) < signedOf(y, bits)
	case enum.IPredSLE:
		return signedOf(x, bits) <= signedOf(y, bits)
	default:
		panic(fmt.Sprintf("IR interpreter: unsupported icmp predicate %v", pred))
	}
}

// externHandler returns the bound handler of one declared function, resolving and
// caching it if it entered the module after bindExterns() ran.
func (ma *machine) externHandler(f *ir.Func) func(args []uint64) uint64 {
	if fn, ok := ma.externBound[f]; ok {
		return fn
	}
	fn := ma.resolveExtern(f)
	if fn == nil {
		panic("IR interpreter: call to undefined external function " + f.Ident())
	}
	ma.externBound[f] = fn
	return fn
}

// extern calls one of the external functions that the compiler grammars use: a host
// function of the attached runtime (the js_* of jsrt.go) or one of the built-in libc like
// ones. The handler is looked up by function OBJECT, not by name - see externBound.
func (ma *machine) extern(f *ir.Func, args []uint64) uint64 {
	return ma.externHandler(f)(args)
}

// bindExterns resolves the handler of every declared (block-less) function of the module
// once, so that no call has to go through the function name any more. It is called after
// the extern table of the machine is in place (see jsrt.attach()).
func (ma *machine) bindExterns() {
	for _, f := range ma.funcs {
		if len(f.Blocks) > 0 {
			continue
		}
		if fn := ma.resolveExtern(f); fn != nil {
			ma.externBound[f] = fn
		}
	}
}

// ----- The C standard library, natively, for the IR interpreter -----
//
// libcExterns (above) is the set clang links against for real, and every name in it is
// implemented here as well - that is condition 2 of the bar documented at libcExterns,
// and it is the only reason -exe can be trusted: a module has to run the same way under
// `llvm.Run` and as a clang-built binary.
//
// The converse also holds: nothing is implemented here that is not in libcExterns.
// Adding an implementation here alone would make `llvm.Run` answer where -exe stubs a
// zero, which is the same divergence pointing the other way.
//
// Names deliberately absent from BOTH, and therefore a loud panic under `llvm.Run`
// naming the function (externHandler) plus a loud stderr warning under -exe:
//   - printf / fprintf / sprintf / snprintf / fwrite / fflush / putc / fputc / fputs:
//     varargs and FILE* streams. The interpreter has no varargs ABI at all - the
//     argument list it receives is already flattened to uint64 with no type tags - so a
//     format string could not be walked correctly, and an approximation that got %ld
//     wrong would be exactly the "wrong answer instead of an error" this change is
//     about. Use putchar/puts under llvm.Run.
//   - exit / abort: the interpreter has no unwind path out of ma.call, so these would
//     have to fake a return, which is not what they mean.
//   - qsort / bsearch: they call back through a function pointer, and in the C model a
//     function pointer is a funcId in an i32, not an address the machine can call.
//   - getenv / time / rand and friends: not reproducible, so never byte-identical.

// memAt bounds-checks a slice of the machine's arena. It returns nil for a null or
// out-of-range address, which the caller treats as "do nothing" rather than panicking -
// the same tolerance rxPtrCStr already applies.
func (ma *machine) memAt(addr, n uint64) []byte {
	if addr == 0 || addr >= uint64(len(ma.mem)) || addr+n > uint64(len(ma.mem)) {
		return nil
	}
	return ma.mem[addr : addr+n]
}

// heapAlloc is malloc: a bump allocation out of the same arena the allocas use. It can
// never answer 0, because newMachine reserves offset 0 as null.
func (ma *machine) heapAlloc(size uint64) uint64 {
	if size == 0 {
		size = 1 // malloc(0) must still return a distinct, non-null pointer.
	}
	addr := ma.alloc(size)
	if ma.heapSize == nil {
		ma.heapSize = map[uint64]uint64{}
	}
	ma.heapSize[addr] = size
	return addr
}

// byteAt reads one byte of the arena, answering 0 (a string terminator) for a null or
// out-of-range address so that a wild pointer stops a scan instead of panicking.
func (ma *machine) byteAt(addr uint64) byte {
	if addr == 0 || addr >= uint64(len(ma.mem)) {
		return 0
	}
	return ma.mem[addr]
}

// cstrEnd answers the address of the NUL that terminates the string at addr, clamped
// to the end of the arena. A null or out-of-range address answers addr itself, so every
// str* native below sees a zero-length string rather than panicking - the same tolerance
// memAt and rxPtrCStr already apply.
func (ma *machine) cstrEnd(addr uint64) uint64 {
	if addr == 0 || addr >= uint64(len(ma.mem)) {
		return addr
	}
	end := addr
	for end < uint64(len(ma.mem)) && ma.mem[end] != 0 {
		end++
	}
	return end
}

// byteDiff is what strcmp/strncmp/memcmp answer for one differing byte pair. C leaves
// the MAGNITUDE unspecified - only the sign is defined - and Apple's libc returns the
// UNSIGNED byte difference, so that is what these return here.
//
// MEASURED 2026-08-02, Apple clang 21.0.0 / arm64-darwin, tests/../tmp probe:
//
//	char a[4]="xy", b[4]="xc";  strcmp(a,b)          ==> 22    ('y'-'c')
//	const char *A="\x80", *B="\x01";  strcmp(A,B)    ==> 127   (0x80-0x01, UNSIGNED)
//	                                  memcmp(A,B,1)  ==> 127
//	strcmp("\x80", "\x01")                           ==> 1     <- constant-folded
//
// The last line is the trap: with LITERAL operands cc folds the call at compile time and
// normalizes to a sign, while the linked symbol returns the raw difference. Returning a
// normalized -1/0/1 here would therefore make `llvm.Run` and the -exe binary disagree on
// exactly the runtime-operand cases, which is what condition 2 at libcExterns forbids.
// Only the SIGN is a property of C, so only the sign is compared against an independent
// oracle in abnf/libcnative_test.go; the magnitudes there are the measured libc ones.
func byteDiff(x, y byte) uint64 {
	return uint64(int64(int(x) - int(y)))
}

// cAtoi is atoi/atol: optional whitespace, optional sign, then decimal digits, stopping
// at the first non-digit. Overflow is undefined in C, so wrapping is as good an answer
// as any; the ratchet asserts only in-range values.
func (ma *machine) cAtoi(addr uint64) int64 {
	s := ma.mem[ma.min(addr):ma.cstrEnd(addr)]
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\v' || s[i] == '\f' || s[i] == '\r') {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var n int64
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		n = n*10 + int64(s[i]-'0')
		i++
	}
	if neg {
		return -n
	}
	return n
}

// min clamps an address into the arena, so that a null or wild pointer yields an empty
// slice instead of a panic.
func (ma *machine) min(addr uint64) uint64 {
	if addr >= uint64(len(ma.mem)) {
		return uint64(len(ma.mem))
	}
	return addr
}

// libcNative returns the interpreter's implementation of one libc name, or nil if the
// name is not one it implements (see the list of exclusions above).
func libcNative(ma *machine, name string) func(args []uint64) uint64 {
	switch name {
	// ----- The byte/address family, unlocked by the one-byte `char` of 93c814f -----
	// Every one of these is pure work over ma.mem (plus heapAlloc for strdup), which is
	// laid out exactly as the clang-built binary lays it out now that a char is a byte.
	case "strlen":
		return func(args []uint64) uint64 { return ma.cstrEnd(args[0]) - args[0] }
	case "strcmp":
		return func(args []uint64) uint64 {
			a, b := args[0], args[1]
			for {
				x, y := ma.byteAt(a), ma.byteAt(b)
				if x != y || x == 0 {
					return byteDiff(x, y)
				}
				a, b = a+1, b+1
			}
		}
	case "strncmp":
		return func(args []uint64) uint64 {
			a, b, n := args[0], args[1], args[2]
			for i := uint64(0); i < n; i++ {
				x, y := ma.byteAt(a+i), ma.byteAt(b+i)
				if x != y || x == 0 {
					return byteDiff(x, y)
				}
			}
			return 0
		}
	case "strcpy":
		return func(args []uint64) uint64 {
			d, s := args[0], args[1]
			n := ma.cstrEnd(s) - s
			if dst := ma.memAt(d, n+1); dst != nil {
				copy(dst, ma.mem[s:s+n])
				dst[n] = 0
			}
			return d
		}
	case "strncpy":
		return func(args []uint64) uint64 {
			d, s, n := args[0], args[1], args[2]
			dst := ma.memAt(d, n)
			if dst == nil {
				return d
			}
			l := ma.cstrEnd(s) - s
			if l > n {
				l = n
			}
			copy(dst, ma.mem[s:s+l])
			for i := l; i < n; i++ { // strncpy PADS the rest with NUL, it does not stop.
				dst[i] = 0
			}
			return d
		}
	case "strcat", "strncat":
		return func(args []uint64) uint64 {
			d, s := args[0], args[1]
			n := ma.cstrEnd(s) - s
			if name == "strncat" && args[2] < n {
				n = args[2] // strncat appends at most n bytes AND always terminates.
			}
			e := ma.cstrEnd(d)
			if dst := ma.memAt(e, n+1); dst != nil {
				copy(dst, ma.mem[s:s+n])
				dst[n] = 0
			}
			return d
		}
	case "strchr", "strrchr":
		return func(args []uint64) uint64 {
			s, c := args[0], byte(args[1])
			end := ma.cstrEnd(s)
			// Both search the terminator too: strchr(s, 0) is the end of the string.
			if name == "strchr" {
				for p := s; p <= end; p++ {
					if ma.byteAt(p) == c {
						return p
					}
				}
				return 0
			}
			for p := end + 1; p > s; {
				p--
				if ma.byteAt(p) == c {
					return p
				}
			}
			return 0
		}
	case "strstr":
		return func(args []uint64) uint64 {
			h, n := args[0], args[1]
			hl, nl := ma.cstrEnd(h)-h, ma.cstrEnd(n)-n
			if nl == 0 {
				return h // The empty needle matches at the front.
			}
			if nl > hl {
				return 0
			}
			for i := uint64(0); i+nl <= hl; i++ {
				if string(ma.mem[h+i:h+i+nl]) == string(ma.mem[n:n+nl]) {
					return h + i
				}
			}
			return 0
		}
	case "memcpy", "memmove":
		return func(args []uint64) uint64 {
			d, s, n := args[0], args[1], args[2]
			if src := ma.memAt(s, n); src != nil {
				if dst := ma.memAt(d, n); dst != nil {
					copy(dst, src) // Go's copy already handles overlap, so memmove is the same.
				}
			}
			return d
		}
	case "memset":
		return func(args []uint64) uint64 {
			d, c, n := args[0], byte(args[1]), args[2]
			if dst := ma.memAt(d, n); dst != nil {
				for i := range dst {
					dst[i] = c
				}
			}
			return d
		}
	case "memcmp":
		return func(args []uint64) uint64 {
			a, b, n := args[0], args[1], args[2]
			for i := uint64(0); i < n; i++ {
				if x, y := ma.byteAt(a+i), ma.byteAt(b+i); x != y {
					return byteDiff(x, y)
				}
			}
			return 0
		}
	case "memchr":
		return func(args []uint64) uint64 {
			s, c, n := args[0], byte(args[1]), args[2]
			for i := uint64(0); i < n; i++ {
				if ma.byteAt(s+i) == c {
					return s + i
				}
			}
			return 0
		}
	case "atoi", "atol":
		return func(args []uint64) uint64 { return uint64(ma.cAtoi(args[0])) }
	case "strdup":
		return func(args []uint64) uint64 {
			s := args[0]
			n := ma.cstrEnd(s) - s
			d := ma.heapAlloc(n + 1) // NOTE: this may GROW ma.mem, so re-slice after it.
			copy(ma.mem[d:d+n], ma.mem[s:s+n])
			ma.mem[d+n] = 0
			return d
		}
	}
	switch name {
	case "malloc":
		return func(args []uint64) uint64 { return ma.heapAlloc(args[0]) }
	case "calloc":
		// The arena is zeroed by alloc(), so calloc is malloc of the product.
		return func(args []uint64) uint64 { return ma.heapAlloc(args[0] * args[1]) }
	case "realloc":
		return func(args []uint64) uint64 {
			old, size := args[0], args[1]
			nw := ma.heapAlloc(size)
			if old != 0 {
				n := ma.heapSize[old]
				if n > size {
					n = size
				}
				if src := ma.memAt(old, n); src != nil {
					copy(ma.mem[nw:nw+n], src)
				}
			}
			return nw
		}
	case "free":
		return func(args []uint64) uint64 { return 0 } // See heapSize: the arena never reuses.
	case "labs":
		return func(args []uint64) uint64 {
			v := int64(args[0])
			if v < 0 {
				v = -v
			}
			return uint64(v)
		}
	}
	return nil
}

// resolveExtern finds the handler for one external function by name, or nil if the name is
// not known at all. The host functions of the attached runtime win over the built-in ones.
// getchar() reads from the input given to run() and returns 0 at its end.
func (ma *machine) resolveExtern(f *ir.Func) func(args []uint64) uint64 {
	name := f.Name()
	if ma.externs != nil {
		if fn, ok := ma.externs[name]; ok {
			return fn
		}
	}
	// The POINTER-based natives (abnf/jsrtregexptr.go). They live here rather than in
	// a runtime's extern table because a grammar with no handle runtime - bash, batch -
	// runs with ma.externs == nil and never reaches one.
	if fn := nativeExtern(ma, name); fn != nil {
		return fn
	}
	if fn := libcNative(ma, name); fn != nil {
		return fn
	}
	switch name {
	case "putchar":
		return func(args []uint64) uint64 {
			ma.out.WriteByte(byte(args[0]))
			return args[0]
		}
	case "getchar":
		return func(args []uint64) uint64 {
			if ma.inPos < len(ma.input) {
				c := ma.input[ma.inPos]
				ma.inPos++
				return uint64(c)
			}
			return 0
		}
	case "puts":
		return func(args []uint64) uint64 {
			addr := args[0]
			for addr < uint64(len(ma.mem)) && ma.mem[addr] != 0 {
				ma.out.WriteByte(ma.mem[addr])
				addr++
			}
			ma.out.WriteByte(0x0a)
			return 0
		}
	case "abs":
		return func(args []uint64) uint64 {
			v := signedOf(args[0], 32)
			if v < 0 {
				v = -v
			}
			return uint64(v)
		}
	}
	return nil
}
