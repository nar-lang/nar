package runtime

import (
	"fmt"
	"nar-compiler/pkg/bytecode"
)

//TODO: make const objects when loading binary, and dont copy for every instance
//TODO: make string objects for every string in binary, and dont copy for every instance

func NewRuntime(program *bytecode.Binary) *Runtime {
	runtime := &Runtime{
		program: program,
		defs:    map[bytecode.FullIdentifier]Object{},
		scopes:  map[ModuleName]any{},
		arena:   newArena(256),
	}
	return runtime
}

type Runtime struct {
	program          *bytecode.Binary
	defs             map[bytecode.FullIdentifier]Object
	scopes           map[ModuleName]any
	awaitingDeps     []awaitingCallback
	arena            *Arena
	initialArenaSize uint64
}

type ModuleName string

type DefName string

type awaitingCallback struct {
	deps     []ModuleName
	callback func()
}

func (r *Runtime) Register(moduleName ModuleName, definitions map[DefName]Object, scope any) {
	for name, def := range definitions {
		r.defs[bytecode.FullIdentifier(moduleName)+"."+bytecode.FullIdentifier(name)] = def
	}
	r.scopes[moduleName] = scope
	r.checkAwaitingDeps()
}

func (r *Runtime) AfterRegistered(callback func(), deps ...ModuleName) {
	r.awaitingDeps = append(r.awaitingDeps, awaitingCallback{deps: deps, callback: callback})
	r.checkAwaitingDeps()
}

func (r *Runtime) checkAwaitingDeps() {
	for i := 0; i < len(r.awaitingDeps); i++ {
		ad := r.awaitingDeps[i]
		ready := true
		for _, dep := range ad.deps {
			if _, ok := r.scopes[dep]; !ok {
				ready = false
				break
			}
		}
		if ready {
			r.awaitingDeps = append(r.awaitingDeps[:i], r.awaitingDeps[i+1:]...)
			i--
			ad.callback()
		}
	}
}

func (r *Runtime) Scope(moduleName ModuleName) (any, bool) {
	s, ok := r.scopes[moduleName]
	return s, ok
}

func (r *Runtime) NewArena() *Arena {
	return newArena(64)
}

func (r *Runtime) Execute(defName bytecode.FullIdentifier, args ...Object) (Object, error) {
	return r.ExecuteInArena(r.arena, defName, args...)
}

func (r *Runtime) ExecuteInArena(arena *Arena, defName bytecode.FullIdentifier, args ...Object) (Object, error) {
	if len(arena.callStack) > 0 || len(arena.patternStack) > 0 {
		return invalidObject, fmt.Errorf("arena is already in use or was not cleared")
	}
	fnIndex, ok := r.program.Exports[defName]
	if !ok {
		return invalidObject, fmt.Errorf("definition `%s` is not exported by loaded binary", defName)
	}
	if kDebug && fnIndex >= bytecode.Pointer(len(r.program.Funcs)) {
		return invalidObject, fmt.Errorf("loaded binary is corrupted (invalid function pointer)")
	}
	fn := r.program.Funcs[fnIndex]
	if uint32(len(args)) != fn.NumArgs {
		return invalidObject, fmt.Errorf("function `%s` requires %d arguments, but %d given", defName, fn.NumArgs, len(args))
	}
	err := r.executeFunc(arena, fn)
	if err != nil {
		return invalidObject, err
	}
	return arena.objectStack[len(arena.objectStack)-1], nil
}

func (r *Runtime) ExecuteFunc(afn Object, args ...Object) (Object, error) {
	return r.ExecuteFuncInArena(r.arena, afn, args...)
}

