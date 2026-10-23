package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// regFullDataType extracts digit sequences from a rendered type declaration for
// MigrateColumn's fallback size comparison. This is a heuristic for type text, not a
// SQL type parser; only a single match is used for that comparison.
var regFullDataType = regexp.MustCompile(`\D*(\d+)\D?`)

// The compile-time assertion checks that the shared implementation satisfies
// gorm.Migrator. Dialects embed it and override methods for their SQL and metadata.
var _ gorm.Migrator = (*Migrator)(nil)

// Migrator is the shared implementation embedded by database-specific migrators.
// Obtain an application's migrator through db.Migrator() so dialect overrides are
// retained. This base implementation uses information_schema and several
// MySQL/PostgreSQL-style statements that are not portable on their own.
//
// Migration calls use Config.DB and return execution errors where the API permits;
// they do not wrap a multi-statement migration in a transaction. Schema changes may
// therefore be partially applied when a later operation fails.
type Migrator struct {
	// Config supplies the session and dialect used by shared migration methods.
	Config
}

// Config supplies a database session, dialect, and index-creation strategy to the
// shared Migrator. Dialect implementations initialize it when constructing their
// migrator; the zero value cannot execute migrations.
type Config struct {
	// CreateIndexAfterCreateTable emits indexes as separate CreateIndex calls after a
	// successful CREATE TABLE instead of embedding them in the table definition. A later
	// index failure is returned but does not undo the table creation.
	CreateIndexAfterCreateTable bool
	// DB is the GORM session used for model parsing, naming, metadata queries, context,
	// logging, and DDL execution. Obtain it from the calling database operation; it must
	// not be nil.
	DB *gorm.DB
	// Dialector supplies SQL type mapping, quoting, placeholders, and value rendering.
	gorm.Dialector
}

// printSQLLogger prints migration DDL to standard output for AutoMigrate's DryRun
// execution session, then forwards traces to the wrapped logger. It does not make
// the accompanying schema-introspection queries dry runs.
type printSQLLogger struct {
	// Interface receives the original trace after SQL is printed.
	logger.Interface
}

// Trace prints the rendered SQL followed by a semicolon, then delegates the original
// trace callback and error to the wrapped logger. The callback may therefore be
// evaluated again by that logger.
func (l *printSQLLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, _ := fc()
	fmt.Println(sql + ";")
	l.Interface.Trace(ctx, begin, fc, err)
}

// GormDataTypeInterface lets a custom field type choose its database column type for
// migration. Implement it on the field type or its pointer; DataTypeOf checks a newly
// allocated pointer to the field's indirect type. This migration hook is separate
// from schema.GormDataTypeInterface, which supplies the logical GORM type.
type GormDataTypeInterface interface {
	// GormDBDataType returns a trusted SQL type declaration for db's dialect and field
	// metadata, or an empty string to fall back to Dialector.DataTypeOf. It must not rely
	// on the field instance containing a model value.
	GormDBDataType(*gorm.DB, *schema.Field) string
}

// RunWithValue resolves a model or table name into a fresh statement and calls fc.
// It carries the current DB statement's Table and TableExpr forward, parses models
// with that special table name, and returns parse or callback errors. A string only
// sets Table and leaves Schema nil; callbacks requiring fields must receive a model.
// It does not begin a transaction or recover callback panics.
func (m Migrator) RunWithValue(value interface{}, fc func(*gorm.Statement) error) error {
	stmt := &gorm.Statement{DB: m.DB}
	if m.DB.Statement != nil {
		stmt.Table = m.DB.Statement.Table
		stmt.TableExpr = m.DB.Statement.TableExpr
	}

	if table, ok := value.(string); ok {
		stmt.Table = table
	} else if err := stmt.ParseWithSpecialTableName(value, stmt.Table); err != nil {
		return err
	}

	return fc(stmt)
}

// DataTypeOf resolves a field's database type using GormDataTypeInterface first and
// Dialector.DataTypeOf when the custom hook is absent or returns an empty string.
// The field must be non-nil and parsed; returned text is SQL syntax, not a bound value.
func (m Migrator) DataTypeOf(field *schema.Field) string {
	fieldValue := reflect.New(field.IndirectFieldType)
	if dataTyper, ok := fieldValue.Interface().(GormDataTypeInterface); ok {
		if dataType := dataTyper.GormDBDataType(m.DB, field); dataType != "" {
			return dataType
		}
	}

	return m.Dialector.DataTypeOf(field)
}

// FullDataTypeOf builds the field's SQL type followed by NOT NULL and DEFAULT when
// configured. Parsed Go default values are rendered through the dialect; textual
// SQL defaults are inserted directly, and the special default "(-)" suppresses the
// DEFAULT clause. It does not itself append unique, check, or foreign-key constraints.
func (m Migrator) FullDataTypeOf(field *schema.Field) (expr clause.Expr) {
	expr.SQL = m.DataTypeOf(field)

	if field.NotNull {
		expr.SQL += " NOT NULL"
	}

	if field.HasDefaultValue && (field.DefaultValueInterface != nil || field.DefaultValue != "") {
		if field.DefaultValueInterface != nil {
			defaultStmt := &gorm.Statement{Vars: []interface{}{field.DefaultValueInterface}}
			m.Dialector.BindVarTo(defaultStmt, defaultStmt, field.DefaultValueInterface)
			expr.SQL += " DEFAULT " + m.Dialector.Explain(defaultStmt.SQL.String(), field.DefaultValueInterface)
		} else if field.DefaultValue != "(-)" {
			expr.SQL += " DEFAULT " + field.DefaultValue
		}
	}

	return
}

