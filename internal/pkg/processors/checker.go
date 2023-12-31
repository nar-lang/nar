package processors

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"slices"
	"strings"
)

func CheckPatterns(modules map[ast.QualifiedIdentifier]*typed.Module) (errors []error) {
	var names []ast.QualifiedIdentifier
	for name := range modules {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		module := modules[name]
		for _, definition := range module.Definitions {
			err := checkDefinition(definition)
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}
	return
}

func checkDefinition(definition *typed.Definition) error {
	for _, pattern := range definition.Params {
		if err := checkPattern(pattern); err != nil {
			return err
		}
	}
	return checkExpression(definition.Expression)
}

func checkExpression(expression typed.Expression) error {
	if expression == nil {
		return nil
	}
	switch expression.(type) {
	case *typed.Select:
		e := expression.(*typed.Select)
		if err := checkExpression(e.Condition); err != nil {
			return err
		}

		if err := checkPatterns(common.Map(func(cs typed.SelectCase) typed.Pattern {
			return cs.Pattern
		}, e.Cases)); err != nil {
			return err
		}

		for _, cs := range e.Cases {
			if err := checkExpression(cs.Expression); err != nil {
				return err
			}
		}
		return nil
	case *typed.Let:
		e := expression.(*typed.Let)
		if err := checkPattern(e.Pattern); err != nil {
			return err
		}
		if err := checkExpression(e.Value); err != nil {
			return err
		}
		return checkExpression(e.Body)
	case *typed.Access:
		e := expression.(*typed.Access)
		return checkExpression(e.Record)
	case *typed.Apply:
		e := expression.(*typed.Apply)
		if err := checkExpression(e.Func); err != nil {
			return err
		}
		for _, arg := range e.Args {
			if err := checkExpression(arg); err != nil {
				return err
			}
		}
		return nil
	case *typed.Const:
		return nil
	case *typed.List:
		e := expression.(*typed.List)
		for _, item := range e.Items {
			if err := checkExpression(item); err != nil {
				return err
			}
		}
		return nil
	case *typed.Record:
		e := expression.(*typed.Record)
		for _, field := range e.Fields {
			if err := checkExpression(field.Value); err != nil {
				return err
			}
		}
		return nil
	case *typed.Tuple:
		e := expression.(*typed.Tuple)
		for _, item := range e.Items {
			if err := checkExpression(item); err != nil {
				return err
			}
		}
		return nil
	case *typed.UpdateLocal:
		e := expression.(*typed.UpdateLocal)
		for _, field := range e.Fields {
			if err := checkExpression(field.Value); err != nil {
				return err
			}
		}
		return nil
	case *typed.UpdateGlobal:
		e := expression.(*typed.UpdateGlobal)
		for _, field := range e.Fields {
			if err := checkExpression(field.Value); err != nil {
				return err
			}
		}
		return nil
	case *typed.Constructor:
		e := expression.(*typed.Constructor)
		for _, arg := range e.Args {
			if err := checkExpression(arg); err != nil {
				return err
			}
		}
		return nil
	case *typed.NativeCall:
		e := expression.(*typed.NativeCall)
		for _, arg := range e.Args {
			if err := checkExpression(arg); err != nil {
				return err
			}
		}
		return nil
	case *typed.Local:
		return nil
	case *typed.Global:
		return nil
	}
	return common.NewCompilerError("impossible case")
}

func checkPattern(pattern typed.Pattern) error {
	return checkPatterns([]typed.Pattern{pattern})
}

func checkPatterns(patterns []typed.Pattern) error {
	if matrix, redundant, err := toNonRedundantRows(patterns); err != nil {
		return err
	} else if len(redundant) > 0 {
		return common.Error{
			Location: redundant[0].GetLocation(),
			Extra:    common.Map(func(p typed.Pattern) ast.Location { return p.GetLocation() }, redundant[1:]),
			Message:  "pattern matching is redundant",
		}
	} else {
		missingPatterns, err := isExhaustive(matrix, 1)
		if err != nil {
			return err
		}
		if len(missingPatterns) > 0 {
			return common.Error{
				Location: patterns[len(patterns)-1].GetLocation(),
				Message: "pattern matching is not exhaustive, missing patterns: \n\t" +
					strings.Join(
						common.Map(func(r []Pattern) string { return common.Join(r, ", ") }, missingPatterns),
						"\n\t"),
			}
		}
	}
	return nil
}

func toNonRedundantRows(patterns []typed.Pattern) ([][]Pattern, []typed.Pattern, error) {
	var matrix [][]Pattern
	var redundant []typed.Pattern
	for _, pattern := range patterns {
		simplified, err := simplifyPattern(pattern)
		if err != nil {
			return nil, nil, err
		}
		row := []Pattern{simplified}
		useful, err := isUseful(matrix, row)
		if err != nil {
			return nil, nil, err
		}
		if useful {
			matrix = append(matrix, row)
		} else {
			redundant = append(redundant, pattern)
		}
	}
	return matrix, redundant, nil
}

func isUseful(matrix [][]Pattern, vector []Pattern) (bool, error) {
	if len(matrix) == 0 {
		return true, nil
	} else {
		if len(vector) == 0 {
			return false, nil
		} else {
			switch vector[0].(type) {
			case PatternConstructor:
				e := vector[0].(PatternConstructor)
				option, err := e.Option()
				if err != nil {
					return false, err
				}
				patterns, err := common.MapIfError(specializeRowByCtor(option), matrix)
				if err != nil {
					return false, err
				}
				return isUseful(patterns, append(e.Args, vector[1:]...))
			case PatternAnything:
				if alts, ok := isComplete(matrix); ok {
					isUsefulAlt := func(c typed.DataOption) (bool, error) {
						patterns, err := common.MapIfError(specializeRowByCtor(c), matrix)
						if err != nil {
							return false, err
						}
						return isUseful(patterns,
							append(common.Repeat(Pattern(PatternAnything{}), len(c.Values)), vector[1:]...))
					}
					return common.AnyError(isUsefulAlt, alts)
				} else {
					patterns, err := common.MapIfError(specializeRowByAnything, matrix)
					if err != nil {
						return false, err
					}
					return isUseful(patterns, vector[1:])
				}
			case PatternLiteral:
				e := vector[0].(PatternLiteral)
				patterns, err := common.MapIfError(specializeRowByLiteral(e), matrix)
				if err != nil {
					return false, err
				}
				return isUseful(patterns, vector[1:])
			}
			return false, common.NewCompilerError("impossible case")
		}
	}
}

func specializeRowByCtor(ctor typed.DataOption) func(row []Pattern) ([]Pattern, bool, error) {
	return func(row []Pattern) ([]Pattern, bool, error) {
		if len(row) == 0 {
			return nil, false, common.NewCompilerError("Empty matrices should not get specialized.")
		} else {
			switch row[0].(type) {
			case PatternConstructor:
				e := row[0].(PatternConstructor)
				if e.Name == ctor.Name {
					return append(e.Args, row[1:]...), true, nil
				} else {
					return nil, false, nil
				}
			case PatternAnything:
				return append(common.Repeat(Pattern(PatternAnything{}), len(ctor.Values)), row[1:]...), true, nil
			case PatternLiteral:
				return nil, false, common.NewCompilerError("After type checking, constructors and literals" +
					" should never align in pattern match exhaustiveness checks.")
			}
			return nil, false, common.NewCompilerError("impossible case")
		}
	}
}

func specializeRowByAnything(row []Pattern) ([]Pattern, bool, error) {
	if len(row) == 0 {
		return nil, false, nil
	} else {
		switch row[0].(type) {
		case PatternConstructor:
			return nil, false, nil
		case PatternAnything:
			return row[1:], true, nil
		case PatternLiteral:
			return nil, false, nil
		}
		return nil, false, common.NewCompilerError("impossible case")
	}
}

func specializeRowByLiteral(literal PatternLiteral) func(row []Pattern) ([]Pattern, bool, error) {
	return func(row []Pattern) ([]Pattern, bool, error) {
		if len(row) == 0 {
			return nil, false, common.NewCompilerError("Empty matrices should not get specialized.")
		} else {
			switch row[0].(type) {
			case PatternConstructor:
				return nil, false, common.NewCompilerError("After type checking, constructors and literals" +
					" should never align in pattern match exhaustiveness checks.")
			case PatternAnything:
				return row[1:], true, nil
			case PatternLiteral:
				e := row[0].(PatternLiteral)
				if e.Literal.EqualsTo(literal.Literal) {
					return row[1:], true, nil
				} else {
					return nil, false, nil
				}
			}
			return nil, false, common.NewCompilerError("impossible case")
		}
	}
}

func isComplete(matrix [][]Pattern) ([]typed.DataOption, bool) {
	ctors := collectCtors(matrix)
	t := firstCtor(ctors)
	if t == nil {
		return nil, false
	}
	if len(t.Options) == len(ctors) {
		return t.Options, true
	} else {
		return nil, false
	}
}

func firstCtor(ctors map[ast.DataOptionIdentifier]*typed.TData) *typed.TData {
	var minKey ast.DataOptionIdentifier
	for key := range ctors {
		if key < minKey || minKey == "" {
			minKey = key
		}
	}
	if minKey == "" {
		return nil
	}
	return ctors[minKey]
}

func collectCtors(matrix [][]Pattern) map[ast.DataOptionIdentifier]*typed.TData {
	ctors := map[ast.DataOptionIdentifier]*typed.TData{}
	for _, row := range matrix {
		if row == nil {
			return nil
		}
		if c, ok := row[0].(PatternConstructor); ok {
			ctors[c.Name] = c.Union
		}
	}
	return ctors
}

func isExhaustive(matrix [][]Pattern, n int) (missing [][]Pattern, err error) {
	if len(matrix) == 0 {
		return [][]Pattern{common.Repeat(Pattern(PatternAnything{}), n)}, nil
	}
	if n == 0 {
		return nil, nil
	}
	ctors := collectCtors(matrix)
	numSeen := len(ctors)
	if numSeen == 0 {
		patterns, err := common.MapIfError(specializeRowByAnything, matrix)
		if err != nil {
			return nil, err
		}
		exhaustive, err := isExhaustive(patterns, n-1)
		if err != nil {
			return nil, err
		}
		return common.Map(
			func(row []Pattern) []Pattern {
				return append([]Pattern{PatternAnything{}}, row...)
			},
			exhaustive), nil
	}
	alts := firstCtor(ctors)
	altList := alts.Options
	numAlts := len(altList)
	if numSeen < numAlts {
		patterns, err := common.MapIfError(specializeRowByAnything, matrix)
		if err != nil {
			return nil, err
		}
		matrix, err = isExhaustive(patterns, n-1)
		if err != nil {
			return nil, err
		}
		rest := common.MapIf(isMissing(alts, ctors), altList)
		for i, row := range matrix {
			if i < len(rest) {
				matrix[i] = append([]Pattern{rest[i]}, row...)
			}
		}
		n = len(rest)
		if len(matrix) < n {
			n = len(matrix)
		}
		return matrix[:n], nil
	} else {
		isAltExhaustive := func(alt typed.DataOption) ([][]Pattern, error) {
			patterns, err := common.MapIfError(specializeRowByCtor(alt), matrix)
			if err != nil {
				return nil, err
			}
			mx, err := isExhaustive(patterns, len(alt.Values)+n-1)
			if err != nil {
				return nil, err
			}
			for i, row := range mx {
				mx[i] = append(recoverCtor(alts, alt, row), row...)
			}
			return mx, nil
		}
		return common.ConcatMapError(isAltExhaustive, altList)
	}
}

func isMissing(union *typed.TData, ctors map[ast.DataOptionIdentifier]*typed.TData) func(alt typed.DataOption) (Pattern, bool) {
	return func(alt typed.DataOption) (Pattern, bool) {
		if _, ok := ctors[alt.Name]; ok {
			return nil, false
		} else {
			return PatternConstructor{
				Union: union,
				Name:  alt.Name,
				Args:  common.Repeat(Pattern(PatternAnything{}), len(alt.Values)),
			}, true
		}
	}
}

func recoverCtor(union *typed.TData, alt typed.DataOption, patterns []Pattern) []Pattern {
	args := patterns[:len(alt.Values)]
	rest := patterns[len(alt.Values):]
	return append([]Pattern{
		PatternConstructor{
			Union: union,
			Name:  alt.Name,
			Args:  args,
		},
	}, rest...)
}

func simplifyPattern(pattern typed.Pattern) (Pattern, error) {
	switch pattern.(type) {
	case *typed.PAny:
		return PatternAnything{}, nil
	case *typed.PNamed:
		return PatternAnything{}, nil
	case *typed.PRecord:
		return PatternAnything{}, nil
	case *typed.PConst:
		e := pattern.(*typed.PConst)
		if _, ok := e.Value.(ast.CUnit); ok {
			return PatternConstructor{
				Union: &typed.TData{
					Location: e.Location,
					Name:     "!!Unit",
					Options:  []typed.DataOption{{Name: "Only"}},
				},
				Name: "Only",
			}, nil
		}
		return PatternLiteral{e.Value}, nil
	case *typed.PTuple:
		e := pattern.(*typed.PTuple)
		args, err := common.MapError(simplifyPattern, e.Items)
		if err != nil {
			return nil, err
		}
		return PatternConstructor{
			Union: &typed.TData{
				Location: e.Location,
				Name:     ast.FullIdentifier(fmt.Sprintf("!!%d", len(e.Items))),
				Options: []typed.DataOption{
					{
						Name: "Only",
						Values: common.Map(
							func(i typed.Pattern) typed.Type { return i.GetType() },
							e.Items,
						),
					},
				},
			},
			Name: "Only",
			Args: args,
		}, nil
	case *typed.PDataOption:
		e := pattern.(*typed.PDataOption)
		args, err := common.MapError(simplifyPattern, e.Args)
		if err != nil {
			return nil, err
		}
		if dataType, ok := e.Type.(*typed.TData); ok {
			return PatternConstructor{
				Union: dataType,
				Name:  common.MakeDataOptionIdentifier(e.DataName, e.OptionName),
				Args:  args,
			}, nil
		} else {
			return nil, common.NewCompilerError("Data option pattern should have a data type.")
		}
	case *typed.PList:
		e := pattern.(*typed.PList)
		var nested []Pattern
		ctor := "Nil"
		if len(e.Items) > 0 {
			item, err := simplifyPattern(&typed.PList{
				Location: e.Location,
				Type:     e.Type,
				Items:    e.Items[1:],
			})
			if err != nil {
				return nil, err
			}
			ctor = "Cons"
			nested = []Pattern{item}
		}
		unboundIndex++
		a := typed.Type(&typed.TUnbound{
			Location: e.Location,
			Index:    unboundIndex,
		})
		return PatternConstructor{
			Union: &typed.TData{
				Location: e.Location,
				Name:     "!!list",
				Options: []typed.DataOption{
					{Name: "Nil"},
					{Name: "Cons", Values: []typed.Type{a, &typed.TNative{
						Location: e.Location,
						Name:     common.NarCoreListList,
						Args:     []typed.Type{a},
					}}},
				},
			},
			Name: ast.DataOptionIdentifier(ctor),
			Args: nested,
		}, nil
	case *typed.PCons:
		e := pattern.(*typed.PCons)
		a := typed.Type(&typed.TUnbound{
			Location: e.Location,
			Index:    unboundIndex,
		})
		head, err := simplifyPattern(e.Head)
		if err != nil {
			return nil, err
		}
		tail, err := simplifyPattern(e.Tail)
		if err != nil {
			return nil, err
		}
		return PatternConstructor{
			Union: &typed.TData{
				Location: e.Location,
				Name:     "!!list",
				Options: []typed.DataOption{
					{Name: "Nil"},
					{Name: "Cons", Values: []typed.Type{a, &typed.TNative{
						Location: e.Location,
						Name:     common.NarCoreListList,
						Args:     []typed.Type{a},
					}}},
				},
			},
			Name: "Cons",
			Args: []Pattern{head, tail},
		}, nil
	case *typed.PAlias:
		e := pattern.(*typed.PAlias)
		return simplifyPattern(e.Nested)
	}
	return nil, common.NewCompilerError("impossible case")
}

type Pattern interface {
	fmt.Stringer
	_pattern()
}

type PatternAnything struct{}

func (PatternAnything) _pattern() {}

func (PatternAnything) String() string {
	return "_"
}

type PatternLiteral struct {
	Literal ast.ConstValue
}

func (PatternLiteral) _pattern() {}

func (p PatternLiteral) String() string {
	return p.Literal.String()
}

type PatternConstructor struct {
	Union *typed.TData
	Name  ast.DataOptionIdentifier
	Args  []Pattern
}

func (PatternConstructor) _pattern() {}

func (c PatternConstructor) String() string {
	params := common.Join(c.Args, ", ")
	if params != "" {
		params = fmt.Sprintf("(%s)", params)
	}
	return fmt.Sprintf("%s%s", c.Name, params)
}

func (c PatternConstructor) Option() (typed.DataOption, error) {
	for _, o := range c.Union.Options {
		if o.Name == c.Name {
			return o, nil
		}
	}
	return typed.DataOption{}, common.NewCompilerError("option not found")
}