func (r *Runtime) ExecuteFuncInArena(arena *Arena, fn Object, args ...Object) (Object, error) {
	afn, err := fn.asClosure()
	if err != nil {
		return invalidObject, err
	}
	curried, err := afn.curried.AsObjectList()
	if err != nil {
		return invalidObject, err
	}
	arena.objectStack = append(arena.objectStack, curried...)
	arena.objectStack = append(arena.objectStack, args...)
	numArgs := len(args) + len(curried)
	if afn.fn.NumArgs == uint32(numArgs) {
		err = r.executeFunc(arena, afn.fn)
		if err != nil {
			return invalidObject, err
		}
		result, err := pop(&arena.objectStack)
		if err != nil {
			return invalidObject, err
		}
		return result, nil
	} else {
		resultArgs, err := popX(&arena.objectStack, numArgs)
		if err != nil {
			return invalidObject, err
		}
		resultClosure := arena.newClosure(afn.fn, resultArgs...)
		return resultClosure, nil
	}
}

func (r *Runtime) executeFunc(arena *Arena, fn bytecode.Func) error {
	arena.callStack = append(arena.callStack, fn.Name)
	numLocals := 0
	numOps := len(fn.Ops)
	for index := 0; index < numOps; index++ {
		opKind, b, c, a := fn.Ops[index].Decompose()
		switch opKind {
		case bytecode.OpKindLoadLocal:
			if kDebug && a >= uint32(len(r.program.Strings)) {
				return fmt.Errorf("loaded binary is corrupted (invalid local name)")
			}
			name := r.program.Strings[a]
			found := false
			for i := len(arena.locals) - 1; i >= len(arena.locals)-numLocals; i-- {
				if arena.locals[i].name == name {
					arena.objectStack = append(arena.objectStack, arena.locals[i].value)
					found = true
					break
				}
			}
			if kDebug && !found {
				return fmt.Errorf("loaded binary is corrupted (undefined local `%s`)", name)
			}
		case bytecode.OpKindLoadGlobal:
			if kDebug && a >= uint32(len(r.program.Funcs)) {
				return fmt.Errorf("loaded binary is corrupted (invalid global pointer)")
			}
			glob := r.program.Funcs[a]
			if glob.NumArgs == 0 {
				if cached, ok := arena.cachedExpressions[bytecode.Pointer(a)]; !ok {
					var err error
					err = r.executeFunc(arena, glob)
					if err != nil {
						return err
					}
					if kDebug && len(arena.objectStack) == 0 {
						return fmt.Errorf("stack is empty after executing `%s`", r.program.Strings[glob.Name])

					}
					arena.cachedExpressions[bytecode.Pointer(a)] = arena.objectStack[len(arena.objectStack)-1]
				} else {
					arena.objectStack = append(arena.objectStack, cached)
				}
			} else {
				arena.objectStack = append(arena.objectStack, arena.newClosure(glob))
			}
		case bytecode.OpKindLoadConst:
			var value Object
			switch bytecode.ConstKind(b) {
			case bytecode.ConstKindUnit:
				value = arena.NewUnit()
			case bytecode.ConstKindChar:
				value = arena.NewChar(rune(a))
			case bytecode.ConstKindInt:
				if kDebug && a >= uint32(len(r.program.Consts)) {
					return fmt.Errorf("loaded binary is corrupted (invalid const index)")
				}
				value = arena.NewInt(r.program.Consts[a].Int())
			case bytecode.ConstKindFloat:
				if kDebug && a >= uint32(len(r.program.Consts)) {
					return fmt.Errorf("loaded binary is corrupted (invalid const index)")
				}
				value = arena.NewFloat(r.program.Consts[a].Float())
			case bytecode.ConstKindString:
				if kDebug && a >= uint32(len(r.program.Strings)) {
					return fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				value = arena.NewString(r.program.Strings[a])
			default:
				if kDebug {
					return fmt.Errorf("loaded binary is corrupted (invalid const kind)")
				}
			}
			switch bytecode.StackKind(c) {
			case bytecode.StackKindObject:
				arena.objectStack = append(arena.objectStack, value)
			case bytecode.StackKindPattern:
				arena.patternStack = append(arena.patternStack, value)
			default:
				if kDebug {
					return fmt.Errorf("loaded binary is corrupted (invalid stack kind)")
				}
			}
		case bytecode.OpKindApply:
			x, err := pop(&arena.objectStack)
			if err != nil {
				return err
			}
			afn, err := x.asClosure()
			if err != nil {
				return err
			}
			numArgs := int(b)
			start := len(arena.objectStack) - numArgs
			if kDebug && start < 0 {
				return fmt.Errorf("stack underflow when trying to apply function")
			}
			curried, err := afn.curried.AsObjectList()
			if err != nil {
				return err
			}
			if len(curried) > 0 {
				arena.objectStack = append(arena.objectStack[:start], append(curried, arena.objectStack[start:]...)...)
			}
			if int(afn.fn.NumArgs) == numArgs {
				err = r.executeFunc(arena, afn.fn)
				if err != nil {
					return err
				}
			} else {
				arena.objectStack = append(
					arena.objectStack[:start],
					arena.newClosure(afn.fn, arena.NewObjectList(arena.objectStack[start:]...)))
			}
		case bytecode.OpKindCall:
			if kDebug && a >= uint32(len(r.program.Strings)) {
				return fmt.Errorf("loaded binary is corrupted (invalid string index)")
			}
			name := r.program.Strings[a]
			def, ok := r.defs[bytecode.FullIdentifier(name)]
			if !ok {
				return fmt.Errorf("definition `%s` is not registered", name)
			}
			var err error
			arena.objectStack, err = def.call(arena.objectStack)
			if err != nil {
				return err
			}
		case bytecode.OpKindJump:
			if b == 0 {
				index += int(a)
			} else {
				pattern, err := pop(&arena.patternStack)
				if err != nil {
					return err
				}
				obj, err := pop(&arena.objectStack)
				if err != nil {
					return err
				}
				match, err := r.match(arena, pattern, obj, &numLocals)
				if err != nil {
					return err
				}
				if !match {
					if kDebug && a == 0 {
						return fmt.Errorf("pattern match with jump delta 0 should not fail")
					}
					index += int(a)
				}
			}
		case bytecode.OpKindMakeObject:
			switch bytecode.ObjectKind(b) {
			case bytecode.ObjectKindList:
				items, err := popX(&arena.objectStack, int(a))
				if err != nil {
					return err
				}
				arena.objectStack = append(arena.objectStack, arena.NewObjectList(items...))
			case bytecode.ObjectKindTuple:
				items, err := popX(&arena.objectStack, int(a))
				if err != nil {
					return err
				}
				arena.objectStack = append(arena.objectStack, arena.NewObjectTuple(items...))
			case bytecode.ObjectKindRecord:
				items, err := popX(&arena.objectStack, int(a*2))
				if err != nil {
					return err
				}
				arena.objectStack = append(arena.objectStack, arena.newObjectRecord(items...))
			case bytecode.ObjectKindOption:
				nameObj, err := pop(&arena.objectStack)
				if err != nil {
					return err
				}
				name, err := nameObj.AsString()
				if err != nil {
					return err
				}
				values, err := popX(&arena.objectStack, int(a))
				if err != nil {
					return err
				}
				arena.objectStack = append(arena.objectStack, arena.NewObjectOption(name, values...))
			default:
				if kDebug {
					return fmt.Errorf("loaded binary is corrupted (invalid object kind)")
				}
			}
		case bytecode.OpKindMakePattern:
			name := arena.NewString("")
			var items []Object
			var err error
			switch bytecode.PatternKind(b) {
			case bytecode.PatternKindAlias:
				if kDebug && a >= uint32(len(r.program.Strings)) {
					return fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				name = arena.NewString(r.program.Strings[a])
				items, err = popX(&arena.patternStack, 1)
			case bytecode.PatternKindAny:
				break
			case bytecode.PatternKindCons:
				items, err = popX(&arena.patternStack, 2)
			case bytecode.PatternKindConst:
				if kDebug && len(arena.objectStack) == 0 {
					return fmt.Errorf("stack is empty when trying to make const")
				}
				items, err = popX(&arena.patternStack, 1)
			case bytecode.PatternKindDataOption:
				if kDebug && a >= uint32(len(r.program.Strings)) {
					return fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				name = arena.NewString(r.program.Strings[a])
				items, err = popX(&arena.patternStack, int(c))
			case bytecode.PatternKindList:
				items, err = popX(&arena.patternStack, int(c)) //TODO: use a register for list length
			case bytecode.PatternKindNamed:
				if kDebug && a >= uint32(len(r.program.Strings)) {
					return fmt.Errorf("loaded binary is corrupted (invalid string index)")
				}
				name = arena.NewString(r.program.Strings[a])
			case bytecode.PatternKindRecord:
				items, err = popX(&arena.patternStack, int(c*2)) //TODO: use a register for list length
			case bytecode.PatternKindTuple:
				items, err = popX(&arena.patternStack, int(c))
			default:
				if kDebug {
					return fmt.Errorf("loaded binary is corrupted (invalid pattern kind)")
				}
			}
			if err != nil {
				return err
			}
			p, err := arena.newPattern(name, items)
			if err != nil {
				return err
			}
			arena.patternStack = append(arena.patternStack, p)
		case bytecode.OpKindAccess:
			if kDebug && a >= uint32(len(r.program.Strings)) {
				return fmt.Errorf("loaded binary is corrupted (invalid string index)")
			}
			if kDebug && len(arena.objectStack) == 0 {
				return fmt.Errorf("stack is empty when trying to access record field")
			}
			name := r.program.Strings[a]
			record := arena.objectStack[len(arena.objectStack)-1]
			arena.objectStack = arena.objectStack[:len(arena.objectStack)-1]
			field, ok, err := record.FindField(name)
			if err != nil {
				return err
			}
			if kDebug && !ok {
				return fmt.Errorf("record does not have field `%s`", name)
			}
			arena.objectStack = append(arena.objectStack, field)
		case bytecode.OpKindUpdate:
			if kDebug && a >= uint32(len(r.program.Strings)) {
				return fmt.Errorf("loaded binary is corrupted (invalid string index)")
			}
			if kDebug && len(arena.objectStack) < 2 {
				return fmt.Errorf("stack underflow when trying to update record field")
			}
			key := r.program.Strings[a]
			value := arena.objectStack[len(arena.objectStack)-1]
			record := arena.objectStack[len(arena.objectStack)-2]
			arena.objectStack = arena.objectStack[:len(arena.objectStack)-2]
			updated, err := record.UpdateField(arena.NewString(key), value)
			if err != nil {
				return err
			}
			arena.objectStack = append(arena.objectStack, updated)
		case bytecode.OpKindSwapPop:
			switch bytecode.SwapPopMode(b) {
			case bytecode.SwapPopModeBoth:
				if kDebug && len(arena.objectStack) < 2 {
					return fmt.Errorf("stack underflow when trying to swap and pop")
				}
				arena.objectStack = append(arena.objectStack[:len(arena.objectStack)-2], arena.objectStack[len(arena.objectStack)-1])
			case bytecode.SwapPopModePop:
				if kDebug && len(arena.objectStack) == 0 {
					return fmt.Errorf("stack is empty when trying to pop")
				}
				arena.objectStack = arena.objectStack[:len(arena.objectStack)-1]
			default:
				if kDebug {
					return fmt.Errorf("loaded binary is corrupted (invalid swap pop mode)")
				}
			}
		default:
			if kDebug {
				return fmt.Errorf("loaded binary is corrupted (invalid op kind)")
			}
		}
	}
	_, err := popX(&arena.locals, numLocals)
	if err != nil {
		return err
	}
	_, err = pop(&arena.callStack)
	if err != nil {
		return err
	}
	return nil
}

func (r *Runtime) match(arena *Arena, pattern Object, obj Object, numLocals *int) (bool, error) {
	p, err := pattern.asPattern()
	if err != nil {
		return false, err
	}
	switch p.kind {
	case bytecode.PatternKindAlias:
		name, err := p.name.AsString()
		if err != nil {
			return false, err
		}
		arena.locals = append(arena.locals, local{name: name, value: obj})
		*numLocals = *numLocals + 1
		nested, err := p.items.AsObjectList()
		if err != nil {
			return false, err
		}
		if kDebug && len(nested) != 1 {
			return false, fmt.Errorf("alias pattern should have exactly one nested pattern")
		}
		return r.match(arena, nested[0], obj, numLocals)
	case bytecode.PatternKindAny:
		return true, nil
	case bytecode.PatternKindCons:
		nested, err := p.items.AsObjectList()
		if err != nil {
			return false, err
		}
		if kDebug && len(nested) != 2 {
			return false, fmt.Errorf("cons pattern should have exactly two nested patterns")
		}
		list, err := obj.AsObjectList()
		if err != nil {
			return false, err
		}
		if len(list) < 1 {
			return false, fmt.Errorf("cons pattern should match non-empty list")
		}
		match, err := r.match(arena, nested[1], list[0], numLocals)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
		return r.match(arena, nested[0], arena.NewObjectList(list[1:]...), numLocals) //TODO: optimize: do not create new list, use low level API
	case bytecode.PatternKindConst:
		nested, err := p.items.AsObjectList()
		if err != nil {
			return false, err
		}
		if kDebug && len(nested) != 1 {
			return false, fmt.Errorf("const pattern should have exactly one nested pattern")
		}
		return obj.ConstEqualsTo(nested[0])
	case bytecode.PatternKindDataOption:
		objName, objValues, err := obj.asObjectOption()
		if err != nil {
			return false, err
		}
		eq, err := objName.ConstEqualsTo(p.name)
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
		return r.match(arena, p.items, objValues, numLocals)
	case bytecode.PatternKindList:
		objList, err := obj.AsObjectList()
		if err != nil {
			return false, err
		}
		patList, err := p.items.AsObjectList()
		if err != nil {
			return false, err
		}
		if len(objList) != len(patList) {
			return false, nil
		}
		for i := 0; i < len(objList); i++ {
			match, err := r.match(arena, patList[i], objList[i], numLocals)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
		return true, nil
	case bytecode.PatternKindNamed:
		name, err := p.name.AsString()
		if err != nil {
			return false, err
		}
		arena.locals = append(arena.locals, local{name: name, value: obj})
		*numLocals = *numLocals + 1
		return true, nil
	case bytecode.PatternKindRecord:
		fieldNames, err := p.items.AsObjectList()
		if err != nil {
			return false, err
		}
		for _, fieldName := range fieldNames {
			name, err := fieldName.AsString()
			if err != nil {
				return false, err
			}
			field, ok, err := obj.FindField(name)
			if err != nil {
				return false, err
			}
			if ok {
				arena.locals = append(arena.locals, local{name: name, value: field})
				*numLocals = *numLocals + 1
			} else {
				return false, nil
			}
		}
		return true, nil
	case bytecode.PatternKindTuple:
		objTuple, err := obj.AsObjectTuple()
		if err != nil {
			return false, err
		}
		patTuple, err := p.items.AsObjectList()
		if err != nil {
			return false, err
		}
		if len(objTuple) != len(patTuple) {
			return false, fmt.Errorf("tuple pattern should have exactly %d nested patterns", len(objTuple))
		}
		for i := 0; i < len(objTuple); i++ {
			match, err := r.match(arena, patTuple[i], objTuple[i], numLocals)
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