// GetQueryAndExecTx returns sessions for schema inspection and DDL execution; the
// names do not imply a database transaction. Normally both results share a session.
// For DryRun, queryTx disables DryRun so introspection still reaches the database,
// while execTx retains DryRun and prints planned DDL through printSQLLogger.
func (m Migrator) GetQueryAndExecTx() (queryTx, execTx *gorm.DB) {
	queryTx = m.DB.Session(&gorm.Session{})
	execTx = queryTx
	if m.DB.DryRun {
		queryTx.DryRun = false
		execTx = m.DB.Session(&gorm.Session{Logger: &printSQLLogger{Interface: m.DB.Logger}})
	}
	return queryTx, execTx
}

// AutoMigrate reconciles models with database metadata in dependency order, adding
// referenced models and join tables unless relationship migration is disabled. It
// creates absent tables; for existing tables it adds missing columns, calls the
// dialect's MigrateColumn, and creates missing foreign keys (unless
// DisableForeignKeyConstraintWhenMigrating or IgnoreRelationshipsWhenMigrating is
// set), checks, and indexes.
//
// It does not drop unused columns or rebuild existing indexes whose definitions
// changed. It can alter column types/defaults/comments and reconcile column unique
// constraints, so it is not an additive-only or data-loss-free operation. Inspect
// MigrateColumn and the selected dialect for exact comparison and DDL behavior.
//
// The first returned error stops subsequent work without rolling back prior DDL.
// DryRun still performs real schema-introspection queries. Use versioned migrations
// when deployment ordering, rollback, or reviewed SQL is required.
//
// See https://gorm.io/docs/migration.html.
func (m Migrator) AutoMigrate(values ...interface{}) error {
	for _, value := range m.ReorderModels(values, true) {
		queryTx, execTx := m.GetQueryAndExecTx()
		if !queryTx.Migrator().HasTable(value) {
			if err := execTx.Migrator().CreateTable(value); err != nil {
				return err
			}
		} else {
			if err := m.RunWithValue(value, func(stmt *gorm.Statement) error {

				if stmt.Schema == nil {
					return errors.New("failed to get schema")
				}

				columnTypes, err := queryTx.Migrator().ColumnTypes(value)
				if err != nil {
					return err
				}
				var (
					parseIndexes          = stmt.Schema.ParseIndexes()
					parseCheckConstraints = stmt.Schema.ParseCheckConstraints()
				)
				for _, dbName := range stmt.Schema.DBNames {
					var foundColumn gorm.ColumnType

					for _, columnType := range columnTypes {
						if columnType.Name() == dbName {
							foundColumn = columnType
							break
						}
					}

					if foundColumn == nil {
						// not found, add column
						if err = execTx.Migrator().AddColumn(value, dbName); err != nil {
							return err
						}
					} else {
						// found, smartly migrate
						field := stmt.Schema.FieldsByDBName[dbName]
						if err = execTx.Migrator().MigrateColumn(value, field, foundColumn); err != nil {
							return err
						}
					}
				}

				if !m.DB.DisableForeignKeyConstraintWhenMigrating && !m.DB.IgnoreRelationshipsWhenMigrating {
					for _, rel := range stmt.Schema.Relationships.Relations {
						if rel.Field.IgnoreMigration {
							continue
						}
						if constraint := rel.ParseConstraint(); constraint != nil &&
							constraint.Schema == stmt.Schema && !queryTx.Migrator().HasConstraint(value, constraint.Name) {
							if err := execTx.Migrator().CreateConstraint(value, constraint.Name); err != nil {
								return err
							}
						}
					}
				}

				for _, chk := range parseCheckConstraints {
					if !queryTx.Migrator().HasConstraint(value, chk.Name) {
						if err := execTx.Migrator().CreateConstraint(value, chk.Name); err != nil {
							return err
						}
					}
				}

				for _, idx := range parseIndexes {
					if !queryTx.Migrator().HasIndex(value, idx.Name) {
						if err := execTx.Migrator().CreateIndex(value, idx.Name); err != nil {
							return err
						}
					}
				}

				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetTables queries information_schema.tables for names in CurrentDatabase. The base
// query does not filter out views or other table types. Dialects override this method
// when their catalog differs; query errors are returned.
func (m Migrator) GetTables() (tableList []string, err error) {
	err = m.DB.Raw("SELECT TABLE_NAME FROM information_schema.tables where TABLE_SCHEMA=?", m.CurrentDatabase()).
		Scan(&tableList).Error
	return
}

// CreateTable creates tables from model schemas with columns, primary keys, indexes,
// unique/check constraints, and enabled foreign keys. It skips fields marked to
// ignore migration and appends the trusted SQL in the gorm:table_options setting.
//
// Unlike AutoMigrate, it does not check existence or automatically add referenced
// models. Supplied models retain their order (apart from deduplication and strings
// handled by ReorderModels); callers must supply dependency-safe ordering. A plain
// table-name string has no schema and produces an error. Indexes may execute after
// the table according to CreateIndexAfterCreateTable. The first error stops work
// without undoing earlier tables or indexes.
func (m Migrator) CreateTable(values ...interface{}) error {
	for _, value := range m.ReorderModels(values, false) {
		tx := m.DB.Session(&gorm.Session{})
		if err := m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {

			if stmt.Schema == nil {
				return errors.New("failed to get schema")
			}

			var (
				createTableSQL          = "CREATE TABLE ? ("
				values                  = []interface{}{m.CurrentTable(stmt)}
				hasPrimaryKeyInDataType bool
			)

			for _, dbName := range stmt.Schema.DBNames {
				field := stmt.Schema.FieldsByDBName[dbName]
				if !field.IgnoreMigration {
					createTableSQL += "? ?"
					hasPrimaryKeyInDataType = hasPrimaryKeyInDataType || strings.Contains(strings.ToUpper(m.DataTypeOf(field)), "PRIMARY KEY")
					values = append(values, clause.Column{Name: dbName}, m.DB.Migrator().FullDataTypeOf(field))
					createTableSQL += ","
				}
			}

			if !hasPrimaryKeyInDataType && len(stmt.Schema.PrimaryFields) > 0 {
				createTableSQL += "PRIMARY KEY ?,"
				primaryKeys := make([]interface{}, 0, len(stmt.Schema.PrimaryFields))
				for _, field := range stmt.Schema.PrimaryFields {
					primaryKeys = append(primaryKeys, clause.Column{Name: field.DBName})
				}

				values = append(values, primaryKeys)
			}

			for _, idx := range stmt.Schema.ParseIndexes() {
				if m.CreateIndexAfterCreateTable {
					defer func(value interface{}, name string) {
						if err == nil {
							err = tx.Migrator().CreateIndex(value, name)
						}
					}(value, idx.Name)
				} else {
					if idx.Class != "" {
						createTableSQL += idx.Class + " "
					}
					createTableSQL += "INDEX ? ?"

					if idx.Comment != "" {
						createTableSQL += fmt.Sprintf(" COMMENT '%s'", idx.Comment)
					}

					if idx.Option != "" {
						createTableSQL += " " + idx.Option
					}

					createTableSQL += ","
					values = append(values, clause.Column{Name: idx.Name}, tx.Migrator().(BuildIndexOptionsInterface).BuildIndexOptions(idx.Fields, stmt))
				}
			}

			if !m.DB.DisableForeignKeyConstraintWhenMigrating && !m.DB.IgnoreRelationshipsWhenMigrating {
				for _, rel := range stmt.Schema.Relationships.Relations {
					if rel.Field.IgnoreMigration {
						continue
					}
					if constraint := rel.ParseConstraint(); constraint != nil {
						if constraint.Schema == stmt.Schema {
							sql, vars := constraint.Build()
							createTableSQL += sql + ","
							values = append(values, vars...)
						}
					}
				}
			}

			for _, uni := range stmt.Schema.ParseUniqueConstraints() {
				createTableSQL += "CONSTRAINT ? UNIQUE (?),"
				values = append(values, clause.Column{Name: uni.Name}, clause.Expr{SQL: stmt.Quote(uni.Field.DBName)})
			}

			for _, chk := range stmt.Schema.ParseCheckConstraints() {
				createTableSQL += "CONSTRAINT ? CHECK (?),"
				values = append(values, clause.Column{Name: chk.Name}, clause.Expr{SQL: chk.Constraint})
			}

			createTableSQL = strings.TrimSuffix(createTableSQL, ",")

			createTableSQL += ")"

			if tableOption, ok := m.DB.Get("gorm:table_options"); ok {
				createTableSQL += fmt.Sprint(tableOption)
			}

			err = tx.Exec(createTableSQL, values...).Error
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

// DropTable issues DROP TABLE IF EXISTS for each model or table name in reverse
// ReorderModels order. This destroys table data and does not cascade constraints or
// add related models in the base implementation. It stops on the first execution
// error without restoring tables already dropped; dialect overrides may handle
// foreign-key dependencies differently.
func (m Migrator) DropTable(values ...interface{}) error {
	values = m.ReorderModels(values, false)
	for i := len(values) - 1; i >= 0; i-- {
		tx := m.DB.Session(&gorm.Session{})
		if err := m.RunWithValue(values[i], func(stmt *gorm.Statement) error {
			return tx.Exec("DROP TABLE IF EXISTS ?", m.CurrentTable(stmt)).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

// HasTable reports whether a model or table name resolves to a BASE TABLE in the
// current database's information_schema. Views are excluded by this base query.
// Parse and query errors are not returned, so false is not proof of absence when
// metadata access fails.
func (m Migrator) HasTable(value interface{}) bool {
	var count int64

	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := m.DB.Migrator().CurrentDatabase()
		return m.DB.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ? AND table_type = ?", currentDatabase, stmt.Table, "BASE TABLE").Row().Scan(&count)
	})

	return count > 0
}

// RenameTable issues ALTER TABLE old RENAME TO new. Each argument may be a table-name
// string or a model whose table name is parsed. It returns parse or execution errors
// and does not check whether the source exists or the destination is available.
func (m Migrator) RenameTable(oldName, newName interface{}) error {
	var oldTable, newTable interface{}
	if v, ok := oldName.(string); ok {
		oldTable = clause.Table{Name: v}
	} else {
		stmt := &gorm.Statement{DB: m.DB}
		if err := stmt.Parse(oldName); err == nil {
			oldTable = m.CurrentTable(stmt)
		} else {
			return err
		}
	}

	if v, ok := newName.(string); ok {
		newTable = clause.Table{Name: v}
	} else {
		stmt := &gorm.Statement{DB: m.DB}
		if err := stmt.Parse(newName); err == nil {
			newTable = m.CurrentTable(stmt)
		} else {
			return err
		}
	}

	return m.DB.Exec("ALTER TABLE ? RENAME TO ?", oldTable, newTable).Error
}

// AddColumn looks up name as a Go field name or database column in a model schema
// and adds it using the dialect's FullDataTypeOf. Missing schema/fields return an
// error, while IgnoreMigration fields are a no-op. It does not check existence or
// separately add that field's indexes and relationship constraints.
func (m Migrator) AddColumn(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		// avoid using the same name field
		if stmt.Schema == nil {
			return errors.New("failed to get schema")
		}
		f := stmt.Schema.LookUpField(name)
		if f == nil {
			return fmt.Errorf("failed to look up field with name: %s", name)
		}

		if !f.IgnoreMigration {
			return m.DB.Exec(
				"ALTER TABLE ? ADD ? ?",
				m.CurrentTable(stmt), clause.Column{Name: f.DBName}, m.DB.Migrator().FullDataTypeOf(f),
			).Error
		}

		return nil
	})
}

// DropColumn drops a column, resolving name through the model schema when possible
// and otherwise treating it as a database column name. A table-name string is also
// accepted. It does not check existence, preserve data, or update related indexes
// and constraints itself; database errors are returned.
func (m Migrator) DropColumn(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(name); field != nil {
				name = field.DBName
			}
		}

		return m.DB.Exec(
			"ALTER TABLE ? DROP COLUMN ?", m.CurrentTable(stmt), clause.Column{Name: name},
		).Error
	})
}

// AlterColumn resolves field from a model schema and executes ALTER TABLE ... ALTER
// COLUMN ... TYPE using FullDataTypeOf. Unknown fields return an error. This generic
// DDL is not supported uniformly; use db.Migrator() to retain dialect overrides and
// review possible conversions, locks, and data effects before applying it.
func (m Migrator) AlterColumn(value interface{}, field string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(field); field != nil {
				fieldType := m.FullDataTypeOf(field)
				return m.DB.Exec(
					"ALTER TABLE ? ALTER COLUMN ? TYPE ?",
					m.CurrentTable(stmt), clause.Column{Name: field.DBName}, fieldType,
				).Error

			}
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

// HasColumn checks information_schema.columns for a field in the current database.
// With a model, field is resolved from a Go field name or database name; otherwise it
// is used literally. Parse and query failures are not returned, so false can also
// mean metadata lookup failed.
func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := m.DB.Migrator().CurrentDatabase()
		name := field
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(field); field != nil {
				name = field.DBName
			}
		}

		return m.DB.Raw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?",
			currentDatabase, stmt.Table, name,
		).Row().Scan(&count)
	})

	return count > 0
}

// RenameColumn issues ALTER TABLE ... RENAME COLUMN, mapping oldName and newName to
// database column names when they match model fields. Unresolved names are used
// literally; a table-name string is accepted. It does not perform existence checks
// or automatically change the model's tags.
func (m Migrator) RenameColumn(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(oldName); field != nil {
				oldName = field.DBName
			}

			if field := stmt.Schema.LookUpField(newName); field != nil {
				newName = field.DBName
			}
		}

		return m.DB.Exec(
			"ALTER TABLE ? RENAME COLUMN ? TO ?",
			m.CurrentTable(stmt), clause.Column{Name: oldName}, clause.Column{Name: newName},
		).Error
	})
}

