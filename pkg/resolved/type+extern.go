package resolved

import "strings"

func NewExternType(name string, args GenericArgs) typeExtern {
	return typeExtern{typeBase{
		refName:     name,
		genericArgs: args,
	}}
}

definedType typeExtern struct {
	typeBase
}

func (t typeExtern) write(sb *strings.Builder) {
	if !t.writeNamed(sb) {
		panic("not supported")
	}
	return
}
