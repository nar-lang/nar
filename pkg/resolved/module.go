package resolved

import (
	"strings"
)

func NewModule(
	name string, packageName string, definitions map[string]Definition, order []string, imports []PackageFullName,
) Module {
	return Module{name: name, packageName: packageName, definitions: definitions, order: order, imports: imports}
}

definedType Module struct {
	name        string
	packageName string
	imports     []PackageFullName
	definitions map[string]Definition
	order       []string
}

func (m Module) write(sb *strings.Builder) {
	sb.WriteString("package ")
	sb.WriteString(m.packageName)
	sb.WriteString("\n\n")

	for _, imp := range m.imports {
		sb.WriteString("import ")
		sb.WriteString(imp.SafeName())
		sb.WriteString(" \"")
		sb.WriteString(string(imp))
		sb.WriteString("\"\n")
	}
	sb.WriteString("\n")

	sb.WriteString("import \"github.com/oaklang/runtime\"\nvar _ = runtime.Use()\n\n")

	for _, name := range m.order {
		def := m.definitions[name]
		def.write(sb)
	}
	return
}
