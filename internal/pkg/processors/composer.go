package processors

import (
	"fmt"
	"oak-compiler/internal/pkg/ast"
	"oak-compiler/internal/pkg/ast/bytecode"
	"oak-compiler/internal/pkg/ast/typed"
	"oak-compiler/internal/pkg/common"
	"slices"
)

//TODO: OPTIMIZE: swap-pop*N -> swap-pop(N)

func Compose(
	moduleName ast.QualifiedIdentifier,
	modules map[ast.QualifiedIdentifier]*typed.Module,
	debug bool,
	binary *bytecode.Binary,
) {
	binary.HashString("")

	if slices.Contains(binary.CompiledPaths, moduleName) {
		return
	}

	binary.CompiledPaths = append(binary.CompiledPaths, moduleName)

	m := modules[moduleName]
	for _, depPath := range m.Dependencies {
		Compose(depPath, modules, debug, binary)
	}

	for _, def := range m.Definitions {
		extId := common.MakeExternalIdentifier(m.Name, def.Name)
		binary.FuncsMap[extId] = bytecode.Pointer(len(binary.Funcs))
		binary.Funcs = append(binary.Funcs, bytecode.Func{})
	}

	for _, def := range m.Definitions {
		pathId := common.MakeExternalIdentifier(m.Name, def.Name)

		ptr := binary.FuncsMap[pathId]
		if binary.Funcs[ptr].Ops == nil {
			binary.Funcs[ptr] = composeDefinition(def, pathId, binary)
			if !def.Hidden || debug {
				binary.Exports[pathId] = ptr
			}
		}
	}
}

func composeDefinition(def *typed.Definition, pathId ast.ExternalIdentifier, binary *bytecode.Binary) bytecode.Func {
	var ops []bytecode.Op
	var locations []ast.Location

	if nc, ok := def.Expression.(*typed.NativeCall); ok && pathId == nc.Name {
		ops, locations = call(string(nc.Name), len(nc.Args), nc.Location, ops, locations, binary)
	} else {
		for i := len(def.Params) - 1; i >= 0; i-- {
			p := def.Params[i]
			ops, locations = composePattern(p, ops, locations, binary)
			ops, locations = match(0, p.GetLocation(), ops, locations)
			ops, locations = swapPop(p.GetLocation(), bytecode.SwapPopModePop, ops, locations)
		}
		ops, locations = composeExpression(def.Expression, ops, locations, binary)
	}

	return bytecode.Func{
		NumArgs:   uint32(len(def.Params)),
		Ops:       ops,
		FilePath:  def.Location.FilePath,
		Locations: locations,
	}
}

