package model

type Module struct {
	Cu     *CompilationUnit
	Di     Directive
	Name   string
	isOpen bool
}

func (m *Module) GetAPrimaryQlClass() string {
	return "Module"
}

func (m *Module) GetACompilationUnit() *CompilationUnit {
	return m.Cu
}

func (m *Module) GetName() string {
	return m.Name
}

func (m *Module) ToString() string {
	return m.Name
}

func (m *Module) GetDirective() *Directive {
	return &m.Di
}

func (m *Module) IsOpen() bool {
	return m.isOpen
}

type Directive struct {
	Directive string
}

func (d *Directive) ToString() string {
	return d.Directive
}
