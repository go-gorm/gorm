package migrator

import (
	"database/sql"
	"reflect"
)

// ColumnType implements gorm.ColumnType using metadata supplied by a dialect and,
// optionally, database/sql. Valid explicit values take precedence over SQLColumnType.
// A false availability flag means the property is unknown, not that its value is false
// or zero. The zero value contains no known metadata.
type ColumnType struct {
	// SQLColumnType supplies fallback name, database type, length, precision/scale,
	// nullability, and scan type. It may be nil; properties without a database/sql
	// equivalent use only explicit values.
	SQLColumnType *sql.ColumnType
	// NameValue overrides the column name or alias when Valid is true.
	NameValue sql.NullString
	// DataTypeValue overrides the database type name, usually without length or precision,
	// when Valid is true.
	DataTypeValue sql.NullString
	// ColumnTypeValue holds the dialect-specific full column type, such as varchar(64);
	// Valid reports availability.
	ColumnTypeValue sql.NullString
	// PrimaryKeyValue records primary-key membership; Valid distinguishes false from
	// unknown.
	PrimaryKeyValue sql.NullBool
	// UniqueValue records a column-level unique constraint when the dialect can determine
	// it. Unique indexes are separate metadata and need not set this value.
	UniqueValue sql.NullBool
	// AutoIncrementValue records automatic increment behavior; Valid distinguishes false
	// from unknown.
	AutoIncrementValue sql.NullBool
	// LengthValue overrides variable-length type capacity when Valid is true; units depend
	// on the database type.
	LengthValue sql.NullInt64
	// DecimalSizeValue holds decimal precision. When Valid is true, DecimalSize also
	// returns ScaleValue.Int64 without checking ScaleValue.Valid.
	DecimalSizeValue sql.NullInt64
	// ScaleValue holds decimal scale paired with DecimalSizeValue. Set both values
	// consistently; DecimalSize uses the precision validity flag.
	ScaleValue sql.NullInt64
	// NullableValue overrides whether NULL is allowed when Valid is true.
	NullableValue sql.NullBool
	// ScanTypeValue overrides the suggested Go scan type when non-nil.
	ScanTypeValue reflect.Type
	// CommentValue holds the column comment; Valid permits a known empty comment.
	CommentValue sql.NullString
	// DefaultValueValue holds the dialect-reported default expression or value. Valid
	// reports availability; the text is not a portable SQL expression or parsed Go value.
	DefaultValueValue sql.NullString
}

// Name returns the explicit name when valid, then the database/sql name, or an empty
// string if neither is available.
func (ct ColumnType) Name() string {
	if ct.NameValue.Valid {
		return ct.NameValue.String
	}
	if ct.SQLColumnType != nil {
		return ct.SQLColumnType.Name()
	}
	return ""
}

// DatabaseTypeName returns the explicit database type name when valid, then the
// database/sql type name, or an empty string when unavailable. Names and casing are
// driver-specific; length and precision normally belong in ColumnType.
func (ct ColumnType) DatabaseTypeName() string {
	if ct.DataTypeValue.Valid {
		return ct.DataTypeValue.String
	}
	if ct.SQLColumnType != nil {
		return ct.SQLColumnType.DatabaseTypeName()
	}
	return ""
}

// ColumnType returns the explicit full database type, such as varchar(64), and whether
// it is known. It has no database/sql fallback.
func (ct ColumnType) ColumnType() (columnType string, ok bool) {
	return ct.ColumnTypeValue.String, ct.ColumnTypeValue.Valid
}

// PrimaryKey returns whether the column belongs to the primary key and whether that
// information is known. It has no database/sql fallback.
func (ct ColumnType) PrimaryKey() (isPrimaryKey bool, ok bool) {
	return ct.PrimaryKeyValue.Bool, ct.PrimaryKeyValue.Valid
}

// AutoIncrement returns whether the column auto-increments and whether that information
// is known. It has no database/sql fallback.
func (ct ColumnType) AutoIncrement() (isAutoIncrement bool, ok bool) {
	return ct.AutoIncrementValue.Bool, ct.AutoIncrementValue.Valid
}

// Length returns the explicit capacity when valid, otherwise the database/sql length.
// It returns (0, false) without either source; interpret units according to the
// database type.
func (ct ColumnType) Length() (length int64, ok bool) {
	if ct.LengthValue.Valid {
		return ct.LengthValue.Int64, true
	}
	if ct.SQLColumnType != nil {
		return ct.SQLColumnType.Length()
	}
	return 0, false
}

// DecimalSize returns precision and scale from the explicit values when
// DecimalSizeValue.Valid is true, otherwise from database/sql. ScaleValue.Valid is not
// checked in the explicit path; unavailable metadata returns (0, 0, false).
func (ct ColumnType) DecimalSize() (precision int64, scale int64, ok bool) {
	if ct.DecimalSizeValue.Valid {
		return ct.DecimalSizeValue.Int64, ct.ScaleValue.Int64, true
	}
	if ct.SQLColumnType != nil {
		return ct.SQLColumnType.DecimalSize()
	}
	return 0, 0, false
}

// Nullable reports whether NULL is permitted, preferring the explicit value over
// database/sql. An unavailable result is (false, false), which must not be interpreted
// as NOT NULL.
func (ct ColumnType) Nullable() (nullable bool, ok bool) {
	if ct.NullableValue.Valid {
		return ct.NullableValue.Bool, true
	}
	if ct.SQLColumnType != nil {
		return ct.SQLColumnType.Nullable()
	}
	return false, false
}

// Unique reports the explicit column-level uniqueness information and its availability.
// This does not enumerate unique indexes or establish that a column in a composite
// unique index is individually unique.
func (ct ColumnType) Unique() (unique bool, ok bool) {
	return ct.UniqueValue.Bool, ct.UniqueValue.Valid
}

// ScanType returns the explicit Go type when non-nil, otherwise the database/sql scan
// type, or nil when neither is available. It is a scanning suggestion, not a
// requirement that the model field use that exact type.
func (ct ColumnType) ScanType() reflect.Type {
	if ct.ScanTypeValue != nil {
		return ct.ScanTypeValue
	}
	if ct.SQLColumnType != nil {
		return ct.SQLColumnType.ScanType()
	}
	return nil
}

// Comment returns the explicit column comment and whether it is known; a known comment
// may be empty.
func (ct ColumnType) Comment() (value string, ok bool) {
	return ct.CommentValue.String, ct.CommentValue.Valid
}

// DefaultValue returns the explicit dialect-reported default and whether it is known.
// The string may be a SQL expression and is not evaluated or normalized here.
func (ct ColumnType) DefaultValue() (value string, ok bool) {
	return ct.DefaultValueValue.String, ct.DefaultValueValue.Valid
}
