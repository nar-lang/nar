package linkers

//
//import (
//	"nar-compiler/internal/pkg/ast"
//	"nar-compiler/internal/pkg/locator"
//	"nar-compiler/pkg/logger"
//)
//
//type EmptyLinker struct {
//}
//
//func (l EmptyLinker) GetOutFileLocation(givenLocation string) string {
//	return givenLocation + ".binar"
//}
//
//func (l EmptyLinker) Link(
//	main ast.FullIdentifier, packages map[ast.PackageIdentifier]*locator.LoadedPackage,
//	out string, debug, upgrade bool, cacheDir string,
//	logger *logger.LogWriter,
//) error {
//	return nil
//}
//
//func (l EmptyLinker) Cleanup() {
//}
