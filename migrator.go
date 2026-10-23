package gorm

import (
	"reflect"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Migrator applies pending scopes and returns the dialector's schema manager bound to a
// new session. Database capabilities and the dialector determine the SQL and supported
// operations. Calling this method alone does not migrate tables.
func (db *DB) Migrator() Migrator {
	tx := db.getInstance()

	// apply scopes to migrator
	for len(tx.Statement.scopes) > 0 {
		tx = tx.executeScopes()
	}

	return tx.Dialector.Migrator(tx.Session(&Session{}))
}

// AutoMigrate reconciles the supplied model schemas through the dialector's migrator.
// It creates missing tables, columns, indexes, and permitted constraints, and may alter
// column definitions. It does not remove unused columns. Changes are not guaranteed
// atomic or non-blocking across databases; use reviewed versioned migrations when
// rollout/rollback control is required.
//
// See https://gorm.io/docs/migration.html.
func (db *DB) AutoMigrate(dst ...interface{}) error {
	return db.Migrator().AutoMigrate(dst...)
}

// ViewOption configures CreateView. Query is required, Replace requests CREATE OR
// REPLACE where supported, and CheckOption is trusted dialect-specific SQL appended to
// the definition.
type ViewOption struct {
	// Replace requests CREATE OR REPLACE VIEW when true; false requests CREATE VIEW.
	// Support depends on the database (SQLite does not support OR REPLACE for views).
	Replace bool
	// CheckOption is a trusted SQL suffix, such as WITH LOCAL CHECK OPTION. It is appended
	// verbatim and must not contain unvalidated user input.
	CheckOption string
	// Query is the required subquery defining the view. A nil query produces
	// ErrSubQueryRequired.
	Query *DB
}

// ColumnType exposes database column metadata for migration decisions. Methods
// returning an ok flag can report unknown information; ok=false is not the same as a
// known false or zero. Metadata detail varies by driver and database.
type ColumnType interface {
	// Name returns the database column name.
	Name() string
	// DatabaseTypeName returns the base database type name, such as VARCHAR, in
	// driver-specific spelling.
	DatabaseTypeName() string // varchar
	// ColumnType returns the full type declaration, such as varchar(64), and whether it is
	// available.
	ColumnType() (columnType string, ok bool) // varchar(64)
	// PrimaryKey reports whether the column is a primary key and whether the answer is
	// known.
	PrimaryKey() (isPrimaryKey bool, ok bool)
	// AutoIncrement reports generated increment behavior and whether the metadata is
	// available.
	AutoIncrement() (isAutoIncrement bool, ok bool)
	// Length returns a declared type length and its availability; units depend on the
	// database type.
	Length() (length int64, ok bool)
	// DecimalSize returns precision and scale with an availability flag for numeric
	// metadata.
	DecimalSize() (precision int64, scale int64, ok bool)
	// Nullable reports nullability and whether it is known.
	Nullable() (nullable bool, ok bool)
	// Unique reports single-column uniqueness and whether it is known; it does not fully
	// describe composite indexes.
	Unique() (unique bool, ok bool)
	// ScanType returns the Go type recommended by the driver for scanning, which can be nil
	// when unavailable.
	ScanType() reflect.Type
	// Comment returns the column comment and whether it is available.
	Comment() (value string, ok bool)
	// DefaultValue returns the database default expression and whether one is reported. The
	// expression is metadata, not a Go literal.
	DefaultValue() (value string, ok bool)
}

// Index describes a database index discovered by a migrator. Unique and PrimaryKey have
// availability flags because drivers may not report all metadata. Composite-column
// order can matter.
type Index interface {
	// Table returns the owning database table name.
	Table() string
	// Name returns the database index name.
	Name() string
	// Columns returns indexed column names in the order reported by the migrator.
	Columns() []string
	// PrimaryKey reports whether the index represents a primary key and whether that is
	// known.
	PrimaryKey() (isPrimaryKey bool, ok bool)
	// Unique reports whether the index enforces uniqueness and whether that is known.
	Unique() (unique bool, ok bool)
	// Option returns database-specific index options when provided by the migrator.
	Option() string
}

// TableType reports database table/view metadata. Schema and Type are
// database-specific; an unavailable comment is distinguished by its boolean flag.
type TableType interface {
	// Schema returns the table's database schema or namespace as reported by the migrator.
	Schema() string
	// Name returns the database object name.
	Name() string
	// Type returns the database-specific object classification, such as a table or view.
	Type() string
	// Comment returns the table comment and whether it is available.
	Comment() (comment string, ok bool)
}

// Migrator provides database-specific schema inspection and DDL execution. Pass model
// values to use parsed names/tags, or table names where supported. Method availability,
// locking, transaction behavior, and type compatibility depend on the dialector and
// database version. Destructive calls such as DropTable/DropColumn remove stored data;
// AutoMigrate does not infer those removals.
type Migrator interface {
	// AutoMigrate reconciles supplied models with the database, creating missing objects
	// and performing supported column alterations without dropping unused columns. See
	// DB.AutoMigrate for deployment boundaries.
	AutoMigrate(dst ...interface{}) error

	// CurrentDatabase returns the active database name as reported by the dialect. The
	// interface has no error return, so it cannot distinguish every lookup failure from an
	// empty name.
	CurrentDatabase() string
	// FullDataTypeOf builds the database type expression for a parsed field, including
	// supported nullability/default modifiers. It does not execute DDL.
	FullDataTypeOf(*schema.Field) clause.Expr
	// GetTypeAliases returns database type names treated as aliases of databaseTypeName
	// when comparing schema definitions.
	GetTypeAliases(databaseTypeName string) []string

	// CreateTable creates tables from parsed models, including supported indexes and
	// constraints. It returns the first DDL error and does not promise atomicity across all
	// tables.
	CreateTable(dst ...interface{}) error
	// DropTable drops the specified model tables or names. This is destructive and may
	// affect dependent constraints; ordering and transaction support are dialect-specific.
	DropTable(dst ...interface{}) error
	// HasTable reports whether the model table or supplied name exists. Because it has no
	// error return, false may also reflect a failed metadata lookup.
	HasTable(dst interface{}) bool
	// RenameTable renames a table identified by model or name. Database support and
	// handling of dependent objects are dialect-specific.
	RenameTable(oldName, newName interface{}) error
	// GetTables returns visible table names or a metadata-query error; namespace coverage
	// depends on the dialect.
	GetTables() (tableList []string, err error)
	// TableType returns metadata for a table or an error when unavailable. Not all
	// migrators implement this inspection.
	TableType(dst interface{}) (TableType, error)

	// AddColumn adds the named model field as a database column using its schema tags and
	// dialect type mapping.
	AddColumn(dst interface{}, field string) error
	// DropColumn removes a column and its stored values. field may be resolved through
	// model metadata; supported SQL/rebuild behavior depends on the database.
	DropColumn(dst interface{}, field string) error
	// AlterColumn changes the named column to the model field's definition where supported.
	// Type conversion and locking are database-specific.
	AlterColumn(dst interface{}, field string) error
	// MigrateColumn compares a parsed model field with current column metadata and performs
	// required supported alterations. It is the column-level step used by AutoMigrate.
	MigrateColumn(dst interface{}, field *schema.Field, columnType ColumnType) error
	// MigrateColumnUnique reconciles a field's single-column UNIQUE constraint during
	// column migration. It is distinct from composite unique-index handling.
	MigrateColumnUnique(dst interface{}, field *schema.Field, columnType ColumnType) error
	// HasColumn reports whether the named model field or column exists. A false result can
	// also represent unavailable metadata because this method has no error return.
	HasColumn(dst interface{}, field string) bool
	// RenameColumn renames an existing column, resolving model field names where supported.
	// It does not migrate application SQL references automatically.
	RenameColumn(dst interface{}, oldName, field string) error
	// ColumnTypes returns column metadata or an inspection error. Check each metadata
	// method's availability flag before comparing it with a model.
	ColumnTypes(dst interface{}) ([]ColumnType, error)

	// CreateView creates a named view from ViewOption.Query. A nil query returns
	// ErrSubQueryRequired; Replace and CheckOption require dialect support.
	CreateView(name string, option ViewOption) error
	// DropView drops a named view using the dialector's implementation. Existing consumers
	// of that view may fail afterward.
	DropView(name string) error

	// CreateConstraint creates a named model constraint, resolving relationship names for
	// foreign keys where supported.
	CreateConstraint(dst interface{}, name string) error
	// DropConstraint drops a named constraint, removing its database enforcement.
	// Relationship names may be resolved through model metadata.
	DropConstraint(dst interface{}, name string) error
	// HasConstraint reports whether a constraint exists. A false result is not a substitute
	// for handling metadata-query errors when the dialect cannot expose them through this
	// boolean interface.
	HasConstraint(dst interface{}, name string) bool

	// CreateIndex creates an index defined by model tags, identified by index name or a
	// supported field lookup. Expression, unique, and other options depend on the dialect.
	CreateIndex(dst interface{}, name string) error
	// DropIndex drops the named index or one resolved from model metadata. Dropping a
	// unique index can remove uniqueness enforcement.
	DropIndex(dst interface{}, name string) error
	// HasIndex reports whether an index exists; false can also reflect a failed metadata
	// lookup because no error is returned.
	HasIndex(dst interface{}, name string) bool
	// RenameIndex renames an index where supported; some dialects recreate the index
	// instead.
	RenameIndex(dst interface{}, oldName, newName string) error
	// GetIndexes returns index metadata or an inspection error. Availability and
	// completeness depend on the dialector.
	GetIndexes(dst interface{}) ([]Index, error)
}