// MigrateColumn compares one parsed model field with live column metadata and asks
// the dialect to AlterColumn when supported comparisons differ. It skips
// IgnoreMigration fields, uses type aliases and size/precision heuristics, and
// compares defaults and comments when metadata is available. Primary keys are
// excluded from several, but not all, comparison paths.
//
// A nullability mismatch triggers an alteration only from NOT NULL to nullable for
// non-primary-key fields; this base implementation does not tighten a nullable
// column to NOT NULL. Other simultaneous alterations still use the full model type.
// After any alteration it calls MigrateColumnUnique. Unknown metadata is not a
// universal indication that definitions match, and driver overrides determine the
// actual SQL and conversion behavior.
func (m Migrator) MigrateColumn(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	if field.IgnoreMigration {
		return nil
	}

	// found, smart migrate
	fullDataType := strings.TrimSpace(strings.ToLower(m.DB.Migrator().FullDataTypeOf(field).SQL))
	realDataType := strings.ToLower(columnType.DatabaseTypeName())
	var (
		alterColumn bool
		isSameType  = fullDataType == realDataType
	)

	if !field.PrimaryKey {
		// check type
		if !strings.HasPrefix(fullDataType, realDataType) {
			// check type aliases
			aliases := m.DB.Migrator().GetTypeAliases(realDataType)
			for _, alias := range aliases {
				if strings.HasPrefix(fullDataType, alias) {
					isSameType = true
					break
				}
			}

			if !isSameType {
				alterColumn = true
			}
		}
	}

	if !isSameType {
		// check size
		if length, ok := columnType.Length(); length != int64(field.Size) {
			if length > 0 && field.Size > 0 {
				alterColumn = true
			} else {
				// has size in data type and not equal
				// Since the following code is frequently called in the for loop, reg optimization is needed here
				matches2 := regFullDataType.FindAllStringSubmatch(fullDataType, -1)
				if !field.PrimaryKey &&
					(len(matches2) == 1 && matches2[0][1] != fmt.Sprint(length) && ok) {
					alterColumn = true
				}
			}
		}
	}

	// check precision
	if realDataType == "decimal" || realDataType == "numeric" &&
		regexp.MustCompile(realDataType+`\(.*\)`).FindString(fullDataType) != "" { // if realDataType has no precision,ignore
		precision, scale, ok := columnType.DecimalSize()
		if ok {
			if !strings.HasPrefix(fullDataType, fmt.Sprintf("%s(%d,%d)", realDataType, precision, scale)) &&
				!strings.HasPrefix(fullDataType, fmt.Sprintf("%s(%d)", realDataType, precision)) {
				alterColumn = true
			}
		}
	} else {
		if precision, _, ok := columnType.DecimalSize(); ok && int64(field.Precision) != precision {
			if regexp.MustCompile(fmt.Sprintf("[^0-9]%d[^0-9]", field.Precision)).MatchString(m.DataTypeOf(field)) {
				alterColumn = true
			}
		}
	}

	// check nullable
	if nullable, ok := columnType.Nullable(); ok && nullable == field.NotNull {
		// not primary key & current database is non-nullable(to be nullable)
		if !field.PrimaryKey && !nullable {
			alterColumn = true
		}
	}

	// check default value
	if !field.PrimaryKey {
		currentDefaultNotNull := field.HasDefaultValue && (field.DefaultValueInterface != nil || !strings.EqualFold(field.DefaultValue, "NULL"))
		dv, dvNotNull := columnType.DefaultValue()
		if dvNotNull && !currentDefaultNotNull {
			// default value -> null
			alterColumn = true
		} else if !dvNotNull && currentDefaultNotNull {
			// null -> default value
			alterColumn = true
		} else if currentDefaultNotNull || dvNotNull {
			switch field.GORMDataType {
			case schema.Time:
				if !strings.EqualFold(strings.TrimSuffix(dv, "()"), strings.TrimSuffix(field.DefaultValue, "()")) {
					alterColumn = true
				}
			case schema.Bool:
				v1, _ := strconv.ParseBool(dv)
				v2, _ := strconv.ParseBool(field.DefaultValue)
				if v1 != v2 {
					alterColumn = true
				}
			case schema.String:
				if dv != field.DefaultValue && dv != strings.Trim(field.DefaultValue, "'\"") {
					alterColumn = true
				}
			default:
				if dv != field.DefaultValue {
					alterColumn = true
				}
			}
		}
	}

	// check comment
	if comment, ok := columnType.Comment(); ok && comment != field.Comment {
		// not primary key
		if !field.PrimaryKey {
			alterColumn = true
		}
	}

	if alterColumn {
		if err := m.DB.Migrator().AlterColumn(value, field.DBName); err != nil {
			return err
		}
	}

	if err := m.DB.Migrator().MigrateColumnUnique(value, field, columnType); err != nil {
		return err
	}

	return nil
}

