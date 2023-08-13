package resolved

import "strings"

const kTargetRuntimeVersion string = "v0.0.4"

type PackageFullName string

func (n PackageFullName) SafeName() string {
	s := string(n)
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