func composeExpression(
	expr typed.Expression, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	switch expr.(type) {
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			for _, arg := range e.Args {
				ops, locations = composeExpression(arg, ops, locations, binary)
			}
			ops, locations = call(string(e.Name), len(e.Args), e.Location, ops, locations, binary)
			break
		}
	case *typed.Apply:
		{
			e := expr.(*typed.Apply)
			for _, arg := range e.Args {
				ops, locations = composeExpression(arg, ops, locations, binary)
			}
			ops, locations = composeExpression(e.Func, ops, locations, binary)
			ops, locations = apply(len(e.Args), e.Location, ops, locations)
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			v := e.Value
			if iv, ok := v.(ast.CInt); ok {
				if ex, ok := e.Type.(*typed.TExternal); ok && ex.Name == common.OakCoreBasicsFloat {
					v = ast.CFloat{Value: float64(iv.Value)}
				}
			}
			ops, locations = loadConstValue(v, bytecode.StackKindObject, e.Location, ops, locations, binary)
			break
		}
	case *typed.Local:
		{
			e := expr.(*typed.Local)
			ops, locations = loadLocal(string(e.Name), e.Location, ops, locations, binary)
			break
		}
	case *typed.Global:
		{
			e := expr.(*typed.Global)
			id := common.MakeExternalIdentifier(e.ModuleName, e.DefinitionName)
			funcIndex, ok := binary.FuncsMap[id]
			if !ok {
				panic(common.SystemError{Message: fmt.Sprintf("global definition `%v` not found", id)})
			}
			ops, locations = loadGlobal(funcIndex, e.Location, ops, locations)
			break
		}
	case *typed.Access:
		{
			e := expr.(*typed.Access)
			ops, locations = composeExpression(e.Record, ops, locations, binary)
			ops, locations = access(string(e.FieldName), e.Location, ops, locations, binary)
			break
		}

	case *typed.List:
		{
			e := expr.(*typed.List)
			for _, item := range e.Items {
				ops, locations = composeExpression(item, ops, locations, binary)
			}
			ops, locations = makeObject(bytecode.ObjectKindList, len(e.Items), e.Location, ops, locations)
			break
		}
	case *typed.Tuple:
		{
			e := expr.(*typed.Tuple)
			for _, item := range e.Items {
				ops, locations = composeExpression(item, ops, locations, binary)
			}
			ops, locations = makeObject(bytecode.ObjectKindTuple, len(e.Items), e.Location, ops, locations)
			break
		}
	case *typed.Record:
		{
			e := expr.(*typed.Record)
			for _, f := range e.Fields {
				ops, locations = composeExpression(f.Value, ops, locations, binary)
				ops, locations = loadConstValue(ast.CString{Value: string(f.Name)}, bytecode.StackKindObject, f.Location,
					ops, locations, binary)
			}
			ops, locations = makeObject(bytecode.ObjectKindRecord, len(e.Fields), e.Location, ops, locations)
			break
		}
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			for _, arg := range e.Args {
				ops, locations = composeExpression(arg, ops, locations, binary)
			}
			ops, locations = loadConstValue(ast.CString{Value: string(e.OptionName)}, bytecode.StackKindObject, e.Location,
				ops, locations, binary)
			ops, locations = makeObject(bytecode.ObjectKindData, len(e.Args), e.Location, ops, locations)
			break
		}
	case *typed.UpdateLocal:
		{
			e := expr.(*typed.UpdateLocal)
			ops, locations = loadLocal(string(e.RecordName), e.Location, ops, locations, binary)

			for _, f := range e.Fields {
				ops, locations = composeExpression(f.Value, ops, locations, binary)
				ops, locations = update(string(f.Name), f.Location, ops, locations, binary)
			}
			break
		}
	case *typed.UpdateGlobal:
		{
			e := expr.(*typed.UpdateGlobal)
			id := common.MakeExternalIdentifier(e.ModuleName, e.DefinitionName)
			ops, locations = loadGlobal(binary.FuncsMap[id], e.Location, ops, locations)

			for _, f := range e.Fields {
				ops, locations = composeExpression(f.Value, ops, locations, binary)
				ops, locations = update(string(f.Name), f.Location, ops, locations, binary)
			}
			break
		}
	case *typed.Let:
		{
			e := expr.(*typed.Let)
			ops, locations = composeExpression(e.Value, ops, locations, binary)
			ops, locations = composePattern(e.Pattern, ops, locations, binary)
			ops, locations = match(0, e.Location, ops, locations)
			ops, locations = swapPop(e.Location, bytecode.SwapPopModePop, ops, locations)
			ops, locations = composeExpression(e.Body, ops, locations, binary)
			break
		}
	case *typed.If:
		{
			e := expr.(*typed.If)
			ops, locations = composeExpression(e.Condition, ops, locations, binary)
			ops, locations = composePattern(&typed.PDataOption{
				Location: e.Location,
				Name:     common.OakCoreBasicsTrue,
			}, ops, locations, binary)
			matchOpIndex := len(ops)
			ops, locations = match(0, e.Location, ops, locations) //jump to negative branch

			ops, locations = composeExpression(e.Positive, ops, locations, binary)
			jumpOpIndex := len(ops)
			ops, locations = jump(0, e.Location, ops, locations) //jump to the end

			negBranchIndex := len(ops)
			ops, locations = composeExpression(e.Negative, ops, locations, binary)

			ifEndIndex := len(ops)

			matchOp := ops[matchOpIndex].(bytecode.Match) //jump to negative branch
			matchOp.JumpDelta = int32(negBranchIndex - matchOpIndex - 1)
			ops[matchOpIndex] = matchOp

			jumpOp := ops[jumpOpIndex].(bytecode.Jump) //jump to the end
			jumpOp.Delta = int32(ifEndIndex - jumpOpIndex - 1)
			ops[jumpOpIndex] = jumpOp

			ops, locations = swapPop(e.Location, bytecode.SwapPopModeBoth, ops, locations)
			break
		}
	case *typed.Select:
		{
			e := expr.(*typed.Select)
			ops, locations = composeExpression(e.Condition, ops, locations, binary)
			var jumpToEndIndices []int
			var prevMatchOpIndex int
			for i, cs := range e.Cases {
				if i > 0 {
					matchOp := ops[prevMatchOpIndex].(bytecode.Match) //jump to next case
					matchOp.JumpDelta = int32(len(ops) - prevMatchOpIndex - 1)
					ops[prevMatchOpIndex] = matchOp
				}

				ops, locations = composePattern(cs.Pattern, ops, locations, binary)
				prevMatchOpIndex = len(ops)
				ops, locations = match(0, cs.Location, ops, locations)
				ops, locations = composeExpression(cs.Expression, ops, locations, binary)
				jumpToEndIndices = append(jumpToEndIndices, len(ops))
				ops, locations = jump(0, cs.Location, ops, locations)
			}

			selectEndIndex := len(ops)
			for _, jumpOpIndex := range jumpToEndIndices {
				jumpOp := ops[jumpOpIndex].(bytecode.Jump) //jump to the end
				jumpOp.Delta = int32(selectEndIndex - jumpOpIndex - 1)
				ops[jumpOpIndex] = jumpOp
			}

			ops, locations = swapPop(e.Location, bytecode.SwapPopModeBoth, ops, locations)
			break
		}
	default:
		{
			panic(common.SystemError{Message: "invalid case"})
		}
	}
	return ops, locations
}

