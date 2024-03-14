package runtime

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/pkg/bytecode"
)

//TODO: make const objects when loading binary, and dont copy for every instance
//TODO: make string objects for every string in binary, and dont copy for every instance

func NewRuntime(program *bytecode.Binary) *Runtime {
	rt := &Runtime{
		program:           program,
		defs:              map[bytecode.FullIdentifier]Object{},
		cachedExpressions: map[bytecode.Pointer]Object{},
		arenas:            make([]typedArena, instanceKindCount),
	}

	initialCapacity := 64
	for i := range rt.arenas {
		rt.arenas[i] = newTypedArenaWithKind(InstanceKind(i), initialCapacity)
	}
	rt.Clean(true)
	return rt
}

type Runtime struct {
	program           *bytecode.Binary
	defs              map[bytecode.FullIdentifier]Object
	arenas            []typedArena
	callStack         []bytecode.StringHash
	locals            []local
	cachedExpressions map[bytecode.Pointer]Object
}

func (rt *Runtime) Dispose() {
	for _, arena := range rt.arenas {
		arena.clean(true)
	}
}

func (rt *Runtime) RegisterDef(moduleName bytecode.QualifiedIdentifier, name ast.Identifier, def Object) {
	rt.defs[bytecode.FullIdentifier(moduleName)+"."+bytecode.FullIdentifier(name)] = def
}

func (rt *Runtime) Apply(defName bytecode.FullIdentifier, args ...Object) (Object, error) {
	if len(rt.callStack) > 0 {
		return invalidObject, fmt.Errorf("function is being called already")
	}
	fnIndex, ok := rt.program.Exports[defName]
	if !ok {
		return invalidObject, fmt.Errorf("definition `%s` is not exported by loaded binary", defName)
	}
	if kDebug && fnIndex >= bytecode.Pointer(len(rt.program.Funcs)) {
		return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid function pointer)")
	}
	fn := rt.program.Funcs[fnIndex]
	afn := rt.newClosure(fn)
	return rt.ApplyFunc(afn, args...)
}

func (rt *Runtime) ApplyFunc(fn Object, args ...Object) (Object, error) {
	afn, err := toClosure(rt, fn)
	if err != nil {
		return invalidObject, err
	}
	curried, err := ToList(rt, afn.curried)
	if err != nil {
		return invalidObject, err
	}
	args = append(curried, args...)
	if int(afn.fn.NumArgs) == len(args) {
		return rt.execute(afn.fn, args)
	} else if int(afn.fn.NumArgs) < len(args) {
		restLen := len(args) - int(afn.fn.NumArgs)
		rest := make([]Object, restLen)
		copy(rest, args[len(args)-restLen:])
		args = args[:len(args)-restLen]
		result, err := rt.execute(afn.fn, args)
		if err != nil {
			return invalidObject, err
		}
		return rt.ApplyFunc(result, rest...)
	} else {
		return rt.newClosure(afn.fn, args...), nil
	}
}

