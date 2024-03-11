package parsed

import (
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/common"
	"strings"
	"unicode"
)

type Module struct {
	name        ast.QualifiedIdentifier
	location    ast.Location
	imports     []Import
	aliases     []Alias
	infixFns    []Infix
	definitions []Definition
	dataTypes   []DataType

	packageName        ast.PackageIdentifier
	referencedPackages map[ast.PackageIdentifier]struct{}
}

func NewModule(
	name ast.QualifiedIdentifier, loc ast.Location,
	imports []Import, aliases []Alias, infixFns []Infix, definitions []Definition, dataTypes []DataType,
) *Module {
	return &Module{
		name:               name,
		location:           loc,
		imports:            imports,
		aliases:            aliases,
		infixFns:           infixFns,
		definitions:        definitions,
		dataTypes:          dataTypes,
		referencedPackages: map[ast.PackageIdentifier]struct{}{},
	}
}

func (module *Module) Name() ast.QualifiedIdentifier {
	return module.name
}

func (module *Module) Location() ast.Location {
	return module.location
}

func (module *Module) PackageName() ast.PackageIdentifier {
	return module.packageName
}

func (module *Module) SetPackageName(packageName ast.PackageIdentifier) {
	module.packageName = packageName
}

func (module *Module) SetReferencedPackages(referencedPackages map[ast.PackageIdentifier]struct{}) {
	module.referencedPackages = referencedPackages
}

func (module *Module) Generate(modules map[ast.QualifiedIdentifier]*Module) (errors []error) {
	for _, dt := range module.dataTypes {
		alias, defs := dt.flatten(module.name)
		module.aliases = append(module.aliases, alias)
		module.definitions = append(module.definitions, defs...)
	}

	return module.unwrapImports(modules)
}

func (module *Module) Normalize(
	modules map[ast.QualifiedIdentifier]*Module,
	normalizedModules map[ast.QualifiedIdentifier]*normalized.Module,
) (errors []error) {
	if _, ok := normalizedModules[module.name]; ok {
		return
	}

	o := normalized.NewModule(module.location, module.name, nil)

	for _, def := range module.definitions {
		nDef, params, err := def.normalize(modules, module, o)
		if err != nil {
			errors = append(errors, err...)
		}
		nDef.FlattenLambdas(params, o)

		o.AddDefinition(nDef)
	}

	normalizedModules[module.name] = o

	for _, modName := range o.Dependencies() {
		depModule, ok := modules[modName]
		if !ok {
			errors = append(errors, common.NewErrorOf(depModule, "module `%s` not found", modName))
			continue
		}

		if err := depModule.Normalize(modules, normalizedModules); err != nil {
			errors = append(errors, err...)
		}
	}

	return
}

func (module *Module) unwrapImports(modules map[ast.QualifiedIdentifier]*Module) (errors []error) {
	for _, imp := range module.imports {
		err := imp.unwrap(modules)
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}
	return
}

func (module *Module) Iterate(f func(statement Statement)) {
	for _, alias := range module.aliases {
		if alias != nil {
			alias.Iterate(f)
		}
	}
	for _, def := range module.definitions {
		def.Iterate(f)
	}
}

func (module *Module) isReferenced(submodule *Module) bool {
	if submodule.packageName == module.packageName {
		return true
	}
	_, referenced := module.referencedPackages[submodule.packageName]
	return referenced
}

