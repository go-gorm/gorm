package migrator

import (
	"database/sql"
)

// TableType implements gorm.TableType as a dialect-populated table metadata record.
// Schema, name, and type are returned as supplied by the database; only the comment
// has a separate availability flag.
type TableType struct {
	// SchemaValue identifies the schema containing the table; interpretation is
	// dialect-specific.
	SchemaValue string
	// NameValue is the database table name.
	NameValue string
	// TypeValue is the database-reported table category, such as BASE TABLE or VIEW; values
	// are not normalized.
	TypeValue string
	// CommentValue holds the table comment; Valid distinguishes an empty comment from
	// unavailable information.
	CommentValue sql.NullString
}

// Schema returns the database schema name recorded by the dialect.
func (ct TableType) Schema() string {
	return ct.SchemaValue
}

// Name returns the database table name recorded by the dialect.
func (ct TableType) Name() string {
	return ct.NameValue
}

// Type returns the database-specific table category without normalization.
func (ct TableType) Type() string {
	return ct.TypeValue
}

// Comment returns the stored table comment and whether it is known. A known comment may
// be empty.
func (ct TableType) Comment() (comment string, ok bool) {
	return ct.CommentValue.String, ct.CommentValue.Valid
}
