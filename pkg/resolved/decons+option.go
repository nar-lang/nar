package resolved

import "strings"

func NewOptionDecons(option string, valueType Type, arg Decons, alias string) Decons {
	return deconsOption{option: option, valueType: valueType, arg: arg, alias: alias}
}

type deconsOption struct {
	option    string
	valueType Type
	arg       Decons
	alias     string
}

func (d deconsOption) writeComparison(sb *strings.Builder, name string) {
	sb.WriteString("(\"")
	sb.WriteString(d.option)
	sb.WriteString("\"==")
	sb.WriteString(name)
	sb.WriteString(".Option)")
	if d.arg != nil {
		sb.WriteString("&&")
		tsb := &strings.Builder{}
		d.valueType.write(tsb)
		d.arg.writeComparison(sb, name+".Value.("+tsb.String()+")")
	}
}

func (d deconsOption) writeHeader(sb *strings.Builder, name string) {
	if d.alias != "" {
		sb.WriteString("var ")
		sb.WriteString(d.alias)
		sb.WriteString(" = ")
		sb.WriteString(name)
		sb.WriteString("\n")
		writeUseVar(sb, d.alias)
	}

	tsb := &strings.Builder{}
	tsb.WriteString(name)
	tsb.WriteString(".Value.(")
	d.valueType.write(tsb)
	tsb.WriteString(")")
	d.arg.writeHeader(sb, tsb.String())
}
