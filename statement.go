package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// Statement is the mutable state for one SQL operation or traditional chain. It holds
// model metadata, clauses, bound parameters, context, and the active pool/transaction.
// Plugins may inspect it in callbacks; application code normally uses DB chain methods.
// It is not safe for concurrent mutation, and clone only copies selected containers,
// not every referenced object.
type Statement struct {
	*DB
	// TableExpr holds an optional trusted table expression, taking precedence over simple
	// table quoting.
	TableExpr *clause.Expr
	// Table is the resolved table name or alias used to qualify model columns.
	Table string
	// Model supplies schema and owner identity; it can differ from the scan/write
	// destination.
	Model interface{}
	// Unscoped disables applicable model query/delete scopes such as soft deletion. It does
	// not discard user predicates.
	Unscoped bool
	// Dest is the result destination or write payload consumed by callbacks.
	Dest interface{}
	// ReflectValue is the dereferenced destination/model value used by field setters,
	// hooks, and scanners.
	ReflectValue reflect.Value
	// Clauses stores merged SQL clauses by name; builders decide their final order.
	Clauses map[string]clause.Clause
	// BuildClauses overrides clause order for this operation; the processor supplies
	// defaults when empty.
	BuildClauses []string
	// Distinct requests distinct selection and affects single-column Count.
	Distinct bool
	// Selects lists requested model fields or selection expressions used by query and write
	// builders.
	Selects []string // selected columns
	// Omits lists excluded fields and association paths; omission takes precedence over
	// selection.
	Omits []string // omit columns
	// ColumnMapping maps returned column names to schema fields while scanning.
	ColumnMapping map[string]string // map columns
	// Joins records association and raw joins for later SQL construction.
	Joins []join
	// Preloads maps association paths to their query customization arguments.
	Preloads map[string][]interface{}
	// Settings stores statement settings used by Set/Get, callbacks, and plugins; copied
	// settings can still hold shared mutable values.
	Settings sync.Map
	// ConnPool is the active execution target, which can be a pool, transaction, or
	// dedicated connection.
	ConnPool ConnPool
	// Schema is cached parsed model metadata, shared across statements using the same model
	// configuration.
	Schema *schema.Schema
	// Context carries cancellation, deadlines, and request values to database calls, hooks,
	// and logs.
	Context context.Context
	// RaiseErrorOnNotFound makes Scan add ErrRecordNotFound for an empty successful result;
	// single-row finishers enable it.
	RaiseErrorOnNotFound bool
	// SkipHooks suppresses model hooks where callbacks honor it; it does not disable the
	// callback pipeline.
	SkipHooks bool
	// SQL contains generated SQL with placeholders. Normal execution resets it after
	// tracing; DryRun preserves it.
	SQL strings.Builder
	// Vars contains values bound to SQL placeholders in dialect order. Treat them
	// separately from diagnostic interpolated SQL.
	Vars []interface{}
	// CurDestIndex identifies the current element during per-model hooks on a slice
	// destination.
	CurDestIndex int
	// attrs holds missing-record initialization values supplied by Attrs.
	attrs []interface{}
	// assigns holds unconditional FirstOrInit/FirstOrCreate assignments supplied by Assign.
	assigns []interface{}
	// scopes queues deferred traditional query transformations consumed before callback
	// execution.
	scopes []func(*DB) *DB
	// Result optionally points to a mutable WithResult collector shared with the caller.
	Result *result
}

// join stores a deferred association or raw SQL join, including alias, ON predicates,
// and selected related fields. Query callbacks translate it into SQL and scan metadata.
type join struct {
	// Name is the association path or raw join template.
	Name string
	// Alias is an explicit alias used to qualify joined columns and map scanned relation
	// paths.
	Alias string
	// Conds holds arguments accompanying a raw or association join.
	Conds []interface{}
	// On contains additional predicates for an association join's ON clause.
	On *clause.Where
	// Selects limits fields fetched from the joined model.
	Selects []string
	// Omits excludes fields from the joined model projection.
	Omits []string
	// Expression holds a prebuilt join expression, such as an aliased subquery join.
	Expression clause.Expression
	// JoinType selects LEFT, INNER, or another supported join form when the join is
	// generated.
	JoinType clause.JoinType
}

