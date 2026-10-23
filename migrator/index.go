package migrator

import "database/sql"

// Index implements gorm.Index as a dialect-populated metadata record. PrimaryKey
// and Unique preserve the distinction between a known false value and unavailable
// information. It does not query the database or own an independent copy of ColumnList.
type Index struct {
	// TableName is the database table containing the index.
	TableName string
	// NameValue is the database index name.
	NameValue string
	// ColumnList holds indexed column names in the order supplied by the dialect. Columns
	// returns this slice directly.
	ColumnList []string
	// PrimaryKeyValue records whether this index represents the primary key; Valid reports
	// availability.
	PrimaryKeyValue sql.NullBool
	// UniqueValue records uniqueness of the whole index; Valid reports availability.
	UniqueValue sql.NullBool
	// OptionValue holds dialect-specific index attributes; an empty string can mean absent
	// or unavailable.
	OptionValue string
}

// Table returns the table name supplied by the dialect.
func (idx Index) Table() string {
	return idx.TableName
}

// Name returns the database index name supplied by the dialect.
func (idx Index) Name() string {
	return idx.NameValue
}

// Columns returns the stored column-name slice without copying it. Copy the slice
// before modifying it if the metadata must remain unchanged.
func (idx Index) Columns() []string {
	return idx.ColumnList
}

// PrimaryKey returns whether this is a primary-key index and whether that information
// is known.
func (idx Index) PrimaryKey() (isPrimaryKey bool, ok bool) {
	return idx.PrimaryKeyValue.Bool, idx.PrimaryKeyValue.Valid
}

// Unique returns whether the complete index is unique and whether that information is
// known. A composite unique index does not imply uniqueness of each column
// independently.
func (idx Index) Unique() (unique bool, ok bool) {
	return idx.UniqueValue.Bool, idx.UniqueValue.Valid
}

// Option returns the dialect-specific index attributes without parsing them.
func (idx Index) Option() string {
	return idx.OptionValue
}
