package migrator

import (
	"database/sql"
)

// TableType table type implements TableType interface
type TableType struct {
	CatalogValue string
	SchemaValue  string
	NameValue    string
	TypeValue    string
	EngineValue  sql.NullString
	CommentValue sql.NullString
}

// Catalog returns the catalog of the table.
func (ct TableType) Catalog() string {
	return ct.CatalogValue
}

// Schema returns the schema of the table.
func (ct TableType) Schema() string {
	return ct.SchemaValue
}

// Name returns the name of the table.
func (ct TableType) Name() string {
	return ct.NameValue
}

// Type returns the type of the table.
func (ct TableType) Type() string {
	return ct.TypeValue
}

// Engine returns the engine of current table.
func (ct TableType) Engine() (engine string, ok bool) {
	return ct.EngineValue.String, ct.EngineValue.Valid
}

// Comment returns the comment of current table.
func (ct TableType) Comment() (comment string, ok bool) {
	return ct.CommentValue.String, ct.CommentValue.Valid
}