// StatementModifier adjusts statement state when supplied through supported clause or
// destination paths. It can install query behavior without rendering SQL directly;
// implementations must respect the current operation's mutable state and report errors
// through the attached DB.
type StatementModifier interface {
	// ModifyStatement mutates the supplied operation state. Clause registration or callback
	// execution invokes it; it should not assume that SQL has already been built.
	ModifyStatement(*Statement)
}

// WriteString appends trusted SQL text to the statement buffer. It performs no quoting
// or value binding; use AddVar for values and WriteQuoted for identifiers.
func (stmt *Statement) WriteString(str string) (int, error) {
	return stmt.SQL.WriteString(str)
}

// WriteByte appends one SQL byte without escaping. It implements clause.Writer for SQL
// builders.
func (stmt *Statement) WriteByte(c byte) error {
	return stmt.SQL.WriteByte(c)
}

// WriteQuoted writes a dialect-quoted identifier or structured column/table expression
// into SQL. It is not a data-value placeholder; use AddVar to bind values.
func (stmt *Statement) WriteQuoted(value interface{}) {
	stmt.QuoteTo(&stmt.SQL, value)
}

// QuoteTo writes identifiers and clause column/table references using the dialector. It
// resolves CurrentTable and PrimaryKey from model metadata and records missing-model
// errors. Raw flags bypass quoting, and clause.Expr is built as SQL; do not treat this
// as an arbitrary SQL sanitizer.
func (stmt *Statement) QuoteTo(writer clause.Writer, field interface{}) {
	write := func(raw bool, str string) {
		if raw {
			writer.WriteString(str)
		} else {
			stmt.DB.Dialector.QuoteTo(writer, str)
		}
	}

	switch v := field.(type) {
	case clause.Table:
		if v.Name == clause.CurrentTable {
			if stmt.TableExpr != nil {
				stmt.TableExpr.Build(stmt)
			} else if stmt.Table != "" {
				write(v.Raw, stmt.Table)
			} else if stmt.AddError(stmt.Parse(stmt.Model)) == nil {
				write(v.Raw, stmt.Table)
			}
		} else {
			write(v.Raw, v.Name)
		}

		if v.Alias != "" {
			writer.WriteByte(' ')
			write(v.Raw, v.Alias)
		}
	case clause.Column:
		if v.Table != "" {
			if v.Table == clause.CurrentTable {
				write(v.Raw, stmt.Table)
			} else {
				write(v.Raw, v.Table)
			}
			writer.WriteByte('.')
		}

		if v.Name == clause.PrimaryKey {
			if stmt.Schema == nil {
				stmt.DB.AddError(ErrModelValueRequired)
			} else if stmt.Schema.PrioritizedPrimaryField != nil {
				write(v.Raw, stmt.Schema.PrioritizedPrimaryField.DBName)
			} else if len(stmt.Schema.DBNames) > 0 {
				write(v.Raw, stmt.Schema.DBNames[0])
			} else {
				stmt.DB.AddError(ErrModelAccessibleFieldsRequired) //nolint:typecheck,errcheck
			}
		} else {
			write(v.Raw, v.Name)
		}

		if v.Alias != "" {
			writer.WriteString(" AS ")
			write(v.Raw, v.Alias)
		}
	case []clause.Column:
		writer.WriteByte('(')
		for idx, d := range v {
			if idx > 0 {
				writer.WriteByte(',')
			}
			stmt.QuoteTo(writer, d)
		}
		writer.WriteByte(')')
	case clause.Expr:
		v.Build(stmt)
	case string:
		stmt.DB.Dialector.QuoteTo(writer, v)
	case []string:
		writer.WriteByte('(')
		for idx, d := range v {
			if idx > 0 {
				writer.WriteByte(',')
			}
			stmt.DB.Dialector.QuoteTo(writer, d)
		}
		writer.WriteByte(')')
	default:
		stmt.DB.Dialector.QuoteTo(writer, fmt.Sprint(field))
	}
}