// MigrateColumnUnique reconciles a non-primary-key field's Unique flag with known
// column-level uniqueness metadata. It creates or drops the naming strategy's
// unique constraint when the booleans differ, and does nothing when metadata is
// unknown. UniqueIndex is handled separately and is not consulted here.
func (m Migrator) MigrateColumnUnique(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	unique, ok := columnType.Unique()
	if !ok || field.PrimaryKey {
		return nil // skip primary key
	}
	// By default, ColumnType's Unique is not affected by UniqueIndex, so we don't care about UniqueIndex.
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		// We're currently only receiving boolean values on `Unique` tag,
		// so the UniqueConstraint name is fixed
		constraint := m.DB.NamingStrategy.UniqueName(stmt.Table, field.DBName)
		if unique && !field.Unique {
			return m.DB.Migrator().DropConstraint(value, constraint)
		}
		if !unique && field.Unique {
			return m.DB.Migrator().CreateConstraint(value, constraint)
		}
		return nil
	})
}

// ColumnTypes queries at most one row to obtain database/sql column metadata and
// wraps each result as ColumnType. It closes the rows before returning; no row must
// exist for the driver to describe columns. This base path does not populate
// catalog-only properties such as primary-key membership or defaults.
//
// The deferred rows.Close result replaces the callback's named error, including a
// previous rows.ColumnTypes error. Dialects commonly override this method to enrich
// metadata or use their own catalog queries.
func (m Migrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	columnTypes := make([]gorm.ColumnType, 0)
	execErr := m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
		rows, err := m.DB.Session(&gorm.Session{}).Table(stmt.Table).Limit(1).Rows()
		if err != nil {
			return err
		}

		defer func() {
			err = rows.Close()
		}()

		var rawColumnTypes []*sql.ColumnType
		rawColumnTypes, err = rows.ColumnTypes()
		if err != nil {
			return err
		}

		for _, c := range rawColumnTypes {
			columnTypes = append(columnTypes, ColumnType{SQLColumnType: c})
		}

		return
	})

	return columnTypes, execErr
}