func composePattern(
	pattern typed.Pattern, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	switch pattern.(type) {
	case *typed.PAlias:
		{
			e := pattern.(*typed.PAlias)
			ops, locations = composePattern(e.Nested, ops, locations, binary)
			ops, locations = makePattern(
				bytecode.PatternKindAlias, string(e.Alias), 0, e.Location, ops, locations, binary)
			break
		}
	case *typed.PAny:
		{
			e := pattern.(*typed.PAny)
			ops, locations = makePattern(bytecode.PatternKindAny, "", 0, e.Location, ops, locations, binary)
			break
		}
	case *typed.PCons:
		{
			e := pattern.(*typed.PCons)
			ops, locations = composePattern(e.Tail, ops, locations, binary)
			ops, locations = composePattern(e.Head, ops, locations, binary)
			ops, locations = makePattern(bytecode.PatternKindCons, "", 0, e.Location, ops, locations, binary)
			break
		}
	case *typed.PConst:
		{
			e := pattern.(*typed.PConst)
			ops, locations = loadConstValue(e.Value, bytecode.StackKindPattern, e.Location, ops, locations, binary)
			ops, locations = makePattern(bytecode.PatternKindConst, "", 0, e.Location, ops, locations, binary)
			break
		}
	case *typed.PDataOption:
		{
			e := pattern.(*typed.PDataOption)
			for _, p := range e.Args {
				ops, locations = composePattern(p, ops, locations, binary)
			}
			ops, locations = makePattern(bytecode.PatternKindDataOption,
				string(e.Name), len(e.Args), e.Location, ops, locations, binary)
			break
		}
	case *typed.PList:
		{
			e := pattern.(*typed.PList)
			for _, p := range e.Items {
				ops, locations = composePattern(p, ops, locations, binary)
			}
			ops, locations = makePattern(bytecode.PatternKindList, "", len(e.Items), e.Location, ops, locations, binary)
			break
		}
	case *typed.PNamed:
		{
			e := pattern.(*typed.PNamed)
			ops, locations = makePattern(
				bytecode.PatternKindNamed, string(e.Name), 0, e.Location, ops, locations, binary)
			break
		}
	case *typed.PRecord:
		{
			e := pattern.(*typed.PRecord)
			for _, f := range e.Fields {
				ops, locations = loadConstValue(
					ast.CString{Value: string(f.Name)}, bytecode.StackKindPattern, f.Location, ops, locations, binary)
			}
			ops, locations = makePattern(bytecode.PatternKindRecord, "", len(e.Fields), e.Location, ops, locations, binary)
			break
		}
	case *typed.PTuple:
		{
			e := pattern.(*typed.PTuple)
			for _, p := range e.Items {
				ops, locations = composePattern(p, ops, locations, binary)
			}
			ops, locations = makePattern(bytecode.PatternKindTuple, "", len(e.Items), e.Location, ops, locations, binary)
			break
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
	return ops, locations
}

func match(jumpDelta int, loc ast.Location, ops []bytecode.Op, locations []ast.Location) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.Match{JumpDelta: int32(jumpDelta)}),
		append(locations, loc)
}

