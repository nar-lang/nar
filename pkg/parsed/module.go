package parsed

func NewModule(name string, imports []Import, definitions []Definition) Module {
	module := Module{
		name:        name,
		imports:     imports,
		definitions: map[string]Definition{},
	}

	for _, d := range definitions {
		module.order = append(module.order, d.Name())
		module.definitions[d.Name()] = d
	}
	return module
}

definedType Module struct {
	name            string
	imports         []Import
	definitions     map[string]Definition
	order           []string
	unpackedImports map[string]DefinitionAddress
}

func (m Module) Name() string {
	return m.name
}

func (m Module) unpack(md *Metadata) (Module, error) {
	md.CurrentModule = NewModuleFullName(md.CurrentPackage, m.name)

	for i, name := range m.order {
		def := m.definitions[name]
		md.CurrentDefinition = name

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

func (m Module) precondition(md *Metadata) (Module, error) {
	md.CurrentModule = NewModuleFullName(md.CurrentPackage, m.name)

	for _, name := range m.order {
		def := m.definitions[name]
		md.CurrentDefinition = name

		err := def.precondition(md)
		if err != nil {
			return Module{}, err
		}
	}
	return m, nil
}

func (m Module) inferTypes(md *Metadata) (Module, error) {
	md.CurrentModule = NewModuleFullName(md.CurrentPackage, m.name)

	println("  " + m.name)

	for _, name := range m.order {
		def := m.definitions[name]
		md.CurrentDefinition = name

		println("    " + name)

		var err error
		_, err = def.inferType(md)
		if err != nil {
			return Module{}, err
		}
	}

	return m, nil
}
