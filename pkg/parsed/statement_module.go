package parsed

func NewModuleStatement(name string) StatementModule {
	return StatementModule{name: name}
}

type StatementModule struct {
	name string
}

func (m *StatementModule) Name() string {
	return m.name
}
