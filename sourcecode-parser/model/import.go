package model

import (
	"database/sql"
	"fmt"
)

// ImportType represents a single-type import declaration in Java.
type ImportType struct {
	ImportedType      string // The fully qualified name of the imported type
	SourceDeclaration string // Location of the import statement
}

func (it *ImportType) Insert(db *sql.DB) error {
	query := `
	INSERT INTO import_decl (
		import_type,
		import_name,
		file_path
	) VALUES (?, ?, ?)
	`
	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(it.ImportedType, it.ImportedType, it.SourceDeclaration)
	if err != nil {
		return err
	}
	return nil
}

// NewImportType initializes a new ImportType instance.
func NewImportType(importedType, sourceDeclaration string) *ImportType {
	return &ImportType{
		ImportedType:      importedType,
		SourceDeclaration: sourceDeclaration,
	}
}

// ✅ Implementing AST-Based Predicates

// GetAPrimaryQlClass returns the primary CodeQL class name.
func (it *ImportType) GetAPrimaryQlClass() string {
	return "ImportType"
}

// GetImportedType retrieves the imported type.
func (it *ImportType) GetImportedType() string {
	return it.ImportedType
}

// ToString returns a textual representation of the import statement.
func (it *ImportType) ToString() string {
	return fmt.Sprintf("import %s;", it.ImportedType)
}