func (module *Module) findInfixFn(
	modules map[ast.QualifiedIdentifier]*Module, name ast.InfixIdentifier,
) (Infix, *Module, []ast.FullIdentifier) {
	//1. search in current module
	var infNameEq = func(x Infix) bool { return x.name() == name }
	if inf, ok := common.Find(infNameEq, module.infixFns); ok {
		return inf, module, []ast.FullIdentifier{common.MakeFullIdentifier(module.name, inf.alias())}
	}

	//2. search in imported modules
	if modules != nil {
		for _, imp := range module.imports {
			if imp.exposes(string(name)) {
				return modules[imp.Module()].findInfixFn(nil, name)
			}
		}

		//6. search all modules
		var rInfix Infix
		var rModule *Module
		var rIdent []ast.FullIdentifier
		for _, submodule := range modules {
			if module.isReferenced(submodule) {
				if foundInfix, foundModule, foundId := submodule.findInfixFn(nil, name); foundId != nil {
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

func (module *Module) findType(
	modules map[ast.QualifiedIdentifier]*Module,
	name ast.QualifiedIdentifier,
	args []Type,
	loc ast.Location,
) (Type, *Module, []ast.FullIdentifier, error) {
	var aliasNameEq = func(x Alias) bool { return ast.QualifiedIdentifier(x.Name()) == name }

	// 1. check current module
	if typeAlias, ok := common.Find(aliasNameEq, module.aliases); ok {
		type_, id, err := typeAlias.inferType(module.name, args)
		if err != nil {
			return nil, nil, nil, err
		}
		return type_, module, []ast.FullIdentifier{id}, nil
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

			if imp.exposes(string(name)) {
				return modules[imp.Module()].findType(nil, typeName, args, loc)
			}
		}

		//3. search in all modules by qualified Name
		if modName != "" {
			if submodule, ok := modules[ast.QualifiedIdentifier(modName)]; ok {
				if module.isReferenced(submodule) {
					return submodule.findType(nil, typeName, args, loc)
				}
			}

			//4. search in all modules by short Name
			modName = "." + modName
			for modId, submodule := range modules {
				if module.isReferenced(submodule) {
					if strings.HasSuffix(string(modId), modName) {
						foundType, foundModule, foundId, err := submodule.findType(nil, typeName, args, loc)
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

		//5. search by type Name as module Name
		if unicode.IsUpper([]rune(typeName)[0]) {
			modDotName := string("." + typeName)
			for modId, submodule := range modules {
				if module.isReferenced(submodule) {
					if strings.HasSuffix(string(modId), modDotName) || modId == typeName {
						foundType, foundModule, foundId, err := submodule.findType(nil, typeName, args, loc)
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
				if module.isReferenced(submodule) {
					foundType, foundModule, foundId, err := submodule.findType(nil, typeName, args, loc)
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

func (module *Module) findDefinitionAndAddDependency(
	modules map[ast.QualifiedIdentifier]*Module,
	name ast.QualifiedIdentifier,
	normalizedModule *normalized.Module,
) (Definition, *Module, []ast.FullIdentifier) {
	d, m, id := module.findDefinition(modules, name)
	if len(id) == 1 {
		normalizedModule.AddDependencies(m.name, d.Name())
	}
	return d, m, id
}

func (module *Module) findDefinition(
	modules map[ast.QualifiedIdentifier]*Module, name ast.QualifiedIdentifier,
) (Definition, *Module, []ast.FullIdentifier) {
	var defNameEq = func(x Definition) bool {
		return ast.QualifiedIdentifier(x.Name()) == name
	}

	//1. search in current module
	if def, ok := common.Find(defNameEq, module.definitions); ok {
		return def, module, []ast.FullIdentifier{common.MakeFullIdentifier(module.name, def.Name())}
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
			if imp.exposes(string(name)) {
				return modules[imp.Module()].findDefinition(nil, defName)
			}
		}

		var rDef Definition
		var rModule *Module
		var rIdent []ast.FullIdentifier

		//3. search in all modules by qualified Name
		if modName != "" {
			if submodule, ok := modules[ast.QualifiedIdentifier(modName)]; ok {
				if module.isReferenced(submodule) {
					return submodule.findDefinition(nil, defName)
				}
			}

			//4. search in all modules by short Name
			modName = "." + modName
			for modId, submodule := range modules {
				if module.isReferenced(submodule) {
					if strings.HasSuffix(string(modId), modName) {
						if d, m, i := submodule.findDefinition(nil, defName); len(i) != 0 {
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

		//5. search by definition Name as module Name
		if len(defName) > 0 && unicode.IsUpper([]rune(defName)[0]) {
			modDotName := string("." + defName)
			for modId, submodule := range modules {
				if module.isReferenced(submodule) {
					if strings.HasSuffix(string(modId), modDotName) || modId == defName {
						if d, m, i := submodule.findDefinition(nil, defName); len(i) != 0 {
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
				if module.isReferenced(submodule) {
					if d, m, i := submodule.findDefinition(nil, defName); len(i) != 0 {
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

func (module *Module) InfixFns() []Infix {
	return module.infixFns
}

func (module *Module) Aliases() []Alias {
	return module.aliases
}

func (module *Module) DataTypes() []DataType {
	return module.dataTypes
}

func (module *Module) Definitions() []Definition {
	return module.definitions
}

func (module *Module) Imports() []Import {
	return module.imports
}