func loadConstValue(
	c ast.ConstValue, stack bytecode.StackKind, loc ast.Location,
	ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	switch c.(type) {
	case ast.CUnit:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindUnit,
					Value: 0,
				}),
				append(locations, loc)
		}
	case ast.CChar:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindChar,
					Value: bytecode.ConstHash(c.(ast.CChar).Value),
				}),
				append(locations, loc)
		}
	case ast.CInt:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindInt,
					Value: binary.HashConst(bytecode.PackedInt{Value: c.(ast.CInt).Value}),
				}),
				append(locations, loc)
		}
	case ast.CFloat:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindFloat,
					Value: binary.HashConst(bytecode.PackedFloat{Value: c.(ast.CFloat).Value}),
				}),
				append(locations, loc)
		}
	case ast.CString:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindString,
					Value: bytecode.ConstHash(binary.HashString(c.(ast.CString).Value)),
				}),
				append(locations, loc)
		}
	default:
		panic(common.SystemError{Message: "invalid case"})
	}
}

func loadLocal(
	name string, loc ast.Location, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.LoadLocal{Name: binary.HashString(name)}),
		append(locations, loc)
}

func loadGlobal(
	ptr bytecode.Pointer, loc ast.Location, ops []bytecode.Op, locations []ast.Location,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.LoadGlobal{Pointer: ptr}),
		append(locations, loc)
}

func makePattern(
	kind bytecode.PatternKind, name string, numNested int,
	loc ast.Location, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	if numNested > 255 {
		panic(common.Error{Location: loc, Message: "pattern cannot contain more than 255 nested patterns"})
	}
	return append(ops, bytecode.MakePattern{Kind: kind, Name: binary.HashString(name), NumNested: uint8(numNested)}),
		append(locations, loc)
}

func call(name string, numArgs int, loc ast.Location, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	if numArgs > 255 {
		panic(common.Error{Location: loc, Message: "native function cannot be called with more than 255 arguments"})
	}
	return append(ops, bytecode.Call{Name: binary.HashString(name), NumArgs: uint8(numArgs)}),
		append(locations, loc)
}

func apply(numArgs int, loc ast.Location, ops []bytecode.Op, locations []ast.Location,
) ([]bytecode.Op, []ast.Location) {
	if numArgs > 255 {
		panic(common.Error{Location: loc, Message: "function cannot be applied with more than 255 arguments"})
	}
	return append(ops, bytecode.Apply{NumArgs: uint8(numArgs)}),
		append(locations, loc)
}

func access(filed string, loc ast.Location, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.Access{Field: binary.HashString(filed)}),
		append(locations, loc)
}

func makeObject(kind bytecode.ObjectKind, numArgs int, loc ast.Location,
	ops []bytecode.Op, locations []ast.Location,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.MakeObject{Kind: kind, NumArgs: uint32(numArgs)}),
		append(locations, loc)
}

func update(field string, loc ast.Location,
	ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.Update{Field: binary.HashString(field)}),
		append(locations, loc)
}

func jump(delta int, loc ast.Location,
	ops []bytecode.Op, locations []ast.Location,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.Jump{Delta: int32(delta)}),
		append(locations, loc)
}

func swapPop(loc ast.Location, mode bytecode.SwapPopMode,
	ops []bytecode.Op, locations []ast.Location,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.SwapPop{Mode: mode}),
		append(locations, loc)
}
