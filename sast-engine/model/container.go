package model

type Container struct {
	Top
}

func (c *Container) ToString() string {
	return ""
}

type File struct {
	Container
	File string
}

func (f *File) IsSourceFile() bool {
	// check if file extension is .java
	if f.File[len(f.File)-5:] == ".java" || f.File[len(f.File)-4:] == ".kt" {
		return true
	}
	return false
}

func (f *File) IsJavaSourceFile() bool {
	return f.File[len(f.File)-5:] == ".java"
}

func (f *File) IsKotlinSourceFile() bool {
	return f.File[len(f.File)-5:] == ".kt"
}

func (f *File) GetAPrimaryQlClass() string {
	return "File"
}

type CompilationUnit struct {
	File
	module    Module
	CuPackage Package
	Name      string
}

func (c *CompilationUnit) GetAPrimaryQlClass() string {
	return "CompilationUnit"
}

func (c *CompilationUnit) GetModule() Module {
	return c.module
}

func (c *CompilationUnit) GetName() string {
	return c.Name
}

func (c *CompilationUnit) GetPackage() Package {
	return c.CuPackage
}

func (c *CompilationUnit) HasName(name string) bool {
	return c.Name == name
}

func (c *CompilationUnit) ToString() string {
	return c.Name
}

type JarFile struct {
	File
	JarFile                 string
	ImplementationVersion   string
	ManifestEntryAttributes map[string]map[string]string
	ManifestMainAttributes  map[string]string
	SpecificationVersion    string
}

func (j *JarFile) GetAPrimaryQlClass() string {
	return "JarFile"
}

func (j *JarFile) GetJarFile() string {
	return j.JarFile
}

func (j *JarFile) GetImplementationVersion() string {
	return j.ImplementationVersion
}

func (j *JarFile) GetManifestEntryAttributes(entry, key string) (string, bool) {
	if attributes, exists := j.ManifestEntryAttributes[entry]; exists {
		if value, keyExists := attributes[key]; keyExists {
			return value, true
		}
	}
	return "", false
}

func (j *JarFile) GetManifestMainAttributes(key string) (string, bool) {
	if value, exists := j.ManifestMainAttributes[key]; exists {
		return value, true
	}
	return "", false
}

func (j *JarFile) GetSpecificationVersion() string {
	return j.SpecificationVersion
}

type Package struct {
	Package string
}

func (p *Package) GetAPrimaryQlClass() string {
	return "Package"
}

func (p *Package) GetURL() string {
	return p.Package
}