// CreateView creates a named view from option.Query, returning gorm.ErrSubQueryRequired
// when Query is nil. Replace selects CREATE OR REPLACE; CheckOption is appended as
// trusted SQL. The query is rendered through the dialect into the DDL rather than
// executed as a SELECT. Replacement and check-option support depend on the database.
//
//	query := db.Model(&User{}).Where("age > ?", 20)
//	err := db.Migrator().CreateView("adult_users", gorm.ViewOption{Query: query})
//
// Use a fresh session for each view definition: this implementation adds subquery
// variables to the receiver's statement while building the SQL.
// See https://gorm.io/docs/migration.html#Views.
func (m Migrator) CreateView(name string, option gorm.ViewOption) error {
	if option.Query == nil {
		return gorm.ErrSubQueryRequired
	}

	sql := new(strings.Builder)
	sql.WriteString("CREATE ")
	if option.Replace {
		sql.WriteString("OR REPLACE ")
	}
	sql.WriteString("VIEW ")
	m.QuoteTo(sql, name)
	sql.WriteString(" AS ")

	m.DB.Statement.AddVar(sql, option.Query)

	if option.CheckOption != "" {
		sql.WriteString(" ")
		sql.WriteString(option.CheckOption)
	}
	return m.DB.Exec(m.Explain(sql.String(), m.DB.Statement.Vars...)).Error
}

