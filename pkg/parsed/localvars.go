package parsed

definedType LocalVars struct {
	parent *LocalVars
	vars   map[string]Type
}

func NewLocalVars(parent *LocalVars) *LocalVars {
	return &LocalVars{
		parent: parent,
		vars:   map[string]Type{},
	}
}

func (lv *LocalVars) Lookup(name string) (Type, bool) {
	if t, ok := lv.vars[name]; ok {
		return t, true
	}
	if lv.parent != nil {
		return lv.parent.Lookup(name)
	}
	return nil, false
}

func (lv *LocalVars) Populate(name string, type_ Type) {
	lv.vars[name] = type_
}