func (rt *Runtime) execute(fn bytecode.Func, objectStack []Object) (Object, error) {
	var patternStack []Object
	rt.callStack = append(rt.callStack, fn.Name)
	numLocals := 0
	numOps := len(fn.Ops)
	for index := 0; index < numOps; index++ {
		opKind, b, c, a := fn.Ops[index].Decompose()
		switch opKind {
		case bytecode.OpKindLoadLocal:
			if kDebug && a >= uint32(len(rt.program.Strings)) {
				return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid local name)")
			}
			name := TString(rt.program.Strings[a])
			found := false
			for i := len(rt.locals) - 1; i >= len(rt.locals)-numLocals; i-- {
				if rt.locals[i].name == name {
					objectStack = append(objectStack, rt.locals[i].value)
					found = true
					break
				}
			}
			if kDebug && !found {
				return invalidObject, fmt.Errorf("loaded binary is corrupted (undefined local `%s`)", name)
			}
		case bytecode.OpKindLoadGlobal:
			if kDebug && a >= uint32(len(rt.program.Funcs)) {
				return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid global pointer)")
			}
			glob := rt.program.Funcs[a]
			if glob.NumArgs == 0 {
				if cached, ok := rt.cachedExpressions[bytecode.Pointer(a)]; !ok {
					result, err := rt.execute(glob, nil)
					if err != nil {
						return invalidObject, err
					}
					rt.cachedExpressions[bytecode.Pointer(a)] = result
					objectStack = append(objectStack, result)
				} else {
					objectStack = append(objectStack, cached)
				}
			} else {
				objectStack = append(objectStack, rt.newClosure(glob))
			}
		case bytecode.OpKindLoadConst:
			var value Object
			switch bytecode.ConstKind(c) {
			case bytecode.ConstKindUnit:
				value = rt.NewUnit()
			case bytecode.ConstKindChar:
				value = rt.NewChar(TChar(a))
			case bytecode.ConstKindInt:
				if kDebug && a >= uint32(len(rt.program.Consts)) {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid const index)")
				}
				value = rt.NewInt(TInt(rt.program.Consts[a].Int()))
			case bytecode.ConstKindFloat:
				if kDebug && a >= uint32(len(rt.program.Consts)) {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid const index)")
				}
				value = rt.NewFloat(TFloat(rt.program.Consts[a].Float()))
			case bytecode.ConstKindString:
				if kDebug && a >= uint32(len(rt.program.Strings)) {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				value = rt.NewString(TString(rt.program.Strings[a]))
			default:
				if kDebug {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid const kind)")
				}
			}
			switch bytecode.StackKind(b) {
			case bytecode.StackKindObject:
				objectStack = append(objectStack, value)
			case bytecode.StackKindPattern:
				patternStack = append(patternStack, value)
			default:
				if kDebug {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid stack kind)")
				}
			}
		case bytecode.OpKindApply:
			x, err := pop(&objectStack)
			if err != nil {
				return invalidObject, err
			}
			afn, err := toClosure(rt, x)
			if err != nil {
				return invalidObject, err
			}
			numArgs := int(b)
			args, err := popX(&objectStack, numArgs)
			if err != nil {
				return invalidObject, err
			}
			curried, err := ToList(rt, afn.curried)
			if err != nil {
				return invalidObject, err
			}
			args = append(curried, args...)
			if int(afn.fn.NumArgs) == len(args) {
				result, err := rt.execute(afn.fn, args)
				if err != nil {
					return invalidObject, err
				}
				objectStack = append(objectStack, result)
			} else {
				objectStack = append(objectStack, rt.newClosure(afn.fn, args...))
			}
		case bytecode.OpKindCall:
			if kDebug && a >= uint32(len(rt.program.Strings)) {
				return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid string index)")
			}
			name := rt.program.Strings[a]
			def, ok := rt.defs[bytecode.FullIdentifier(name)]
			if !ok {
				return invalidObject, fmt.Errorf("definition `%s` is not registered", name)
			}
			if def.Kind() == InstanceKindFunction {
				ptr, arity, err := ToFunc(rt, def)
				if err != nil {
					return invalidObject, err
				}
				args, err := popX(&objectStack, int(arity))
				if err != nil {
					return invalidObject, err
				}
				result, err := callFunc(ptr, args)
				if err != nil {
					return invalidObject, err
				}
				objectStack = append(objectStack, result)
			} else {
				return invalidObject, fmt.Errorf("definition `%s` is not a function", name)
				//objectStack = append(objectStack, def)
			}
		case bytecode.OpKindJump:
			if b == 0 {
				index += int(a)
			} else {
				pt, err := pop(&patternStack)
				if err != nil {
					return invalidObject, err
				}
				if len(objectStack) == 0 {
					return invalidObject, fmt.Errorf("stack is empty")
				}
				match, err := rt.match(pt, objectStack[len(objectStack)-1], &numLocals)
				if err != nil {
					return invalidObject, err
				}
				if !match {
					if kDebug && a == 0 {
						return invalidObject, fmt.Errorf("pattern match with jump delta 0 should not fail")
					}
					index += int(a)
				}
			}
		case bytecode.OpKindMakeObject:
			switch bytecode.ObjectKind(b) {
			case bytecode.ObjectKindList:
				items, err := popX(&objectStack, int(a))
				if err != nil {
					return invalidObject, err
				}
				objectStack = append(objectStack, rt.NewList(items...))
			case bytecode.ObjectKindTuple:
				items, err := popX(&objectStack, int(a))
				if err != nil {
					return invalidObject, err
				}
				objectStack = append(objectStack, rt.NewTuple(items...))
			case bytecode.ObjectKindRecord:
				items, err := popX(&objectStack, int(a*2))
				if err != nil {
					return invalidObject, err
				}
				objectStack = append(objectStack, rt.newObjectRecordRaw(items...))
			case bytecode.ObjectKindOption:
				nameObj, err := pop(&objectStack)
				if err != nil {
					return invalidObject, err
				}
				name, err := ToString(rt, nameObj)
				if err != nil {
					return invalidObject, err
				}
				values, err := popX(&objectStack, int(a))
				if err != nil {
					return invalidObject, err
				}
				objectStack = append(objectStack, rt.NewOption(name, values...))
			default:
				if kDebug {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid object kind)")
				}
			}
		case bytecode.OpKindMakePattern:
			name := rt.NewString("")
			var items []Object
			var err error
			kind := bytecode.PatternKind(b)
			switch kind {
			case bytecode.PatternKindAlias:
				if kDebug && a >= uint32(len(rt.program.Strings)) {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				name = rt.NewString(TString(rt.program.Strings[a]))
				items, err = popX(&patternStack, 1)
			case bytecode.PatternKindAny:
				break
			case bytecode.PatternKindCons:
				items, err = popX(&patternStack, 2)
			case bytecode.PatternKindConst:
				items, err = popX(&patternStack, 1)
			case bytecode.PatternKindDataOption:
				if kDebug && a >= uint32(len(rt.program.Strings)) {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				name = rt.NewString(TString(rt.program.Strings[a]))
				items, err = popX(&patternStack, int(c))
			case bytecode.PatternKindList:
				items, err = popX(&patternStack, int(c)) //TODO: use a register for list length
			case bytecode.PatternKindNamed:
				if kDebug && a >= uint32(len(rt.program.Strings)) {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				name = rt.NewString(TString(rt.program.Strings[a]))
			case bytecode.PatternKindRecord:
				items, err = popX(&patternStack, int(c*2)) //TODO: use a register for list length
			case bytecode.PatternKindTuple:
				items, err = popX(&patternStack, int(c))
			default:
				if kDebug {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid pattern kind)")
				}
			}
			if err != nil {
				return invalidObject, err
			}
			p, err := rt.newPattern(name, items, kind)
			if err != nil {
				return invalidObject, err
			}
			patternStack = append(patternStack, p)
		case bytecode.OpKindAccess:
			if kDebug && a >= uint32(len(rt.program.Strings)) {
				return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid string index)")
			}
			record, err := pop(&objectStack)
			if err != nil {
				return invalidObject, err
			}
			name := rt.program.Strings[a]
			field, ok, err := FindRecordField(rt, record, TString(name))
			if err != nil {
				return invalidObject, err
			}
			if kDebug && !ok {
				return invalidObject, fmt.Errorf("record does not have field `%s`", name)
			}
			objectStack = append(objectStack, field)
		case bytecode.OpKindUpdate:
			if kDebug && a >= uint32(len(rt.program.Strings)) {
				return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid string index)")
			}
			key := rt.program.Strings[a]
			value, err := pop(&objectStack)
			if err != nil {
				return invalidObject, err
			}
			record, err := pop(&objectStack)
			if err != nil {
				return invalidObject, err
			}
			updated, err := UpdateRecordField(rt, record, rt.NewString(TString(key)), value)
			if err != nil {
				return invalidObject, err
			}
			objectStack = append(objectStack, updated)
		case bytecode.OpKindSwapPop:
			switch bytecode.SwapPopMode(b) {
			case bytecode.SwapPopModeBoth:
				tail, err := popX(&objectStack, 2)
				if err != nil {
					return invalidObject, err
				}
				objectStack = append(objectStack, tail[1])
			case bytecode.SwapPopModePop:
				_, err := pop(&objectStack)
				if err != nil {
					return invalidObject, err
				}
			default:
				if kDebug {
					return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid swap pop mode)")
				}
			}
		default:
			if kDebug {
				return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid op kind)")
			}
		}
	}
	_, err := popX(&rt.locals, numLocals)
	if err != nil {
		return invalidObject, err
	}
	_, err = pop(&rt.callStack)
	if err != nil {
		return invalidObject, err
	}
	result, err := pop(&objectStack)
	if err != nil {
		return invalidObject, err
	}
	if kDebug && len(objectStack) != 0 {
		return invalidObject, fmt.Errorf("stack is not empty after execution")
	}
	return result, nil
}

