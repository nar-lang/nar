package parsed

import (
	"oak-compiler/pkg/resolved"
)

func NewModule(statement StatementModule, imports []StatementImport, definitions []Definition) Module {
	module := Module{
		header:      statement,
		imports:     imports,
		definitions: map[string]Definition{},
	}

	for _, d := range definitions {
		module.order = append(module.order, d.Name())
		module.definitions[d.Name()] = d
	}
	return module
}

type Module struct {
	header          StatementModule
	imports         []StatementImport
	definitions     map[string]Definition
	order           []string
	unpackedImports map[string]DefinitionAddress
}

func (m Module) Unpack(md *Metadata) (Module, error) {
	for i, name := range m.order {
		def := m.definitions[name]
		if defs := def.unpackNestedDefinitions(); len(defs) > 0 {
			var extraOrder []string
			for _, x := range defs {
				extraOrder = append(extraOrder, x.Name())
				m.definitions[x.Name()] = x
			}
			m.order = append(m.order[0:i+1], append(extraOrder, m.order[i+1:]...)...)
		}
	}

	m.unpackedImports = map[string]DefinitionAddress{}
	var err error
	for i, imp := range m.imports {
		m.imports[i], err = imp.inject(m.unpackedImports, md)
		if err != nil {
			return m, err
		}
	}

	return m, nil
}

func (m Module) Resolve(md *Metadata) (resolved.Module, error) {
	md.CurrentModule = m

	var err error
	for n, d := range m.definitions {
		m.definitions[n], err = d.precondition(md)
		if err != nil {
			return resolved.Module{}, err
		}
	}

	resolvedDefinitions := map[string]resolved.Definition{}
	var resolvedOrder []string

	for _, name := range m.order {
		rd, ok, err := m.definitions[name].resolve(md)
		if err != nil {
			return resolved.Module{}, err
		}
		if ok {
			resolvedDefinitions[name] = rd
			resolvedOrder = append(resolvedOrder, name)
		}
	}

	imports := map[PackageFullName]struct{}{}
	for _, imp := range m.imports {
		if imp.packageName != md.CurrentPackage.FullName() {
			imports[imp.packageName] = struct{}{}
		}
	}
	var resolvedImports []resolved.PackageFullName
	for n := range imports {
		resolvedImports = append(resolvedImports, resolved.PackageFullName(n))
	}

	return resolved.NewModule(
		m.Name(), md.CurrentPackage.Info.Name, resolvedDefinitions, resolvedOrder, resolvedImports,
	), nil
}

func (m Module) Name() string {
	return m.header.Name()
}