// Quote returns the quoted representation of an ordinary identifier using QuoteTo. It
// is not a pure string renderer for expressions: a clause.Expr, or a table resolved
// through Statement.TableExpr, builds into stmt.SQL and may add Vars/errors instead of
// writing into the local string builder. Use Quote for trusted identifiers and
// Build/AddVar for expression construction.
func (stmt *Statement) Quote(field interface{}) string {
	var builder strings.Builder
	stmt.QuoteTo(&builder, field)
	return builder.String()
}

// AddVar writes placeholders and appends data to Vars, using the dialector's binding
// syntax. Columns/tables are quoted, expressions and GORM Valuer values build SQL, and
// query builders become subqueries with rebound variables. Slices expand as lists
// (empty lists become (NULL)); byte slices remain one value. Expression SQL must be
// trusted even when passed in a value position.
func (stmt *Statement) AddVar(writer clause.Writer, vars ...interface{}) {
	for idx, v := range vars {
		if idx > 0 {
			writer.WriteByte(',')
		}

		switch v := v.(type) {
		case sql.NamedArg:
			stmt.Vars = append(stmt.Vars, v.Value)
		case clause.Column, clause.Table:
			stmt.QuoteTo(writer, v)
		case Valuer:
			reflectValue := reflect.ValueOf(v)
			if reflectValue.Kind() == reflect.Ptr && reflectValue.IsNil() {
				stmt.AddVar(writer, nil)
			} else {
				stmt.AddVar(writer, v.GormValue(stmt.Context, stmt.DB))
			}
		case clause.Interface:
			c := clause.Clause{Name: v.Name()}
			v.MergeClause(&c)
			c.Build(stmt)
		case clause.Expression:
			v.Build(stmt)
		case driver.Valuer:
			stmt.Vars = append(stmt.Vars, v)
			stmt.DB.Dialector.BindVarTo(writer, stmt, v)
		case []byte:
			stmt.Vars = append(stmt.Vars, v)
			stmt.DB.Dialector.BindVarTo(writer, stmt, v)
		case []interface{}:
			if len(v) > 0 {
				writer.WriteByte('(')
				stmt.AddVar(writer, v...)
				writer.WriteByte(')')
			} else {
				writer.WriteString("(NULL)")
			}
		case interface{ getInstance() *DB }:
			cv := v.getInstance()

			subdb := cv.Session(&Session{Logger: logger.Discard, DryRun: true}).getInstance()
			if cv.Statement.SQL.Len() > 0 {
				var (
					vars = subdb.Statement.Vars
					sql  = cv.Statement.SQL.String()
				)

				subdb.Statement.Vars = make([]interface{}, 0, len(vars))
				for _, vv := range vars {
					subdb.Statement.Vars = append(subdb.Statement.Vars, vv)
					bindvar := strings.Builder{}
					cv.BindVarTo(&bindvar, subdb.Statement, vv)
					sql = strings.Replace(sql, bindvar.String(), "?", 1)
				}

				subdb.Statement.SQL.Reset()
				subdb.Statement.Vars = stmt.Vars
				if strings.Contains(sql, "@") {
					clause.NamedExpr{SQL: sql, Vars: vars}.Build(subdb.Statement)
				} else {
					clause.Expr{SQL: sql, Vars: vars}.Build(subdb.Statement)
				}
			} else {
				subdb.Statement.Vars = append(stmt.Vars, subdb.Statement.Vars...)
				subdb.callbacks.Query().Execute(subdb)
			}

			writer.WriteString(subdb.Statement.SQL.String())
			stmt.Vars = subdb.Statement.Vars
		default:
			switch rv := reflect.ValueOf(v); rv.Kind() {
			case reflect.Slice, reflect.Array:
				if rv.Len() == 0 {
					writer.WriteString("(NULL)")
				} else if rv.Type().Elem() == reflect.TypeOf(uint8(0)) {
					stmt.Vars = append(stmt.Vars, v)
					stmt.DB.Dialector.BindVarTo(writer, stmt, v)
				} else {
					writer.WriteByte('(')
					for i := 0; i < rv.Len(); i++ {
						if i > 0 {
							writer.WriteByte(',')
						}
						stmt.AddVar(writer, rv.Index(i).Interface())
					}
					writer.WriteByte(')')
				}
			default:
				stmt.Vars = append(stmt.Vars, v)
				stmt.DB.Dialector.BindVarTo(writer, stmt, v)
			}
		}
	}
}