// DropView executes DROP VIEW IF EXISTS with a quoted view name. It returns execution
// errors and does not remove dependent views or request CASCADE.
func (m Migrator) DropView(name string) error {
	return m.DB.Exec("DROP VIEW IF EXISTS ?", clause.Table{Name: name}).Error
}

// GuessConstraintAndTable resolves a constraint through
// GuessConstraintInterfaceAndTable
// and exposes only foreign-key and check constraint results. A resolved unique
// constraint produces two nil constraint results while retaining the resolved table.
//
// Deprecated: use GuessConstraintInterfaceAndTable instead.
func (m Migrator) GuessConstraintAndTable(stmt *gorm.Statement, name string) (*schema.Constraint, *schema.CheckConstraint, string) {
	constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
	switch c := constraint.(type) {
	case *schema.Constraint:
		return c, nil, table
	case *schema.CheckConstraint:
		return nil, c, table
	default:
		return nil, nil, table
	}
}

// GuessConstraintInterfaceAndTable resolves a check, unique, or relationship
// foreign-key constraint by constraint name or associated Go/database field name.
// For relationship constraints it selects the table that owns the foreign key,
// which may be an associated or join table. With no schema it returns nil and
// stmt.Table; an unresolved name with a schema returns nil and the schema's table.
// A field associated with several constraints does not provide a stable choice;
// use the explicit constraint name when ambiguity matters.
//
// nolint:cyclop
func (m Migrator) GuessConstraintInterfaceAndTable(stmt *gorm.Statement, name string) (_ schema.ConstraintInterface, table string) {
	if stmt.Schema == nil {
		return nil, stmt.Table
	}

	checkConstraints := stmt.Schema.ParseCheckConstraints()
	if chk, ok := checkConstraints[name]; ok {
		return &chk, stmt.Table
	}

	uniqueConstraints := stmt.Schema.ParseUniqueConstraints()
	if uni, ok := uniqueConstraints[name]; ok {
		return &uni, stmt.Table
	}

	getTable := func(rel *schema.Relationship) string {
		switch rel.Type {
		case schema.HasOne, schema.HasMany:
			return rel.FieldSchema.Table
		case schema.Many2Many:
			return rel.JoinTable.Table
		}
		return stmt.Table
	}

	for _, rel := range stmt.Schema.Relationships.Relations {
		if constraint := rel.ParseConstraint(); constraint != nil && constraint.Name == name {
			return constraint, getTable(rel)
		}
	}

	if field := stmt.Schema.LookUpField(name); field != nil {
		for k := range checkConstraints {
			if checkConstraints[k].Field == field {
				v := checkConstraints[k]
				return &v, stmt.Table
			}
		}

		for k := range uniqueConstraints {
			if uniqueConstraints[k].Field == field {
				v := uniqueConstraints[k]
				return &v, stmt.Table
			}
		}

		for _, rel := range stmt.Schema.Relationships.Relations {
			if constraint := rel.ParseConstraint(); constraint != nil && rel.Field == field {
				return constraint, getTable(rel)
			}
		}
	}

	return nil, stmt.Schema.Table
}

// CreateConstraint resolves a model constraint or relationship field name and adds
// its check, unique, or foreign-key definition to the owning table. An explicit
// TableExpr takes precedence over the resolved table. An unknown constraint is a
// successful no-op; this method does not check whether a known constraint already
// exists.
func (m Migrator) CreateConstraint(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
		if constraint != nil {
			vars := []interface{}{clause.Table{Name: table}}
			if stmt.TableExpr != nil {
				vars[0] = stmt.TableExpr
			}
			sql, values := constraint.Build()
			return m.DB.Exec("ALTER TABLE ? ADD "+sql, append(vars, values...)...).Error
		}
		return nil
	})
}

// DropConstraint removes the resolved constraint from its owning table. If name
// does not resolve through model metadata, it is used as a literal constraint name
// on the resolved table. It does not use IF EXISTS or check existence first.
func (m Migrator) DropConstraint(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
		if constraint != nil {
			name = constraint.GetName()
		}
		return m.DB.Exec("ALTER TABLE ? DROP CONSTRAINT ?", clause.Table{Name: table}, clause.Column{Name: name}).Error
	})
}

