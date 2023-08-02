package parsed

import "encoding/json"

func NewModuleStatement(name string) StatementModule {
	return StatementModule{name: name}
}

type StatementModule struct {
	name string
}

func (m *StatementModule) Name() string {
	return m.name
}

func (m *StatementModule) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name string
	}{Name: m.name})
}