// AddClause invokes a StatementModifier immediately or merges the clause into Clauses
// by name using its MergeClause method. It records construction state without executing
// SQL; the clause implementation determines append/replace behavior.
func (stmt *Statement) AddClause(v clause.Interface) {
	if optimizer, ok := v.(StatementModifier); ok {
		optimizer.ModifyStatement(stmt)
	} else {
		name := v.Name()
		c := stmt.Clauses[name]
		c.Name = name
		v.MergeClause(&c)
		stmt.Clauses[name] = c
	}
}

// AddClauseIfNotExists adds v only when its named clause is absent or has no
// Expression. An existing non-nil expression is preserved; this is not an unconditional
// merge.
func (stmt *Statement) AddClauseIfNotExists(v clause.Interface) {
	if c, ok := stmt.Clauses[v.Name()]; !ok || c.Expression == nil {
		stmt.AddClause(v)
	}
}

// BuildCondition converts SQL templates, maps, structs, expressions, nested DB
// predicates, or primary-key inputs into clause expressions. Struct zero values are
// omitted unless fields are explicitly selected; maps retain zero values. String-keyed
// map columns are sorted for deterministic SQL. String inputs can be interpreted as
// SQL, so untrusted identifiers/values must not be passed as condition text. Errors are
// added to the statement DB.
func (stmt *Statement) BuildCondition(query interface{}, args ...interface{}) []clause.Expression {
	if s, ok := query.(string); ok {
		// if it is a number, then treats it as primary key
		if _, err := strconv.Atoi(s); err != nil {
			if s == "" && len(args) == 0 {
				return nil
			}

			if len(args) == 0 || (len(args) > 0 && strings.Contains(s, "?")) {
				// looks like a where condition
				return []clause.Expression{clause.Expr{SQL: s, Vars: args}}
			}

			if len(args) > 0 && strings.Contains(s, "@") {
				// looks like a named query
				return []clause.Expression{clause.NamedExpr{SQL: s, Vars: args}}
			}

			if strings.Contains(strings.TrimSpace(s), " ") {
				// looks like a where condition
				return []clause.Expression{clause.Expr{SQL: s, Vars: args}}
			}

			if len(args) == 1 {
				return []clause.Expression{clause.Eq{Column: s, Value: args[0]}}
			}
		}
	}

	conds := make([]clause.Expression, 0, 4)
	args = append([]interface{}{query}, args...)
	for idx, arg := range args {
		if arg == nil {
			continue
		}
		if valuer, ok := arg.(driver.Valuer); ok {
			arg, _ = valuer.Value()
		}

		curTable := stmt.Table
		if curTable == "" {
			curTable = clause.CurrentTable
		}

		switch v := arg.(type) {
		case clause.Expression:
			conds = append(conds, v)
		case []clause.Expression:
			conds = append(conds, v...)
		case *DB:
			v.executeScopes()

			if cs, ok := v.Statement.Clauses["WHERE"]; ok {
				if where, ok := cs.Expression.(clause.Where); ok {
					if len(where.Exprs) == 1 {
						if orConds, ok := where.Exprs[0].(clause.OrConditions); ok {
							if len(orConds.Exprs) == 1 {
								where.Exprs[0] = clause.AndConditions(orConds)
							}
						}
					}
					conds = append(conds, clause.And(where.Exprs...))
				} else if cs.Expression != nil {
					conds = append(conds, cs.Expression)
				}
			}
		case map[interface{}]interface{}:
			for i, j := range v {
				conds = append(conds, clause.Eq{Column: i, Value: j})
			}
		case map[string]string:
			keys := make([]string, 0, len(v))
			for i := range v {
				keys = append(keys, i)
			}
			sort.Strings(keys)

			for _, key := range keys {
				column := clause.Column{Name: key, Table: curTable}
				if strings.Contains(key, ".") {
					column = clause.Column{Name: key}
				}
				conds = append(conds, clause.Eq{Column: column, Value: v[key]})
			}
		case map[string]interface{}:
			keys := make([]string, 0, len(v))
			for i := range v {
				keys = append(keys, i)
			}
			sort.Strings(keys)

			for _, key := range keys {
				reflectValue := reflect.Indirect(reflect.ValueOf(v[key]))
				column := clause.Column{Name: key, Table: curTable}
				if strings.Contains(key, ".") {
					column = clause.Column{Name: key}
				}
				switch reflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					if _, ok := v[key].(driver.Valuer); ok {
						conds = append(conds, clause.Eq{Column: column, Value: v[key]})
					} else if _, ok := v[key].(Valuer); ok {
						conds = append(conds, clause.Eq{Column: column, Value: v[key]})
					} else {
						// optimize reflect value length
						valueLen := reflectValue.Len()
						values := make([]interface{}, valueLen)
						for i := 0; i < valueLen; i++ {
							values[i] = reflectValue.Index(i).Interface()
						}

						conds = append(conds, clause.IN{Column: column, Values: values})
					}
				default:
					conds = append(conds, clause.Eq{Column: column, Value: v[key]})
				}
			}
		default:
			reflectValue := reflect.Indirect(reflect.ValueOf(arg))
			for reflectValue.Kind() == reflect.Ptr {
				reflectValue = reflectValue.Elem()
			}

			if s, err := schema.Parse(arg, stmt.DB.cacheStore, stmt.DB.NamingStrategy); err == nil {
				selectedColumns := map[string]bool{}
				if idx == 0 {
					for _, v := range args[1:] {
						if vs, ok := v.(string); ok {
							selectedColumns[vs] = true
						}
					}
				}
				restricted := len(selectedColumns) != 0

				switch reflectValue.Kind() {
				case reflect.Struct:
					for _, field := range s.Fields {
						selected := selectedColumns[field.DBName] || selectedColumns[field.Name]
						if selected || (!restricted && field.Readable) {
							if v, isZero := field.ValueOf(stmt.Context, reflectValue); !isZero || selected {
								if field.DBName != "" {
									conds = append(conds, clause.Eq{Column: clause.Column{Table: curTable, Name: field.DBName}, Value: v})
								} else if field.DataType != "" {
									conds = append(conds, clause.Eq{Column: clause.Column{Table: curTable, Name: field.Name}, Value: v})
								}
							}
						}
					}
				case reflect.Slice, reflect.Array:
					for i := 0; i < reflectValue.Len(); i++ {
						for _, field := range s.Fields {
							selected := selectedColumns[field.DBName] || selectedColumns[field.Name]
							if selected || (!restricted && field.Readable) {
								if v, isZero := field.ValueOf(stmt.Context, reflectValue.Index(i)); !isZero || selected {
									if field.DBName != "" {
										conds = append(conds, clause.Eq{Column: clause.Column{Table: curTable, Name: field.DBName}, Value: v})
									} else if field.DataType != "" {
										conds = append(conds, clause.Eq{Column: clause.Column{Table: curTable, Name: field.Name}, Value: v})
									}
								}
							}
						}
					}
				}

				if restricted {
					break
				}
			} else if !reflectValue.IsValid() {
				stmt.AddError(ErrInvalidData)
			} else if len(conds) == 0 {
				if len(args) == 1 {
					switch reflectValue.Kind() {
					case reflect.Slice, reflect.Array:
						// optimize reflect value length
						valueLen := reflectValue.Len()
						values := make([]interface{}, valueLen)
						for i := 0; i < valueLen; i++ {
							values[i] = reflectValue.Index(i).Interface()
						}

						if len(values) > 0 {
							conds = append(conds, clause.IN{Column: clause.Column{Table: curTable, Name: clause.PrimaryKey}, Values: values})
							return []clause.Expression{clause.And(conds...)}
						}
						return nil
					}
				}

				conds = append(conds, clause.IN{Column: clause.Column{Table: curTable, Name: clause.PrimaryKey}, Values: args})
			}
		}
	}

	if len(conds) > 0 {
		return []clause.Expression{clause.And(conds...)}
	}
	return nil
}