func (rt *Runtime) match(pattern Object, obj Object, numLocals *int) (bool, error) {
	p, err := toPattern(rt, pattern)
	if err != nil {
		return false, err
	}
	switch p.kind {
	case bytecode.PatternKindAlias:
		name, err := ToString(rt, p.name)
		if err != nil {
			return false, err
		}
		rt.locals = append(rt.locals, local{name: name, value: obj})
		*numLocals = *numLocals + 1
		nested, err := ToList(rt, p.items)
		if err != nil {
			return false, err
		}
		if kDebug && len(nested) != 1 {
			return false, fmt.Errorf("alias pattern should have exactly one nested pattern")
		}
		return rt.match(nested[0], obj, numLocals)
	case bytecode.PatternKindAny:
		return true, nil
	case bytecode.PatternKindCons:
		nested, err := ToList(rt, p.items)
		if err != nil {
			return false, err
		}
		if kDebug && len(nested) != 2 {
			return false, fmt.Errorf("cons pattern should have exactly two nested patterns")
		}
		if obj.invalid() {
			return false, nil
		}
		list, err := toListItem(rt, obj)
		if err != nil {
			return false, err
		}
		match, err := rt.match(nested[1], list.value, numLocals)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
		return rt.match(nested[0], list.next, numLocals) //TODO: optimize: do not create new list, use low level API
	case bytecode.PatternKindConst:
		nested, err := ToList(rt, p.items)
		if err != nil {
			return false, err
		}
		if kDebug && len(nested) != 1 {
			return false, fmt.Errorf("const pattern should have exactly one nested pattern")
		}
		return constEqualsTo(rt, nested[0], obj)
	case bytecode.PatternKindDataOption:
		objName, objValues, err := ToOption(rt, obj)
		if err != nil {
			return false, err
		}
		eq, err := constEqualsTo(rt, p.name, objName)
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
		ptList, err := ToList(rt, p.items)
		if err != nil {
			return false, err
		}
		if len(ptList) != len(objValues) {
			return false, fmt.Errorf("invalid option pattern match, number of values differs")
		}
		for i := 0; i < len(ptList); i++ {
			match, err := rt.match(ptList[i], objValues[i], numLocals)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
		return true, nil
	case bytecode.PatternKindList:
		objList, err := ToList(rt, obj)
		if err != nil {
			return false, err
		}
		patList, err := ToList(rt, p.items)
		if err != nil {
			return false, err
		}
		if len(objList) != len(patList) {
			return false, nil
		}
		for i := 0; i < len(objList); i++ {
			match, err := rt.match(patList[i], objList[i], numLocals)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
		return true, nil
	case bytecode.PatternKindNamed:
		name, err := ToString(rt, p.name)
		if err != nil {
			return false, err
		}
		rt.locals = append(rt.locals, local{name: name, value: obj})
		*numLocals = *numLocals + 1
		return true, nil
	case bytecode.PatternKindRecord:
		fieldNames, err := ToList(rt, p.items)
		if err != nil {
			return false, err
		}
		for _, fieldName := range fieldNames {
			name, err := ToString(rt, fieldName)
			if err != nil {
				return false, err
			}
			field, ok, err := FindRecordField(rt, obj, name)
			if err != nil {
				return false, err
			}
			if ok {
				rt.locals = append(rt.locals, local{name: name, value: field})
				*numLocals = *numLocals + 1
			} else {
				return false, nil
			}
		}
		return true, nil
	case bytecode.PatternKindTuple:
		objTuple, err := ToTuple(rt, obj)
		if err != nil {
			return false, err
		}
		patTuple, err := ToList(rt, p.items)
		if err != nil {
			return false, err
		}
		if len(objTuple) != len(patTuple) {
			return false, fmt.Errorf("tuple pattern should have exactly %d nested patterns", len(objTuple))
		}
		for i := 0; i < len(objTuple); i++ {
			match, err := rt.match(patTuple[i], objTuple[i], numLocals)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
		return true, nil
	default:
		if kDebug {
			return false, fmt.Errorf("loaded binary is corrupted (invalid pattern kind)")
		}
	}
	return false, nil
}

func (rt *Runtime) Stack() []string {
	stack := make([]string, len(rt.callStack))
	for i, id := range rt.callStack {
		stack[len(stack)-i-1] = rt.program.Strings[id]
	}
	return stack
}

type ModuleName string

type DefName string

type local struct {
	name  TString
	value Object
}

func pop[T any](stack *[]T) (x T, err error) {
	if kDebug && len(*stack) == 0 {
		err = fmt.Errorf("stack is empty")
		return
	}
	x = (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	return
}

func popX[T any](stack *[]T, n int) (xs []T, err error) {
	if kDebug && len(*stack) < n {
		err = fmt.Errorf("stack underflow")
		return
	}
	xs = (*stack)[len(*stack)-n:]
	*stack = (*stack)[:len(*stack)-n]
	return
}
