package resolved

import "strings"

const kTargetRuntimeVersion string = "v0.0.6"

type PackageFullName string

func (n PackageFullName) SafeName() string {
	s := string(n)
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func writeUseVar(sb *strings.Builder, name string) {
	if name != "_" && name != "" {
		sb.WriteString("runtime.UseVar(")
		sb.WriteString(name)
		sb.WriteString(")\n")
	}
}