// Build appends present clauses in the requested order, using Config.ClauseBuilders
// overrides when registered. It does not reset SQL or Vars; callers must start with the
// appropriate buffer state. Missing clause names are skipped.
func (stmt *Statement) Build(clauses ...string) {
	var firstClauseWritten bool

	for _, name := range clauses {
		if c, ok := stmt.Clauses[name]; ok {
			if firstClauseWritten {
				stmt.WriteByte(' ')
			}

			firstClauseWritten = true
			if b, ok := stmt.DB.ClauseBuilders[name]; ok {
				b(c, stmt)
			} else {
				c.Build(stmt)
			}
		}
	}
}

// Parse resolves and caches schema metadata for value and fills the statement table
// when it is unset. It is shorthand for ParseWithSpecialTableName with no table
// override; it does not inspect or migrate the database.
func (stmt *Statement) Parse(value interface{}) (err error) {
	return stmt.ParseWithSpecialTableName(value, "")
}

// ParseWithSpecialTableName parses value using the shared schema cache and naming
// strategy, optionally overriding its table name. It sets Schema and fills Table only
// if the current table is empty, handling schema-qualified tables with a quoted
// TableExpr. It returns parsing errors without executing SQL.
func (stmt *Statement) ParseWithSpecialTableName(value interface{}, specialTableName string) (err error) {
	if stmt.Schema, err = schema.ParseWithSpecialTableName(value, stmt.DB.cacheStore, stmt.DB.NamingStrategy, specialTableName); err == nil && stmt.Table == "" {
		if tables := strings.Split(stmt.Schema.Table, "."); len(tables) == 2 {
			stmt.TableExpr = &clause.Expr{SQL: stmt.Quote(stmt.Schema.Table)}
			stmt.Table = tables[1]
			return
		}

		stmt.Table = stmt.Schema.Table
	}
	return err
}

