package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"slices"
	"strings"
	"unicode"
)

func findParsedType(
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	name ast.QualifiedIdentifier,
	args []Type,
	loc ast.Location,
) (Type, *Module, []ast.FullIdentifier, error) {
	var aliasNameEq = func(x *Alias) bool {
		return ast.QualifiedIdentifier(x.name) == name
	}

	// 1. check current module
	if alias, ok := common.Find(aliasNameEq, module.aliases); ok {
		id := common.MakeFullIdentifier(module.name, alias.name)
		if alias.type_ == nil {
			return NewTNative(loc, id, args), module, []ast.FullIdentifier{id}, nil
		}
		if len(alias.params) != len(args) {
			return nil, nil, nil, nil
		}
		typeMap := map[ast.Identifier]Type{}
		for i, x := range alias.params {
			typeMap[x] = args[i]
		}
		withAppliedArgs, err := applyTypeArgs(alias.type_, typeMap, loc)
		if err != nil {
			return nil, nil, nil, err
		}
		return withAppliedArgs, module, []ast.FullIdentifier{id}, nil
	}

	lastDot := strings.LastIndex(string(name), ".")
	typeName := name[lastDot+1:]
	modName := ""
	if lastDot >= 0 {
		modName = string(name[:lastDot])
	}

	//2. search in imported modules
	if modules != nil {
		var rType Type
		var rModule *Module
		var rIdent []ast.FullIdentifier

		for _, imp := range module.imports {
			if slices.Contains(imp.exposing, string(name)) {
				return findParsedType(nil, modules[imp.moduleIdentifier], typeName, args, loc)
			}
		}

		//3. search in all modules by qualified name
		if modName != "" {
			if submodule, ok := modules[ast.QualifiedIdentifier(modName)]; ok {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					return findParsedType(nil, submodule, typeName, args, loc)
				}
			}

			//4. search in all modules by short name
			modName = "." + modName
			for modId, submodule := range modules {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					if strings.HasSuffix(string(modId), modName) {
						foundType, foundModule, foundId, err := findParsedType(nil, submodule, typeName, args, loc)
						if err != nil {
							return nil, nil, nil, err
						}
						if foundId != nil {
							rType = foundType
							rModule = foundModule
							rIdent = append(rIdent, foundId...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rType, rModule, rIdent, nil
			}
		}

		//5. search by type name as module name
		if unicode.IsUpper([]rune(typeName)[0]) {
			modDotName := string("." + typeName)
			for modId, submodule := range modules {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					if strings.HasSuffix(string(modId), modDotName) || modId == typeName {
						foundType, foundModule, foundId, err := findParsedType(nil, submodule, typeName, args, loc)
						if err != nil {
							return nil, nil, nil, err
						}
						if foundId != nil {
							rType = foundType
							rModule = foundModule
							rIdent = append(rIdent, foundId...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rType, rModule, rIdent, nil
			}
		}

		if modName == "" {
			//6. search all modules
			for _, submodule := range modules {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					foundType, foundModule, foundId, err := findParsedType(nil, submodule, typeName, args, loc)
					if err != nil {
						return nil, nil, nil, err
					}
					if foundId != nil {
						rType = foundType
						rModule = foundModule
						rIdent = append(rIdent, foundId...)
					}
				}
			}
			if len(rIdent) != 0 {
				return rType, rModule, rIdent, nil
			}
		}
	}

	return nil, nil, nil, nil
}

// TODO: rewrite
func applyTypeArgs(t Type, params map[ast.Identifier]Type, loc ast.Location) (Type, error) {
	doMap := func(x Type) (Type, error) { return applyTypeArgs(x, params, loc) }
	var err error
	switch t.(type) {
	case *TFunc:
		{
			p := t.(*TFunc)
			fnParams, err := common.MapError(doMap, p.params)
			if err != nil {
				return nil, err
			}
			ret, err := applyTypeArgs(p.return_, params, loc)
			if err != nil {
				return nil, err
			}
			return NewTFunc(loc, fnParams, ret), nil
		}
	case *TRecord:
		{
			p := t.(*TRecord)
			fields := map[ast.Identifier]Type{}
			for name, f := range p.fields {
				fields[name], err = applyTypeArgs(f, params, loc)
				if err != nil {
					return nil, err
				}
			}
			return NewTRecord(loc, fields), nil
		}
	case *TTuple:
		{
			p := t.(*TTuple)
			items, err := common.MapError(doMap, p.items)
			if err != nil {
				return nil, err
			}
			return NewTTuple(loc, items), nil
		}
	case *TUnit:
		return t, nil
	case *TData:
		{
			p := t.(*TData)
			args, err := common.MapError(doMap, p.args)
			if err != nil {
				return nil, err
			}
			return NewTData(loc, p.name, args, p.options), nil
		}
	case *TNamed:
		{
			p := t.(*TNamed)
			args, err := common.MapError(doMap, p.args)
			if err != nil {
				return nil, err
			}
			return NewTNamed(loc, p.name, args), nil
		}
	case *TNative:
		{
			p := t.(*TNative)
			args, err := common.MapError(doMap, p.args)
			if err != nil {
				return nil, err
			}
			return NewTNative(loc, p.name, args), nil
		}
	case *TParameter:
		{
			e := t.(*TParameter)
			return params[e.name], nil
		}
	}
	return nil, common.NewCompilerError("impossible case")
}

func findParsedDefinition(
	modules map[ast.QualifiedIdentifier]*Module,
	module *Module,
	name ast.QualifiedIdentifier,
	normalizedModule *normalized.Module,
) (*Definition, *Module, []ast.FullIdentifier) {
	d, m, id := findParsedDefinitionImpl(modules, module, name)
	if len(id) == 1 {
		normalizedModule.AddDependencies(m.name, d.name)
	}
	return d, m, id
}

func findParsedDefinitionImpl(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, name ast.QualifiedIdentifier,
) (*Definition, *Module, []ast.FullIdentifier) {
	var defNameEq = func(x *Definition) bool {
		return ast.QualifiedIdentifier(x.name) == name
	}

	//1. search in current module
	if def, ok := common.Find(defNameEq, module.definitions); ok {
		return def, module, []ast.FullIdentifier{common.MakeFullIdentifier(module.name, def.name)}
	}

	lastDot := strings.LastIndex(string(name), ".")
	defName := name[lastDot+1:]
	modName := ""
	if lastDot >= 0 {
		modName = string(name[:lastDot])
	}

	//2. search in imported modules
	if modules != nil {
		for _, imp := range module.imports {
			if slices.Contains(imp.exposing, string(name)) {
				return findParsedDefinitionImpl(nil, modules[imp.moduleIdentifier], defName)
			}
		}

		var rDef *Definition
		var rModule *Module
		var rIdent []ast.FullIdentifier

		//3. search in all modules by qualified name
		if modName != "" {
			if submodule, ok := modules[ast.QualifiedIdentifier(modName)]; ok {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					return findParsedDefinitionImpl(nil, submodule, defName)
				}
			}

			//4. search in all modules by short name
			modName = "." + modName
			for modId, submodule := range modules {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					if strings.HasSuffix(string(modId), modName) {
						if d, m, i := findParsedDefinitionImpl(nil, submodule, defName); len(i) != 0 {
							rDef = d
							rModule = m
							rIdent = append(rIdent, i...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rDef, rModule, rIdent
			}
		}

		//5. search by definition name as module name
		if len(defName) > 0 && unicode.IsUpper([]rune(defName)[0]) {
			modDotName := string("." + defName)
			for modId, submodule := range modules {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					if strings.HasSuffix(string(modId), modDotName) || modId == defName {
						if d, m, i := findParsedDefinitionImpl(nil, submodule, defName); len(i) != 0 {
							rDef = d
							rModule = m
							rIdent = append(rIdent, i...)
						}
					}
				}
			}
			if len(rIdent) != 0 {
				return rDef, rModule, rIdent
			}
		}

		if modName == "" {
			//6. search all modules
			for _, submodule := range modules {
				if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
					if d, m, i := findParsedDefinitionImpl(nil, submodule, defName); len(i) != 0 {
						rDef = d
						rModule = m
						rIdent = append(rIdent, i...)
					}
				}
			}
			if len(rIdent) != 0 {
				return rDef, rModule, rIdent
			}
		}
	}

	return nil, nil, nil
}

func findParsedInfixFn(
	modules map[ast.QualifiedIdentifier]*Module, module *Module, name ast.InfixIdentifier,
) (*Infix, *Module, []ast.FullIdentifier) {
	//1. search in current module
	var infNameEq = func(x *Infix) bool { return x.name == name }
	if inf, ok := common.Find(infNameEq, module.infixFns); ok {
		id := common.MakeFullIdentifier(module.name, inf.alias)
		return inf, module, []ast.FullIdentifier{id}
	}

	//2. search in imported modules
	if modules != nil {
		for _, imp := range module.imports {
			if slices.Contains(imp.exposing, string(name)) {
				return findParsedInfixFn(nil, modules[imp.moduleIdentifier], name)
			}
		}

		//6. search all modules
		var rInfix *Infix
		var rModule *Module
		var rIdent []ast.FullIdentifier
		for _, submodule := range modules {
			if _, referenced := module.referencedPackages[submodule.packageName]; referenced {
				if foundInfix, foundModule, foundId := findParsedInfixFn(nil, submodule, name); foundId != nil {
					rInfix = foundInfix
					rModule = foundModule
					rIdent = append(rIdent, foundId...)
				}
			}
		}
		return rInfix, rModule, rIdent
	}
	return nil, nil, nil
}