// HasConstraint resolves model constraint or relationship field names and queries
// information_schema.table_constraints on the owning table. An unresolved name is
// queried literally. Parse/query errors are not returned, so false can mean either
// absence or failure to inspect metadata.
func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := m.DB.Migrator().CurrentDatabase()
		constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
		if constraint != nil {
			name = constraint.GetName()
		}

		return m.DB.Raw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.table_constraints WHERE constraint_schema = ? AND table_name = ? AND constraint_name = ?",
			currentDatabase, table, name,
		).Row().Scan(&count)
	})

	return count > 0
}

// BuildIndexOptions renders index fields in the supplied order. It quotes ordinary
// column names, uses a trusted Expression verbatim instead when set, applies prefix
// Length to ordinary columns, and appends Collate and Sort SQL. The returned
// clause.Expr values contain SQL syntax, so expression/collation/sort inputs must be
// trusted schema configuration rather than request data.
func (m Migrator) BuildIndexOptions(opts []schema.IndexOption, stmt *gorm.Statement) (results []interface{}) {
	for _, opt := range opts {
		str := stmt.Quote(opt.DBName)
		if opt.Expression != "" {
			str = opt.Expression
		} else if opt.Length > 0 {
			str += fmt.Sprintf("(%d)", opt.Length)
		}

		if opt.Collate != "" {
			str += " COLLATE " + opt.Collate
		}

		if opt.Sort != "" {
			str += " " + opt.Sort
		}
		results = append(results, clause.Expr{SQL: str})
	}
	return
}

// BuildIndexOptionsInterface is the additional dialect hook used to render indexed
// columns and expressions. Shared CreateTable and CreateIndex type-assert the
// active db.Migrator() to this interface; custom migrators using these methods must
// implement it, normally by embedding Migrator.
type BuildIndexOptionsInterface interface {
	// BuildIndexOptions returns ordered SQL builder values for the index fields using
	// stmt's quoting rules and the dialect's expression/length/collation/sort syntax.
	BuildIndexOptions([]schema.IndexOption, *gorm.Statement) []interface{}
}

// CreateIndex resolves an index name or indexed model field through schema.LookIndex
// and executes its definition using the dialect's BuildIndexOptions. Missing schema
// or index metadata returns an error. Class, type, comment, and option text enter
// DDL directly and must be trusted; this base implementation does not render the
// index's Where predicate. It does not check existence before creating the index.
func (m Migrator) CreateIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema == nil {
			return errors.New("failed to get schema")
		}
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			opts := m.DB.Migrator().(BuildIndexOptionsInterface).BuildIndexOptions(idx.Fields, stmt)
			values := []interface{}{clause.Column{Name: idx.Name}, m.CurrentTable(stmt), opts}

			createIndexSQL := "CREATE "
			if idx.Class != "" {
				createIndexSQL += idx.Class + " "
			}
			createIndexSQL += "INDEX ? ON ??"

			if idx.Type != "" {
				createIndexSQL += " USING " + idx.Type
			}

			if idx.Comment != "" {
				createIndexSQL += fmt.Sprintf(" COMMENT '%s'", idx.Comment)
			}

			if idx.Option != "" {
				createIndexSQL += " " + idx.Option
			}

			return m.DB.Exec(createIndexSQL, values...).Error
		}

		return fmt.Errorf("failed to create index with name %s", name)
	})
}

// DropIndex resolves an indexed model field to its index name when possible, then
// issues DROP INDEX ... ON the table. Unresolved names are used literally. It does
// not use IF EXISTS; the database decides how missing or required indexes fail.
func (m Migrator) DropIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				name = idx.Name
			}
		}

		return m.DB.Exec("DROP INDEX ? ON ?", clause.Column{Name: name}, m.CurrentTable(stmt)).Error
	})
}

// HasIndex resolves an index name or indexed model field and checks the current
// database's information_schema.statistics. Parse/query errors are not returned, so
// false is not a reliable absence check when metadata access fails.
func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := m.DB.Migrator().CurrentDatabase()
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				name = idx.Name
			}
		}

		return m.DB.Raw(
			"SELECT count(*) FROM information_schema.statistics WHERE table_schema = ? AND table_name = ? AND index_name = ?",
			currentDatabase, stmt.Table, name,
		).Row().Scan(&count)
	})

	return count > 0
}

// RenameIndex issues ALTER TABLE ... RENAME INDEX using oldName and newName as actual
// database index names. Unlike CreateIndex and DropIndex, this base method does not
// resolve model field names to index names. Dialect overrides may use different SQL
// or add their own name resolution.
func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Exec(
			"ALTER TABLE ? RENAME INDEX ? TO ?",
			m.CurrentTable(stmt), clause.Column{Name: oldName}, clause.Column{Name: newName},
		).Error
	})
}

// CurrentDatabase executes SELECT DATABASE() and returns the scanned name. It does
// not return query/scan errors; the result can be empty on failure. Databases with
// different current-schema semantics override this base implementation.
func (m Migrator) CurrentDatabase() (name string) {
	m.DB.Raw("SELECT DATABASE()").Row().Scan(&name)
	return
}