// clone copies statement containers needed for session isolation, including clauses,
// preloads, joins, queued scopes, settings, and built SQL/Vars. Model, destination,
// schema, context, connection, result collector, and nested reference values remain
// shared. The caller must attach the new DB pointer; this is not a general deep copy.
func (stmt *Statement) clone() *Statement {
	newStmt := &Statement{
		TableExpr:            stmt.TableExpr,
		Table:                stmt.Table,
		Model:                stmt.Model,
		Unscoped:             stmt.Unscoped,
		Dest:                 stmt.Dest,
		ReflectValue:         stmt.ReflectValue,
		Clauses:              map[string]clause.Clause{},
		Distinct:             stmt.Distinct,
		Selects:              stmt.Selects,
		Omits:                stmt.Omits,
		ColumnMapping:        stmt.ColumnMapping,
		Preloads:             map[string][]interface{}{},
		ConnPool:             stmt.ConnPool,
		Schema:               stmt.Schema,
		Context:              stmt.Context,
		RaiseErrorOnNotFound: stmt.RaiseErrorOnNotFound,
		SkipHooks:            stmt.SkipHooks,
		Result:               stmt.Result,
	}

	if stmt.SQL.Len() > 0 {
		newStmt.SQL.WriteString(stmt.SQL.String())
		newStmt.Vars = make([]interface{}, 0, len(stmt.Vars))
		newStmt.Vars = append(newStmt.Vars, stmt.Vars...)
	}

	for k, c := range stmt.Clauses {
		newStmt.Clauses[k] = c
	}

	for k, p := range stmt.Preloads {
		newStmt.Preloads[k] = p
	}

	if len(stmt.Joins) > 0 {
		newStmt.Joins = make([]join, len(stmt.Joins))
		copy(newStmt.Joins, stmt.Joins)
	}

	if len(stmt.scopes) > 0 {
		newStmt.scopes = make([]func(*DB) *DB, len(stmt.scopes))
		copy(newStmt.scopes, stmt.scopes)
	}

	stmt.Settings.Range(func(k, v interface{}) bool {
		newStmt.Settings.Store(k, v)
		return true
	})

	return newStmt
}

