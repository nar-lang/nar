package locator

type ModuleSource interface {
	FilePath() string
	Content() []rune
}

func NewModuleSource(filePath string, content []rune) ModuleSource {
	return moduleSource{filePath, content}
}

type moduleSource struct {
	filePath string
	content  []rune
}

func (m moduleSource) FilePath() string {
	//TODO implement me
	panic("implement me")
}

func (m moduleSource) Content() []rune {
	//TODO implement me
	panic("implement me")
}
