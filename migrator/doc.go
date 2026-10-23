// Package migrator provides the shared schema-migration implementation and
// metadata adapters used by GORM's database dialects. Applications normally call
// db.AutoMigrate or db.Migrator(); constructing this base Migrator directly loses
// dialect-specific SQL, catalog queries, and compatibility workarounds.
//
// # Choosing a migration operation
//
// AutoMigrate reconciles model metadata with the live schema. It creates absent
// tables, columns, indexes, and constraints, and can alter existing columns. It
// does not delete unused columns, rename columns after a Go field is renamed, or
// replace an existing index merely because its definition changed. It is not a
// versioned migration planner and does not guarantee that alterations preserve
// data or avoid locking. For controlled deployments, review and order explicit
// operations using db.Migrator() or a versioned migration tool.
//
// Typical calls use a model so the migrator can resolve fields, tags, naming
// conventions, and relationships:
//
//	if err := db.AutoMigrate(&User{}); err != nil {
//		return err
//	}
//
// Alternatively, for an explicitly planned migration where the column is known
// to be absent and User declares DisplayName:
//
//	if err := db.Migrator().AddColumn(&User{}, "DisplayName"); err != nil {
//		return err
//	}
//
// Table-name strings work for operations such as HasTable and DropTable. Calls
// that derive column definitions, indexes, or constraints need a model schema.
// Column operations generally accept a Go field name or its database column
// name; index and constraint operations have their own resolution rules. Read the
// individual method contract instead of assuming all names are interchangeable.
//
// # Execution, dependencies, and errors
//
// AutoMigrate includes relationship dependencies and join tables unless
// IgnoreRelationshipsWhenMigrating is set. DisableForeignKeyConstraintWhenMigrating
// suppresses foreign-key DDL without itself disabling dependency discovery.
// Explicit CreateTable does not automatically add or dependency-sort the supplied
// models; order those models to satisfy the database's constraints. Fields marked
// to ignore migration are skipped. Dropping tables or columns is destructive and
// is only performed by the corresponding explicit operation.
//
// A migration can execute several statements and stop after partial success.
// These methods do not create one transaction around the whole migration, and
// transactional DDL support depends on the database. A caller-provided transaction
// can only offer the guarantees supported by that database and its dialect.
// Context and logger settings come from the GORM session used to obtain the
// migrator. AutoMigrate in DryRun mode still queries the real database to inspect
// the existing schema; it prints planned DDL instead of applying that DDL.
//
// HasTable, HasColumn, HasIndex, and HasConstraint return only booleans. The shared
// implementation suppresses inspection errors, so false alone cannot distinguish
// absence from a permission, connection, or parsing failure. Metadata methods
// that return an error must be checked before their result is consumed. For
// ColumnType and other metadata records, an availability flag of false means
// unknown, not a known zero or false value.
//
// # Dialect extension and trusted SQL
//
// Dialects embed Migrator and override operations their SQL engine implements
// differently. The shared implementation uses information_schema queries and
// several SQL forms that are not universally supported. GetIndexes and TableType
// intentionally return errors unless overridden. BuildIndexOptionsInterface is
// also required by the shared index-building paths.
//
// GormDataTypeInterface customizes a field's database type for migrations.
// Schema tags, index expressions/options, check constraints, view CheckOption,
// and the gorm:table_options setting can contribute raw SQL to DDL. They are
// trusted application configuration, not places for unvalidated request input.
// Quoting identifiers and binding ordinary values does not sanitize these SQL
// fragments.
//
// See the migration guide at https://gorm.io/docs/migration.html, model mapping at
// https://gorm.io/docs/models.html, indexes at https://gorm.io/docs/indexes.html,
// constraints at https://gorm.io/docs/constraints.html, and custom data types at
// https://gorm.io/docs/data_types.html.
package migrator
