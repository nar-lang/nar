package resolved

definedType definitionBase struct {
	name  string
	type_ Type
}

definedType definitionBaseWithGenerics struct {
	definitionBase
	genericParams GenericParams
}

func (d definitionBaseWithGenerics) getGenerics() GenericParams {
	return d.genericParams
}
