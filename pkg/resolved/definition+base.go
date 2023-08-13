package resolved

type definitionBase struct {
	name  string
	type_ Type
}

type definitionBaseWithGenerics struct {
	definitionBase
	genericParams GenericParams
}

func (d definitionBaseWithGenerics) getGenerics() GenericParams {
	return d.genericParams
}