// SetColumn changes a destination field during hooks or callbacks through schema-aware
// assignment. Map destinations are updated directly. For slice models a hook updates
// CurDestIndex; supplying the optional fromCallbacks argument applies to every element
// (its presence, not its boolean value, controls this). Assignment failures are added
// to DB.Error. It changes pending values, not the database by itself.
func (stmt *Statement) SetColumn(name string, value interface{}, fromCallbacks ...bool) {
	if v, ok := stmt.Dest.(map[string]interface{}); ok {
		v[name] = value
	} else if v, ok := stmt.Dest.([]map[string]interface{}); ok {
		for _, m := range v {
			m[name] = value
		}
	} else if stmt.Schema != nil {
		if field := stmt.Schema.LookUpField(name); field != nil {
			destValue := reflect.ValueOf(stmt.Dest)
			for destValue.Kind() == reflect.Ptr {
				destValue = destValue.Elem()
			}

			if stmt.ReflectValue != destValue {
				if !destValue.CanAddr() {
					destValueCanAddr := reflect.New(destValue.Type())
					destValueCanAddr.Elem().Set(destValue)
					stmt.Dest = destValueCanAddr.Interface()
					destValue = destValueCanAddr.Elem()
				}

				switch destValue.Kind() {
				case reflect.Struct:
					stmt.AddError(field.Set(stmt.Context, destValue, value))
				default:
					stmt.AddError(ErrInvalidData)
				}
			}

			switch stmt.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				if len(fromCallbacks) > 0 {
					for i := 0; i < stmt.ReflectValue.Len(); i++ {
						stmt.AddError(field.Set(stmt.Context, stmt.ReflectValue.Index(i), value))
					}
				} else {
					stmt.AddError(field.Set(stmt.Context, stmt.ReflectValue.Index(stmt.CurDestIndex), value))
				}
			case reflect.Struct:
				if !stmt.ReflectValue.CanAddr() {
					stmt.AddError(ErrInvalidValue)
					return
				}

				stmt.AddError(field.Set(stmt.Context, stmt.ReflectValue, value))
			}
		} else {
			stmt.AddError(ErrInvalidField)
		}
	} else {
		stmt.AddError(ErrInvalidField)
	}
}

