package processors

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/bytecode"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"slices"
)

func Compose(
	moduleName ast.QualifiedIdentifier,
	modules map[ast.QualifiedIdentifier]*typed.Module,
	debug bool,
	binary *bytecode.Binary,
) error {
	binary.HashString("")

	if slices.Contains(binary.CompiledPaths, moduleName) {
		return nil
	}

	binary.CompiledPaths = append(binary.CompiledPaths, moduleName)

	m := modules[moduleName]
	for depModule := range m.Dependencies {
		if err := Compose(depModule, modules, debug, binary); err != nil {
			return err
		}
	}

	for _, def := range m.Definitions {
		extId := common.MakeFullIdentifier(m.Name, def.Name)
		binary.FuncsMap[extId] = bytecode.Pointer(len(binary.Funcs))
		binary.Funcs = append(binary.Funcs, bytecode.Func{})
	}

	for _, def := range m.Definitions {
		pathId := common.MakeFullIdentifier(m.Name, def.Name)

		ptr := binary.FuncsMap[pathId]
		if binary.Funcs[ptr].Ops == nil {
			var err error
			binary.Funcs[ptr], err = composeDefinition(def, pathId, binary)
			if err != nil {
				return err
			}
			if !def.Hidden || debug {
				binary.Exports[pathId] = ptr
			}
		}
	}
	return nil
}

func composeDefinition(
	def *typed.Definition, pathId ast.FullIdentifier, binary *bytecode.Binary,
) (bytecode.Func, error) {
	var ops []bytecode.Op
	var locations []ast.Location
	var err error

	if nc, ok := def.Expression.(*typed.NativeCall); ok && pathId == nc.Name {
		ops, locations, err = call(string(nc.Name), len(nc.Args), nc.Location, ops, locations, binary)
		if err != nil {
			return bytecode.Func{}, err
		}
	} else {
		for i := len(def.Params) - 1; i >= 0; i-- {
			p := def.Params[i]
			ops, locations, err = composePattern(p, ops, locations, binary)
			if err != nil {
				return bytecode.Func{}, err
			}
			ops, locations = match(0, p.GetLocation(), ops, locations)
			ops, locations = swapPop(p.GetLocation(), bytecode.SwapPopModePop, ops, locations)
		}
		ops, locations, err = composeExpression(def.Expression, ops, locations, binary)
		if err != nil {
			return bytecode.Func{}, err
		}
	}

	return bytecode.Func{
		NumArgs:   uint32(len(def.Params)),
		Ops:       ops,
		FilePath:  def.Location.FilePath(),
		Locations: locations,
	}, nil
}

