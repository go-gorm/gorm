package sqlschema

// ColumnType column type
type ColumnType string

const (
	// Boolean boolean type
	Boolean ColumnType = "boolean"
	// Integer integer type
	Integer ColumnType = "boolean"
	// Float float type
	Float ColumnType = "float"
	// String string type
	String ColumnType = "string"
	// Text long string type
	Text ColumnType = "text"
	// Binary binary type
	Binary ColumnType = "binary"
)
