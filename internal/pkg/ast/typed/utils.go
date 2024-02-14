package typed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/common"
	"strings"
)

func checkPattern(pattern Pattern) error {
	return checkPatterns([]Pattern{pattern})
}

func checkPatterns(patterns []Pattern) error {
	if matrix, redundant, err := toNonRedundantRows(patterns); err != nil {
		return err
	} else if len(redundant) > 0 {
		return common.Error{
			Location: redundant[0].Location(),
			Extra:    common.Map(func(p Pattern) ast.Location { return p.Location() }, redundant[1:]),
			Message:  "pattern matching is redundant",
		}
	} else {
		missingPatterns, err := isExhaustive(matrix, 1)
		if err != nil {
			return err
		}
		if len(missingPatterns) > 0 {
			return common.Error{
				Location: patterns[len(patterns)-1].Location(),
				Message: "pattern matching is not exhaustive, missing patterns: \n\t" +
					strings.Join(
						common.Map(func(r []simplePattern) string { return common.Join(r, ", ") }, missingPatterns),
						"\n\t"),
			}
		}
	}
	return nil
}

func toNonRedundantRows(patterns []Pattern) ([][]simplePattern, []Pattern, error) {
	var matrix [][]simplePattern
	var redundant []Pattern
	for _, pattern := range patterns {
		simplified := pattern.simplify()
		row := []simplePattern{simplified}
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

func isUseful(matrix [][]simplePattern, vector []simplePattern) (bool, error) {
	if len(matrix) == 0 {
		return true, nil
	} else {
		if len(vector) == 0 {
			return false, nil
		} else {
			switch vector[0].(type) {
			case simpleConstructor:
				e := vector[0].(simpleConstructor)
				option, err := e.Option()
				if err != nil {
					return false, err
				}
				patterns, err := common.MapIfError(specializeRowByCtor(option), matrix)
				if err != nil {
					return false, err
				}
				return isUseful(patterns, append(e.Args, vector[1:]...))
			case simpleAnything:
				if alts, ok := isComplete(matrix); ok {
					isUsefulAlt := func(c *DataOption) (bool, error) {
						patterns, err := common.MapIfError(specializeRowByCtor(c), matrix)
						if err != nil {
							return false, err
						}
						return isUseful(patterns,
							append(common.Repeat(simplePattern(simpleAnything{}), len(c.values)), vector[1:]...))
					}
					return common.AnyError(isUsefulAlt, alts)
				} else {
					patterns, err := common.MapIfError(specializeRowByAnything, matrix)
					if err != nil {
						return false, err
					}
					return isUseful(patterns, vector[1:])
				}
			case simpleLiteral:
				e := vector[0].(simpleLiteral)
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

func specializeRowByCtor(ctor *DataOption) func(row []simplePattern) ([]simplePattern, bool, error) {
	return func(row []simplePattern) ([]simplePattern, bool, error) {
		if len(row) == 0 {
			return nil, false, common.NewCompilerError("Empty matrices should not get specialized.")
		} else {
			switch row[0].(type) {
			case simpleConstructor:
				e := row[0].(simpleConstructor)
				if e.Name == ctor.name {
					return append(e.Args, row[1:]...), true, nil
				} else {
					return nil, false, nil
				}
			case simpleAnything:
				return append(common.Repeat(simplePattern(simpleAnything{}), len(ctor.values)), row[1:]...), true, nil
			case simpleLiteral:
				return nil, false, common.NewCompilerError("After type checking, constructors and literals" +
					" should never align in pattern match exhaustiveness checks.")
			}
			return nil, false, common.NewCompilerError("impossible case")
		}
	}
}

func specializeRowByAnything(row []simplePattern) ([]simplePattern, bool, error) {
	if len(row) == 0 {
		return nil, false, nil
	} else {
		switch row[0].(type) {
		case simpleConstructor:
			return nil, false, nil
		case simpleAnything:
			return row[1:], true, nil
		case simpleLiteral:
			return nil, false, nil
		}
		return nil, false, common.NewCompilerError("impossible case")
	}
}

func specializeRowByLiteral(literal simpleLiteral) func(row []simplePattern) ([]simplePattern, bool, error) {
	return func(row []simplePattern) ([]simplePattern, bool, error) {
		if len(row) == 0 {
			return nil, false, common.NewCompilerError("Empty matrices should not get specialized.")
		} else {
			switch row[0].(type) {
			case simpleConstructor:
				return nil, false, common.NewCompilerError("After type checking, constructors and literals" +
					" should never align in pattern match exhaustiveness checks.")
			case simpleAnything:
				return row[1:], true, nil
			case simpleLiteral:
				e := row[0].(simpleLiteral)
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

func isComplete(matrix [][]simplePattern) ([]*DataOption, bool) {
	ctors := collectCtors(matrix)
	t := firstCtor(ctors)
	if t == nil {
		return nil, false
	}
	if len(t.options) == len(ctors) {
		return t.options, true
	} else {
		return nil, false
	}
}

func firstCtor(ctors map[ast.DataOptionIdentifier]*TData) *TData {
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

func collectCtors(matrix [][]simplePattern) map[ast.DataOptionIdentifier]*TData {
	ctors := map[ast.DataOptionIdentifier]*TData{}
	for _, row := range matrix {
		if row == nil {
			return nil
		}
		if c, ok := row[0].(simpleConstructor); ok {
			ctors[c.Name] = c.Union
		}
	}
	return ctors
}

func isExhaustive(matrix [][]simplePattern, n int) (missing [][]simplePattern, err error) {
	if len(matrix) == 0 {
		return [][]simplePattern{common.Repeat(simplePattern(simpleAnything{}), n)}, nil
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
			func(row []simplePattern) []simplePattern {
				return append([]simplePattern{simpleAnything{}}, row...)
			},
			exhaustive), nil
	}
	alts := firstCtor(ctors)
	altList := alts.options
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
				matrix[i] = append([]simplePattern{rest[i]}, row...)
			}
		}
		n = len(rest)
		if len(matrix) < n {
			n = len(matrix)
		}
		return matrix[:n], nil
	} else {
		isAltExhaustive := func(alt *DataOption) ([][]simplePattern, error) {
			patterns, err := common.MapIfError(specializeRowByCtor(alt), matrix)
			if err != nil {
				return nil, err
			}
			mx, err := isExhaustive(patterns, len(alt.values)+n-1)
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

func isMissing(union *TData, ctors map[ast.DataOptionIdentifier]*TData) func(alt *DataOption) (simplePattern, bool) {
	return func(alt *DataOption) (simplePattern, bool) {
		if _, ok := ctors[alt.name]; ok {
			return nil, false
		} else {
			return simpleConstructor{
				Union: union,
				Name:  alt.name,
				Args:  common.Repeat(simplePattern(simpleAnything{}), len(alt.values)),
			}, true
		}
	}
}

func recoverCtor(union *TData, alt *DataOption, patterns []simplePattern) []simplePattern {
	args := patterns[:len(alt.values)]
	rest := patterns[len(alt.values):]
	return append([]simplePattern{
		simpleConstructor{
			Union: union,
			Name:  alt.name,
			Args:  args,
		},
	}, rest...)
}

func getConstType(ctx *SolvingContext, cv ast.ConstValue, src annotationSource) Type {
	switch cv.(type) {
	case ast.CChar:
		return NewTNative(src.Location(), common.NarBaseCharChar, nil)
	case ast.CInt:
		return ctx.newAnnotatedConstraint(src, nil, ast.Identifier(common.ConstraintNumber))
	case ast.CFloat:
		return NewTNative(src.Location(), common.NarBaseMathFloat, nil)
	case ast.CString:
		return NewTNative(src.Location(), common.NarBaseStringString, nil)
	case ast.CUnit:
		return NewTNative(src.Location(), common.NarBaseBasicsUnit, nil)
	}
	panic("normalized.getConstType(): switch not exhaustive")
}

func newTypeMatchError(loc ast.Location, a, t Type) error {
	return common.NewError(loc, "cannot match %s and %s", t.Code(""), a.Code(""))
}