func composeExpression(
	expr typed.Expression, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location, error) {
	var err error
	switch expr.(type) {
	case *typed.NativeCall:
		{
			e := expr.(*typed.NativeCall)
			for _, arg := range e.Args {
				ops, locations, err = composeExpression(arg, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations, err = call(string(e.Name), len(e.Args), e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.Apply:
		{
			e := expr.(*typed.Apply)
			for _, arg := range e.Args {
				ops, locations, err = composeExpression(arg, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations, err = composeExpression(e.Func, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations, err = apply(len(e.Args), e.Location, ops, locations)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.Const:
		{
			e := expr.(*typed.Const)
			v := e.Value
			if iv, ok := v.(ast.CInt); ok {
				if ex, ok := e.Type.(*typed.TNative); ok && ex.Name == common.NarCoreMathFloat {
					v = ast.CFloat{Value: float64(iv.Value)}
				}
			}
			ops, locations, err = loadConstValue(v, bytecode.StackKindObject, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
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
			id := common.MakeFullIdentifier(e.ModuleName, e.DefinitionName)
			funcIndex, ok := binary.FuncsMap[id]
			if !ok {
				return nil, nil, common.Error{
					Location: e.Location,
					Message:  fmt.Sprintf("global definition `%v` not found", id),
				}
			}
			ops, locations = loadGlobal(funcIndex, e.Location, ops, locations)
			break
		}
	case *typed.Access:
		{
			e := expr.(*typed.Access)
			ops, locations, err = composeExpression(e.Record, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations = access(string(e.FieldName), e.Location, ops, locations, binary)
			break
		}

	case *typed.List:
		{
			e := expr.(*typed.List)
			for _, item := range e.Items {
				ops, locations, err = composeExpression(item, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations = makeObject(bytecode.ObjectKindList, len(e.Items), e.Location, ops, locations)
			break
		}
	case *typed.Tuple:
		{
			e := expr.(*typed.Tuple)
			for _, item := range e.Items {
				ops, locations, err = composeExpression(item, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations = makeObject(bytecode.ObjectKindTuple, len(e.Items), e.Location, ops, locations)
			break
		}
	case *typed.Record:
		{
			e := expr.(*typed.Record)
			for _, f := range e.Fields {
				ops, locations, err = composeExpression(f.Value, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
				ops, locations, err = loadConstValue(
					ast.CString{Value: string(f.Name)}, bytecode.StackKindObject, f.Location,
					ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations = makeObject(bytecode.ObjectKindRecord, len(e.Fields), e.Location, ops, locations)
			break
		}
	case *typed.Constructor:
		{
			e := expr.(*typed.Constructor)
			for _, arg := range e.Args {
				ops, locations, err = composeExpression(arg, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations, err = loadConstValue(
				ast.CString{Value: string(common.MakeDataOptionIdentifier(e.DataName, e.OptionName))},
				bytecode.StackKindObject, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations = makeObject(bytecode.ObjectKindData, len(e.Args), e.Location, ops, locations)
			break
		}
	case *typed.UpdateLocal:
		{
			e := expr.(*typed.UpdateLocal)
			ops, locations = loadLocal(string(e.RecordName), e.Location, ops, locations, binary)

			for _, f := range e.Fields {
				ops, locations, err = composeExpression(f.Value, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
				ops, locations = update(string(f.Name), f.Location, ops, locations, binary)
			}
			break
		}
	case *typed.UpdateGlobal:
		{
			e := expr.(*typed.UpdateGlobal)
			id := common.MakeFullIdentifier(e.ModuleName, e.DefinitionName)
			ops, locations = loadGlobal(binary.FuncsMap[id], e.Location, ops, locations)

			for _, f := range e.Fields {
				ops, locations, err = composeExpression(f.Value, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
				ops, locations = update(string(f.Name), f.Location, ops, locations, binary)
			}
			break
		}
	case *typed.Let:
		{
			e := expr.(*typed.Let)
			ops, locations, err = composeExpression(e.Value, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations, err = composePattern(e.Pattern, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations = match(0, e.Location, ops, locations)
			ops, locations = swapPop(e.Location, bytecode.SwapPopModePop, ops, locations)
			ops, locations, err = composeExpression(e.Body, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.Select:
		{
			e := expr.(*typed.Select)
			ops, locations, err = composeExpression(e.Condition, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			var jumpToEndIndices []int
			var prevMatchOpIndex int
			for i, cs := range e.Cases {
				if i > 0 {
					matchOp := ops[prevMatchOpIndex].(bytecode.Match) //jump to next case
					matchOp.JumpDelta = int32(len(ops) - prevMatchOpIndex - 1)
					ops[prevMatchOpIndex] = matchOp
				}

				ops, locations, err = composePattern(cs.Pattern, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
				prevMatchOpIndex = len(ops)
				ops, locations = match(0, cs.Location, ops, locations)
				ops, locations, err = composeExpression(cs.Expression, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
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
			return nil, nil, common.NewCompilerError("impossible case")
		}
	}
	return ops, locations, nil
}

func composePattern(
	pattern typed.Pattern, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location, error) {
	var err error
	switch pattern.(type) {
	case *typed.PAlias:
		{
			e := pattern.(*typed.PAlias)
			ops, locations, err = composePattern(e.Nested, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations, err = makePattern(
				bytecode.PatternKindAlias, string(e.Alias), 0, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PAny:
		{
			e := pattern.(*typed.PAny)
			ops, locations, err = makePattern(bytecode.PatternKindAny, "", 0, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PCons:
		{
			e := pattern.(*typed.PCons)
			ops, locations, err = composePattern(e.Tail, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations, err = composePattern(e.Head, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations, err = makePattern(bytecode.PatternKindCons, "", 0, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PConst:
		{
			e := pattern.(*typed.PConst)
			ops, locations, err = loadConstValue(e.Value, bytecode.StackKindPattern, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			ops, locations, err = makePattern(bytecode.PatternKindConst, "", 0, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PDataOption:
		{
			e := pattern.(*typed.PDataOption)
			for _, p := range e.Args {
				ops, locations, err = composePattern(p, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations, err = makePattern(
				bytecode.PatternKindDataOption,
				string(common.MakeDataOptionIdentifier(e.DataName, e.OptionName)),
				len(e.Args), e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PList:
		{
			e := pattern.(*typed.PList)
			for _, p := range e.Items {
				ops, locations, err = composePattern(p, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations, err = makePattern(
				bytecode.PatternKindList, "", len(e.Items), e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PNamed:
		{
			e := pattern.(*typed.PNamed)
			ops, locations, err = makePattern(
				bytecode.PatternKindNamed, string(e.Name), 0, e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PRecord:
		{
			e := pattern.(*typed.PRecord)
			for _, f := range e.Fields {
				ops, locations, err = loadConstValue(
					ast.CString{Value: string(f.Name)}, bytecode.StackKindPattern, f.Location, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations, err = makePattern(
				bytecode.PatternKindRecord, "", len(e.Fields), e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	case *typed.PTuple:
		{
			e := pattern.(*typed.PTuple)
			for _, p := range e.Items {
				ops, locations, err = composePattern(p, ops, locations, binary)
				if err != nil {
					return nil, nil, err
				}
			}
			ops, locations, err = makePattern(
				bytecode.PatternKindTuple, "", len(e.Items), e.Location, ops, locations, binary)
			if err != nil {
				return nil, nil, err
			}
			break
		}
	default:
		return nil, nil, common.NewCompilerError("impossible case")
	}
	return ops, locations, nil
}

func match(jumpDelta int, loc ast.Location, ops []bytecode.Op, locations []ast.Location) ([]bytecode.Op, []ast.Location) {
	return append(ops, bytecode.Match{JumpDelta: int32(jumpDelta)}),
		append(locations, loc)
}

func loadConstValue(
	c ast.ConstValue, stack bytecode.StackKind, loc ast.Location,
	ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location, error) {
	switch c.(type) {
	case ast.CUnit:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindUnit,
					Value: 0,
				}),
				append(locations, loc),
				nil
		}
	case ast.CChar:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindChar,
					Value: bytecode.ConstHash(c.(ast.CChar).Value),
				}),
				append(locations, loc),
				nil
		}
	case ast.CInt:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindInt,
					Value: binary.HashConst(bytecode.PackedInt{Value: c.(ast.CInt).Value}),
				}),
				append(locations, loc),
				nil
		}
	case ast.CFloat:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindFloat,
					Value: binary.HashConst(bytecode.PackedFloat{Value: c.(ast.CFloat).Value}),
				}),
				append(locations, loc),
				nil
		}
	case ast.CString:
		{
			return append(ops, bytecode.LoadConst{
					Stack: stack,
					Kind:  bytecode.ConstKindString,
					Value: bytecode.ConstHash(binary.HashString(c.(ast.CString).Value)),
				}),
				append(locations, loc),
				nil
		}
	default:
		return nil, nil, common.NewCompilerError("impossible case")
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
) ([]bytecode.Op, []ast.Location, error) {
	if numNested > 255 {
		return nil, nil, common.Error{
			Location: loc,
			Message:  "pattern cannot contain more than 255 nested patterns",
		}
	}
	return append(ops, bytecode.MakePattern{Kind: kind, Name: binary.HashString(name), NumNested: uint8(numNested)}),
		append(locations, loc), nil
}

func call(name string, numArgs int, loc ast.Location, ops []bytecode.Op, locations []ast.Location, binary *bytecode.Binary,
) ([]bytecode.Op, []ast.Location, error) {
	if numArgs > 255 {
		return nil, nil, common.Error{
			Location: loc,
			Message:  "function cannot be called with more than 255 arguments",
		}
	}
	return append(ops, bytecode.Call{Name: binary.HashString(name), NumArgs: uint8(numArgs)}),
		append(locations, loc), nil
}

func apply(numArgs int, loc ast.Location, ops []bytecode.Op, locations []ast.Location,
) ([]bytecode.Op, []ast.Location, error) {
	if numArgs > 255 {
		return nil, nil, common.Error{
			Location: loc,
			Message:  "function cannot be applied with more than 255 arguments",
		}
	}
	return append(ops, bytecode.Apply{NumArgs: uint8(numArgs)}),
		append(locations, loc), nil
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
