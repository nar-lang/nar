package processors

import (
	"fmt"
	"oak-compiler/ast"
	"oak-compiler/ast/bytecode"
	"oak-compiler/ast/typed"
	"oak-compiler/common"
	"slices"
)

var lambdaIndex = uint64(0)

func Compile(path string, modules map[string]*typed.Module, binary *bytecode.Binary) {
	binary.HashString("")

	if slices.Contains(binary.CompiledPaths, path) {
		return
	}

	binary.CompiledPaths = append(binary.CompiledPaths, path)

	m := modules[path]
	for _, depPath := range m.DepPaths {
		Compile(depPath, modules, binary)
	}

	var names []ast.Identifier
	for n := range m.Definitions {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, name := range names {
		if m.Definitions[name].Expression == nil {
			continue
		}
		pathId := common.MakePathIdentifier(m.Path, name)
		binary.FuncsMap[pathId] = bytecode.Pointer(len(binary.Funcs))
		binary.Funcs = append(binary.Funcs, bytecode.Func{})
	}

	for _, name := range names {
		def := m.Definitions[name]
		if def.Expression == nil {
			continue
		}
		pathId := common.MakePathIdentifier(m.Path, name)
		ptr := binary.FuncsMap[pathId]
		if binary.Funcs[ptr].Ops == nil {
			binary.Funcs[ptr] = compileDefinition(def.Expression, binary)
			if !def.Hidden {
				binary.Exports[common.MakeExternalIdentifier(m.Name, name)] = ptr
			}
		}
	}
}

func compileDefinition(def typed.Expression, binary *bytecode.Binary) bytecode.Func {
	if fn, ok := def.(*typed.Lambda); ok {
		var ops []bytecode.Op
		var locations []ast.Location
		for i := len(fn.Params) - 1; i >= 0; i-- {
			p := fn.Params[i]
			ops, locations = compilePattern(p, ops, locations, binary)
			ops, locations = match(0, p.GetLocation(), ops, locations)
		}
		ops, locations = compileExpression(fn.Body, ops, locations, binary)
		return bytecode.Func{
			NumArgs:   uint32(len(fn.Params)),
			Ops:       ops,
			FilePath:  def.GetLocation().FilePath,
			Locations: locations,
		}
	} else {
		ops, locations := compileExpression(def, nil, nil, binary)
		return bytecode.Func{
			NumArgs:   0,
			Ops:       ops,
			FilePath:  def.GetLocation().FilePath,
			Locations: locations,
		}
	}
}

func compileExpression(
	expr typed.Expression, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	switch expr.(type) {
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			for _, arg := range e.Args {
				ops, locations = compileExpression(arg, ops, locations, binary)
			}
			ops, locations = call(string(e.Name), len(e.Args), e.Location, ops, locations, binary)
			break
		}
	case *typed.Apply:
		{
			e := expr.(*typed.Apply)
			for _, arg := range e.Args {
				ops, locations = compileExpression(arg, ops, locations, binary)
			}
			ops, locations = compileExpression(e.Func, ops, locations, binary)
			ops, locations = apply(len(e.Args), e.Location, ops, locations)
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			ops, locations = loadConstValue(e.Value, bytecode.StackKindObject, e.Location, ops, locations, binary)
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
			id := common.MakePathIdentifier(e.ModulePath, e.DefinitionName)
			ops, locations = loadGlobal(binary.FuncsMap[id], e.Location, ops, locations)
			break
		}
	case *typed.Access:
		{
			e := expr.(*typed.Access)
			ops, locations = compileExpression(e.Record, ops, locations, binary)
			ops, locations = access(string(e.FieldName), e.Location, ops, locations, binary)
			break
		}

	case *typed.List:
		{
			e := expr.(*typed.List)
			for _, item := range e.Items {
				ops, locations = compileExpression(item, ops, locations, binary)
			}
			ops, locations = makeObject(bytecode.ObjectKindList, len(e.Items), e.Location, ops, locations)
			break
		}
	case *typed.Tuple:
		{
			e := expr.(*typed.Tuple)
			for _, item := range e.Items {
				ops, locations = compileExpression(item, ops, locations, binary)
			}
			ops, locations = makeObject(bytecode.ObjectKindTuple, len(e.Items), e.Location, ops, locations)
			break
		}
	case *typed.Record:
		{
			e := expr.(*typed.Record)
			for _, f := range e.Fields {
				ops, locations = compileExpression(f.Value, ops, locations, binary)
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
				ops, locations = compileExpression(arg, ops, locations, binary)
			}
			ops, locations = loadConstValue(ast.CString{Value: string(e.OptionName)}, bytecode.StackKindObject, e.Location,
				ops, locations, binary)
			ops, locations = makeObject(bytecode.ObjectKindData, len(e.Args), e.Location, ops, locations)
			break
		}
	case *typed.Lambda:
		{
			e := expr.(*typed.Lambda)
			ptr := bytecode.Pointer(len(binary.Funcs))
			binary.Funcs = append(binary.Funcs, bytecode.Func{})
			lambdaIndex++
			binary.FuncsMap[ast.PathIdentifier(fmt.Sprintf("$%v", lambdaIndex))] = ptr
			binary.Funcs[ptr] = compileDefinition(e, binary)
			ops, locations = loadGlobal(ptr, e.Location, ops, locations)
			break
		}
	case *typed.UpdateLocal:
		{
			e := expr.(*typed.UpdateLocal)
			ops, locations = loadLocal(string(e.RecordName), e.Location, ops, locations, binary)

			for _, f := range e.Fields {
				ops, locations = compileExpression(f.Value, ops, locations, binary)
				ops, locations = update(string(f.Name), f.Location, ops, locations, binary)
			}
			break
		}
	case *typed.UpdateGlobal:
		{
			e := expr.(*typed.UpdateGlobal)
			id := common.MakePathIdentifier(e.ModulePath, e.DefinitionName)
			ops, locations = loadGlobal(binary.FuncsMap[id], e.Location, ops, locations)

			for _, f := range e.Fields {
				ops, locations = compileExpression(f.Value, ops, locations, binary)
				ops, locations = update(string(f.Name), f.Location, ops, locations, binary)
			}
			break
		}
	case *typed.Let:
		{
			e := expr.(*typed.Let)
			ops, locations = compileExpression(e.Definition.Expression, ops, locations, binary)
			ops, locations = compilePattern(e.Definition.Pattern, ops, locations, binary)
			ops, locations = match(0, e.Location, ops, locations)
			break
		}
	case *typed.If:
		{
			e := expr.(*typed.If)
			ops, locations = compileExpression(e.Condition, ops, locations, binary)
			ops, locations = compilePattern(&typed.PDataOption{
				Location:   ast.Location{},
				Type:       nil,
				Name:       "",
				Definition: nil,
				Args:       nil,
			}, ops, locations, binary)
			matchOpIndex := len(ops)
			ops, locations = match(0, e.Location, ops, locations)

			ops, locations = compileExpression(e.Positive, ops, locations, binary)
			jumpOpIndex := len(ops)
			ops, locations = jump(0, e.Location, ops, locations)

			negBranchIndex := len(ops)
			ops, locations = unloadLocal("", e.Location, ops, locations, binary) //unload condition
			ops, locations = compileExpression(e.Negative, ops, locations, binary)

			ifEndIndex := len(ops)

			matchOp := ops[matchOpIndex].(bytecode.Match) //jump to negative branch
			matchOp.JumpDelta = int32(negBranchIndex - matchOpIndex - 1)
			ops[matchOpIndex] = matchOp

			jumpOp := ops[jumpOpIndex].(bytecode.Jump) //jump to the end
			jumpOp.Delta = int32(ifEndIndex - jumpOpIndex - 1)
			ops[jumpOpIndex] = jumpOp
			break
		}
	case *typed.Select:
		{
			e := expr.(*typed.Select)
			ops, locations = compileExpression(e.Condition, ops, locations, binary)
			var jumpToEndIndices []int
			var prevMatchOpIndex int
			for i, cs := range e.Cases {
				if i > 0 {
					matchOp := ops[prevMatchOpIndex].(bytecode.Match) //jump to next case
					matchOp.JumpDelta = int32(len(ops) - prevMatchOpIndex - 1)
					ops[prevMatchOpIndex] = matchOp
				}

				ops, locations = duplicate(cs.Pattern.GetLocation(), ops, locations) //copy condition
				ops, locations = compilePattern(cs.Pattern, ops, locations, binary)
				prevMatchOpIndex = len(ops)
				ops, locations = match(0, cs.Location, ops, locations)
				ops, locations = compileExpression(cs.Expression, ops, locations, binary)
				jumpToEndIndices = append(jumpToEndIndices, len(ops))
				ops, locations = jump(0, cs.Location, ops, locations)
			}

			/* TODO: invalid situation, generate crash?
			// this can happen only if compiler failed to check that cases are not exhausting select condition

			//last case jump out
			matchOp := ops[prevMatchOpIndex].(bytecode.Match) //jump to next case
			matchOp.JumpDelta = int32(len(ops) - prevMatchOpIndex - 1)
			ops[prevMatchOpIndex] = matchOp
			*/

			ops, locations = unloadLocal("", e.GetLocation(), ops, locations, binary) //unload condition

			selectEndIndex := len(ops)
			for _, jumpOpIndex := range jumpToEndIndices {
				jumpOp := ops[jumpOpIndex].(bytecode.Jump) //jump to the end
				jumpOp.Delta = int32(selectEndIndex - jumpOpIndex - 1)
				ops[jumpOpIndex] = jumpOp
			}

			break
		}
	default:
		{
			panic(common.SystemError{Message: "invalid case"})
		}
	}
	return ops, locations
}

func compilePattern(
	pattern typed.Pattern, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	switch pattern.(type) {
	case *typed.PAlias:
		{
			e := pattern.(*typed.PAlias)
			ops, locations = compilePattern(e.Nested, ops, locations, binary)
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
			ops, locations = compilePattern(e.Tail, ops, locations, binary)
			ops, locations = compilePattern(e.Head, ops, locations, binary)
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
				ops, locations = compilePattern(p, ops, locations, binary)
			}
			ops, locations = makePattern(bytecode.PatternKindDataOption,
				string(e.Name), len(e.Args), e.Location, ops, locations, binary)
			break
		}
	case *typed.PList:
		{
			e := pattern.(*typed.PList)
			for _, p := range e.Items {
				ops, locations = compilePattern(p, ops, locations, binary)
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
				ops, locations = compilePattern(p, ops, locations, binary)
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

func unloadLocal(
	name string, loc ast.Location, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.UnloadLocal{Name: binary.HashString(name)}),
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

func duplicate(loc ast.Location,
	ops []bytecode.Op, locations []ast.Location,
) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.Duplicate{}),
		append(locations, loc)
}