// Changed reports whether any named update field differs from the current model value,
// respecting Select/Omit and field permissions. With no names it checks all mapped
// fields. It compares pending Update/Updates values in memory, not current database
// contents; struct zero values are omitted unless selected. Use it in update hooks with
// an initialized schema, not as general dirty tracking for Save.
func (stmt *Statement) Changed(fields ...string) bool {
	modelValue := stmt.ReflectValue
	switch modelValue.Kind() {
	case reflect.Slice, reflect.Array:
		modelValue = stmt.ReflectValue.Index(stmt.CurDestIndex)
	}

	selectColumns, restricted := stmt.SelectAndOmitColumns(false, true)
	changed := func(field *schema.Field) bool {
		fieldValue, _ := field.ValueOf(stmt.Context, modelValue)
		if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
			if mv, mok := stmt.Dest.(map[string]interface{}); mok {
				if fv, ok := mv[field.Name]; ok {
					return !utils.AssertEqual(fv, fieldValue)
				} else if fv, ok := mv[field.DBName]; ok {
					return !utils.AssertEqual(fv, fieldValue)
				}
			} else {
				destValue := reflect.ValueOf(stmt.Dest)
				for destValue.Kind() == reflect.Ptr {
					destValue = destValue.Elem()
				}
				if descSchema, err := schema.Parse(stmt.Dest, stmt.DB.cacheStore, stmt.DB.NamingStrategy); err == nil {
					if destField := descSchema.LookUpField(field.DBName); destField != nil {
						changedValue, zero := destField.ValueOf(stmt.Context, destValue)
						if v {
							return !utils.AssertEqual(changedValue, fieldValue)
						}
						return !zero && !utils.AssertEqual(changedValue, fieldValue)
					}
				}
			}
		}
		return false
	}

	if len(fields) == 0 {
		for _, field := range stmt.Schema.FieldsByDBName {
			if changed(field) {
				return true
			}
		}
	} else {
		for _, name := range fields {
			if field := stmt.Schema.LookUpField(name); field != nil {
				if changed(field) {
					return true
				}
			}
		}
	}

	return false
}

// matchName recognizes a simple optionally qualified column or wildcard for selection
// mapping. It returns empty strings when the expression does not match; it is not a
// full SQL parser or validation boundary.
var matchName = func() func(tableColumn string) (table, column string) {
	nameMatcher := regexp.MustCompile(`^(?:\W?(\w+?)\W?\.)?(?:(\*)|\W?(\w+?)\W?)$`)
	return func(tableColumn string) (table, column string) {
		if matches := nameMatcher.FindStringSubmatch(tableColumn); len(matches) == 4 {
			table = matches[1]
			star := matches[2]
			columnName := matches[3]
			if star != "" {
				return table, star
			}
			return table, columnName
		}
		return "", ""
	}
}()

// SelectAndOmitColumns resolves selected/omitted names, wildcards, and association
// names into a database-column map (true means included). Omit overrides Select;
// requireCreate/requireUpdate also exclude fields lacking those permissions. The second
// result reports an explicit restricted selection, so absence from the map must be
// interpreted with that flag.
func (stmt *Statement) SelectAndOmitColumns(requireCreate, requireUpdate bool) (map[string]bool, bool) {
	results := map[string]bool{}
	notRestricted := false

	processColumn := func(column string, result bool) {
		if stmt.Schema == nil {
			results[column] = result
		} else if column == "*" {
			notRestricted = result
			for _, dbName := range stmt.Schema.DBNames {
				results[dbName] = result
			}
		} else if column == clause.Associations {
			for _, rel := range stmt.Schema.Relationships.Relations {
				results[rel.Name] = result
			}
		} else if field := stmt.Schema.LookUpField(column); field != nil && field.DBName != "" {
			results[field.DBName] = result
		} else if table, col := matchName(column); col != "" && (table == stmt.Table || table == "") {
			if col == "*" {
				for _, dbName := range stmt.Schema.DBNames {
					results[dbName] = result
				}
			} else {
				results[col] = result
			}
		} else {
			results[column] = result
		}
	}

	// select columns
	for _, column := range stmt.Selects {
		processColumn(column, true)
	}

	// omit columns
	for _, column := range stmt.Omits {
		processColumn(column, false)
	}

	if stmt.Schema != nil {
		for _, field := range stmt.Schema.FieldsByName {
			name := field.DBName
			if name == "" {
				name = field.Name
			}

			if requireCreate && !field.Creatable {
				results[name] = false
			} else if requireUpdate && !field.Updatable {
				results[name] = false
			}
		}
	}

	return results, !notRestricted && len(stmt.Selects) > 0
}
