package abnf

import (
	"fmt"
	"strings"

	"./r"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
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
		"FuncAttrNoMerge":                     enum.FuncAttrNoMerge,                     // nomerge
		"FuncAttrNonLazyBind":                 enum.FuncAttrNonLazyBind,                 // nonlazybind
		"FuncAttrNoRecurse":                   enum.FuncAttrNoRecurse,                   // norecurse
		"FuncAttrNoRedZone":                   enum.FuncAttrNoRedZone,                   // noredzone
		"FuncAttrNoReturn":                    enum.FuncAttrNoReturn,                    // noreturn
		"FuncAttrNoSync":                      enum.FuncAttrNoSync,                      // nosync
		"FuncAttrNoUnwind":                    enum.FuncAttrNoUnwind,                    // nounwind
		"FuncAttrNullPointerIsValid":          enum.FuncAttrNullPointerIsValid,          // null_pointer_is_valid
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
		"ParamAttrImmArg":             enum.ParamAttrImmArg,             // immarg
		"ParamAttrInAlloca":           enum.ParamAttrInAlloca,           // inalloca
		"ParamAttrInReg":              enum.ParamAttrInReg,              // inreg
		"ParamAttrNest":               enum.ParamAttrNest,               // nest
		"ParamAttrNoAlias":            enum.ParamAttrNoAlias,            // noalias
		"ParamAttrNoCapture":          enum.ParamAttrNoCapture,          // nocapture
		"ParamAttrNoFree":             enum.ParamAttrNoFree,             // nofree
		"ParamAttrNoMerge":            enum.ParamAttrNoMerge,            // nomerge
		"ParamAttrNonNull":            enum.ParamAttrNonNull,            // nonnull
		"ParamAttrNullPointerIsValid": enum.ParamAttrNullPointerIsValid, // null_pointer_is_valid
		"ParamAttrReadNone":           enum.ParamAttrReadNone,           // readnone
		"ParamAttrReadOnly":           enum.ParamAttrReadOnly,           // readonly
		"ParamAttrReturned":           enum.ParamAttrReturned,           // returned
		"ParamAttrSignExt":            enum.ParamAttrSignExt,            // signext
		"ParamAttrSRet":               enum.ParamAttrSRet,               // sret
		"ParamAttrSwiftError":         enum.ParamAttrSwiftError,         // swifterror
		"ParamAttrSwiftSelf":          enum.ParamAttrSwiftSelf,          // swiftself
		"ParamAttrWriteOnly":          enum.ParamAttrWriteOnly,          // writeonly
		"ParamAttrZeroExt":            enum.ParamAttrZeroExt,            // zeroext
		// Preemption specifies the preemtion of a global identifier.
		// Preemption kinds.
		"PreemptionNone":           enum.PreemptionNone,           // none
		"PreemptionDSOLocal":       enum.PreemptionDSOLocal,       // dso_local
		"PreemptionDSOPreemptable": enum.PreemptionDSOPreemptable, // dso_preemptable
		// ReturnAttr is a return argument attribute.
		// Return argument attributes.
		"ReturnAttrInReg":              enum.ReturnAttrInReg,              // inreg
		"ReturnAttrNoAlias":            enum.ReturnAttrNoAlias,            // noalias
		"ReturnAttrNoMerge":            enum.ReturnAttrNoMerge,            // nomerge
		"ReturnAttrNonNull":            enum.ReturnAttrNonNull,            // nonnull
		"ReturnAttrNullPointerIsValid": enum.ReturnAttrNullPointerIsValid, // null_pointer_is_valid
		"ReturnAttrSignExt":            enum.ReturnAttrSignExt,            // signext
		"ReturnAttrZeroExt":            enum.ReturnAttrZeroExt,            // zeroext
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
