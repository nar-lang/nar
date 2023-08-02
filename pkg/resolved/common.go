package resolved

import "strings"

type PackageFullName string

func (n PackageFullName) SafeName() string {
	s := string(n)
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