// ReorderModels parses and deduplicates model schemas by table and returns table-name
// strings first, followed by models. With autoAdd, it recursively adds relationship
// dependencies and join tables, ordering dependencies before their dependents and
// using visited sets to break cycles. Cycles are not rejected or guaranteed to be
// valid for a database's foreign-key creation rules.
//
// With autoAdd false, model dependencies are not recursively ordered or appended;
// the explicitly supplied models retain their first-seen order after deduplication.
// IgnoreRelationshipsWhenMigrating disables relationship discovery. Inputs must be
// valid models or table names: parse errors are logged, not returned, and subsequent
// schema access is not a safe error-reporting path.
func (m Migrator) ReorderModels(values []interface{}, autoAdd bool) (results []interface{}) {
	// Dependency associates a parsed migration target with referenced schemas.
	type Dependency struct {
		// Statement retains the parsed schema and original destination value.
		*gorm.Statement
		// Depends lists schemas that should precede this model when autoAdd is true.
		Depends []*schema.Schema
	}

	var (
		modelNames, orderedModelNames []string
		orderedModelNamesMap          = map[string]bool{}
		parsedSchemas                 = map[*schema.Schema]bool{}
		valuesMap                     = map[string]Dependency{}
		insertIntoOrderedList         func(name string)
		parseDependence               func(value interface{}, addToList bool)
	)

	parseDependence = func(value interface{}, addToList bool) {
		dep := Dependency{
			Statement: &gorm.Statement{DB: m.DB, Dest: value},
		}
		beDependedOn := map[*schema.Schema]bool{}
		// support for special table name
		if err := dep.ParseWithSpecialTableName(value, m.DB.Statement.Table); err != nil {
			m.DB.Logger.Error(context.Background(), "failed to parse value %#v, got error %v", value, err)
		}
		if _, ok := parsedSchemas[dep.Statement.Schema]; ok {
			return
		}
		parsedSchemas[dep.Statement.Schema] = true

		if !m.DB.IgnoreRelationshipsWhenMigrating {
			for _, rel := range dep.Schema.Relationships.Relations {
				if rel.Field.IgnoreMigration {
					continue
				}
				if c := rel.ParseConstraint(); c != nil && c.Schema == dep.Statement.Schema && c.Schema != c.ReferenceSchema {
					dep.Depends = append(dep.Depends, c.ReferenceSchema)
				}

				if rel.Type == schema.HasOne || rel.Type == schema.HasMany {
					beDependedOn[rel.FieldSchema] = true
				}

				if rel.JoinTable != nil {
					// append join value
					defer func(rel *schema.Relationship, joinValue interface{}) {
						if !beDependedOn[rel.FieldSchema] {
							dep.Depends = append(dep.Depends, rel.FieldSchema)
						} else {
							fieldValue := reflect.New(rel.FieldSchema.ModelType).Interface()
							parseDependence(fieldValue, autoAdd)
						}
						parseDependence(joinValue, autoAdd)
					}(rel, reflect.New(rel.JoinTable.ModelType).Interface())
				}
			}
		}

		valuesMap[dep.Schema.Table] = dep

		if addToList {
			modelNames = append(modelNames, dep.Schema.Table)
		}
	}

	insertIntoOrderedList = func(name string) {
		if _, ok := orderedModelNamesMap[name]; ok {
			return // avoid loop
		}
		orderedModelNamesMap[name] = true

		if autoAdd {
			dep := valuesMap[name]
			for _, d := range dep.Depends {
				if _, ok := valuesMap[d.Table]; ok {
					insertIntoOrderedList(d.Table)
				} else {
					parseDependence(reflect.New(d.ModelType).Interface(), autoAdd)
					insertIntoOrderedList(d.Table)
				}
			}
		}

		orderedModelNames = append(orderedModelNames, name)
	}

	for _, value := range values {
		if v, ok := value.(string); ok {
			results = append(results, v)
		} else {
			parseDependence(value, true)
		}
	}

	for _, name := range modelNames {
		insertIntoOrderedList(name)
	}

	for _, name := range orderedModelNames {
		results = append(results, valuesMap[name].Statement.Dest)
	}
	return
}

// CurrentTable returns the explicit TableExpr value when present, otherwise a
// clause.Table for stmt.Table. The caller's SQL builder quotes the ordinary name;
// a TableExpr carries caller-supplied SQL and must already be trusted.
func (m Migrator) CurrentTable(stmt *gorm.Statement) interface{} {
	if stmt.TableExpr != nil {
		return *stmt.TableExpr
	}
	return clause.Table{Name: stmt.Table}
}

// GetIndexes returns an unsupported-operation error in the shared implementation.
// A dialect must override it to return database index metadata; callers must check
// the error rather than interpret a nil slice as a table with no indexes.
func (m Migrator) GetIndexes(dst interface{}) ([]gorm.Index, error) {
	return nil, errors.New("not support")
}

// GetTypeAliases returns no aliases in the shared implementation. Dialects override
// it with equivalent type names used by MigrateColumn's comparison heuristics;
// it does not query the database.
func (m Migrator) GetTypeAliases(databaseTypeName string) []string {
	return nil
}

// TableType returns an unsupported-operation error in the shared implementation.
// A dialect must override it to describe the table; callers must check the error
// before using the returned gorm.TableType.
func (m Migrator) TableType(dst interface{}) (gorm.TableType, error) {
	return nil, errors.New("not support")
}
