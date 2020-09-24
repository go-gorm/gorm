package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// Statement statement
type Statement struct {
	*DB
	TableExpr            *clause.Expr
	Table                string
	Model                interface{}
	Unscoped             bool
	Dest                 interface{}
	ReflectValue         reflect.Value
	Clauses              map[string]clause.Clause
	Distinct             bool
	Selects              []string // selected columns
	Omits                []string // omit columns
	Joins                []join
	Preloads             map[string][]interface{}
	Settings             sync.Map
	ConnPool             ConnPool
	Schema               *schema.Schema
	Context              context.Context
	RaiseErrorOnNotFound bool
	UpdatingColumn       bool
	SQL                  strings.Builder
	Vars                 []interface{}
	CurDestIndex         int
	attrs                []interface{}
	assigns              []interface{}
}

type join struct {
	Name  string
	Conds []interface{}
}

// StatementModifier statement modifier interface
type StatementModifier interface {
	ModifyStatement(*Statement)
}

// Write write string
func (stmt *Statement) WriteString(str string) (int, error) {
	return stmt.SQL.WriteString(str)
}

// Write write string
func (stmt *Statement) WriteByte(c byte) error {
	return stmt.SQL.WriteByte(c)
}

// WriteQuoted write quoted value
func (stmt *Statement) WriteQuoted(value interface{}) {
	stmt.QuoteTo(&stmt.SQL, value)
}

// QuoteTo write quoted value to writer
func (stmt *Statement) QuoteTo(writer clause.Writer, field interface{}) {
	switch v := field.(type) {
	case clause.Table:
		if v.Name == clause.CurrentTable {
			if stmt.TableExpr != nil {
				stmt.TableExpr.Build(stmt)
			} else {
				stmt.DB.Dialector.QuoteTo(writer, stmt.Table)
			}
		} else if v.Raw {
			writer.WriteString(v.Name)
		} else {
			stmt.DB.Dialector.QuoteTo(writer, v.Name)
		}

		if v.Alias != "" {
			writer.WriteByte(' ')
			stmt.DB.Dialector.QuoteTo(writer, v.Alias)
		}
	case clause.Column:
		if v.Table != "" {
			if v.Table == clause.CurrentTable {
				stmt.DB.Dialector.QuoteTo(writer, stmt.Table)
			} else {
				stmt.DB.Dialector.QuoteTo(writer, v.Table)
			}
			writer.WriteByte('.')
		}

		if v.Name == clause.PrimaryKey {
			if stmt.Schema == nil {
				stmt.DB.AddError(ErrModelValueRequired)
			} else if stmt.Schema.PrioritizedPrimaryField != nil {
				stmt.DB.Dialector.QuoteTo(writer, stmt.Schema.PrioritizedPrimaryField.DBName)
			} else if len(stmt.Schema.DBNames) > 0 {
				stmt.DB.Dialector.QuoteTo(writer, stmt.Schema.DBNames[0])
			}
		} else if v.Raw {
			writer.WriteString(v.Name)
		} else {
			stmt.DB.Dialector.QuoteTo(writer, v.Name)
		}

		if v.Alias != "" {
			writer.WriteString(" AS ")
			stmt.DB.Dialector.QuoteTo(writer, v.Alias)
		}
	case []clause.Column:
		writer.WriteByte('(')
		for idx, d := range v {
			if idx > 0 {
				writer.WriteString(",")
			}
			stmt.QuoteTo(writer, d)
		}
		writer.WriteByte(')')
	case string:
		stmt.DB.Dialector.QuoteTo(writer, v)
	case []string:
		writer.WriteByte('(')
		for idx, d := range v {
			if idx > 0 {
				writer.WriteString(",")
			}
			stmt.DB.Dialector.QuoteTo(writer, d)
		}
		writer.WriteByte(')')
	default:
		stmt.DB.Dialector.QuoteTo(writer, fmt.Sprint(field))
	}
}

// Quote returns quoted value
func (stmt *Statement) Quote(field interface{}) string {
	var builder strings.Builder
	stmt.QuoteTo(&builder, field)
	return builder.String()
}

// Write write string
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
			stmt.AddVar(writer, v.GormValue(stmt.Context, stmt.DB))
		case clause.Expr:
			var varStr strings.Builder
			var sql = v.SQL
			for _, arg := range v.Vars {
				stmt.Vars = append(stmt.Vars, arg)
				stmt.DB.Dialector.BindVarTo(&varStr, stmt, arg)
				sql = strings.Replace(sql, "?", varStr.String(), 1)
				varStr.Reset()
			}

			writer.WriteString(sql)
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
		case *DB:
			subdb := v.Session(&Session{Logger: logger.Discard, DryRun: true, WithConditions: true}).getInstance()
			subdb.Statement.Vars = append(subdb.Statement.Vars, stmt.Vars...)
			subdb.callbacks.Query().Execute(subdb)
			writer.WriteString(subdb.Statement.SQL.String())
			stmt.Vars = subdb.Statement.Vars
		default:
			switch rv := reflect.ValueOf(v); rv.Kind() {
			case reflect.Slice, reflect.Array:
				if rv.Len() == 0 {
					writer.WriteString("(NULL)")
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

// AddClause add clause
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

// AddClauseIfNotExists add clause if not exists
func (stmt *Statement) AddClauseIfNotExists(v clause.Interface) {
	if c, ok := stmt.Clauses[v.Name()]; !ok || c.Expression == nil {
		stmt.AddClause(v)
	}
}

// BuildCondition build condition
func (stmt *Statement) BuildCondition(query interface{}, args ...interface{}) (conds []clause.Expression) {
	if s, ok := query.(string); ok {
		// if it is a number, then treats it as primary key
		if _, err := strconv.Atoi(s); err != nil {
			if s == "" && len(args) == 0 {
				return
			} else if len(args) == 0 || (len(args) > 0 && strings.Contains(s, "?")) {
				// looks like a where condition
				return []clause.Expression{clause.Expr{SQL: s, Vars: args}}
			} else if len(args) > 0 && strings.Contains(s, "@") {
				// looks like a named query
				return []clause.Expression{clause.NamedExpr{SQL: s, Vars: args}}
			} else if len(args) == 1 {
				return []clause.Expression{clause.Eq{Column: s, Value: args[0]}}
			}
		}
	}

	args = append([]interface{}{query}, args...)
	for _, arg := range args {
		if valuer, ok := arg.(driver.Valuer); ok {
			arg, _ = valuer.Value()
		}

		switch v := arg.(type) {
		case clause.Expression:
			conds = append(conds, v)
		case *DB:
			if cs, ok := v.Statement.Clauses["WHERE"]; ok {
				if where, ok := cs.Expression.(clause.Where); ok {
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
			var keys = make([]string, 0, len(v))
			for i := range v {
				keys = append(keys, i)
			}
			sort.Strings(keys)

			for _, key := range keys {
				conds = append(conds, clause.Eq{Column: key, Value: v[key]})
			}
		case map[string]interface{}:
			var keys = make([]string, 0, len(v))
			for i := range v {
				keys = append(keys, i)
			}
			sort.Strings(keys)

			for _, key := range keys {
				reflectValue := reflect.Indirect(reflect.ValueOf(v[key]))
				switch reflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					if _, ok := v[key].(driver.Valuer); ok {
						conds = append(conds, clause.Eq{Column: key, Value: v[key]})
					} else if _, ok := v[key].(Valuer); ok {
						conds = append(conds, clause.Eq{Column: key, Value: v[key]})
					} else {
						values := make([]interface{}, reflectValue.Len())
						for i := 0; i < reflectValue.Len(); i++ {
							values[i] = reflectValue.Index(i).Interface()
						}

						conds = append(conds, clause.IN{Column: key, Values: values})
					}
				default:
					conds = append(conds, clause.Eq{Column: key, Value: v[key]})
				}
			}
		default:
			reflectValue := reflect.Indirect(reflect.ValueOf(arg))
			if s, err := schema.Parse(arg, stmt.DB.cacheStore, stmt.DB.NamingStrategy); err == nil {
				switch reflectValue.Kind() {
				case reflect.Struct:
					for _, field := range s.Fields {
						if field.Readable {
							if v, isZero := field.ValueOf(reflectValue); !isZero {
								if field.DBName != "" {
									conds = append(conds, clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: field.DBName}, Value: v})
								} else if field.DataType != "" {
									conds = append(conds, clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: field.Name}, Value: v})
								}
							}
						}
					}
				case reflect.Slice, reflect.Array:
					for i := 0; i < reflectValue.Len(); i++ {
						for _, field := range s.Fields {
							if field.Readable {
								if v, isZero := field.ValueOf(reflectValue.Index(i)); !isZero {
									if field.DBName != "" {
										conds = append(conds, clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: field.DBName}, Value: v})
									} else if field.DataType != "" {
										conds = append(conds, clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: field.Name}, Value: v})
									}
								}
							}
						}
					}
				}
			} else if len(conds) == 0 {
				if len(args) == 1 {
					switch reflectValue.Kind() {
					case reflect.Slice, reflect.Array:
						values := make([]interface{}, reflectValue.Len())
						for i := 0; i < reflectValue.Len(); i++ {
							values[i] = reflectValue.Index(i).Interface()
						}

						if len(values) > 0 {
							conds = append(conds, clause.IN{Column: clause.PrimaryColumn, Values: values})
						}
						return
					}
				}

				conds = append(conds, clause.IN{Column: clause.PrimaryColumn, Values: args})
			}
		}
	}

	return
}

// Build build sql with clauses names
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

func (stmt *Statement) Parse(value interface{}) (err error) {
	if stmt.Schema, err = schema.Parse(value, stmt.DB.cacheStore, stmt.DB.NamingStrategy); err == nil && stmt.Table == "" {
		if tables := strings.Split(stmt.Schema.Table, "."); len(tables) == 2 {
			stmt.TableExpr = &clause.Expr{SQL: stmt.Quote(stmt.Schema.Table)}
			stmt.Table = tables[1]
			return
		}

		stmt.Table = stmt.Schema.Table
	}
	return err
}

func (stmt *Statement) clone() *Statement {
	newStmt := &Statement{
		TableExpr:            stmt.TableExpr,
		Table:                stmt.Table,
		Model:                stmt.Model,
		Dest:                 stmt.Dest,
		ReflectValue:         stmt.ReflectValue,
		Clauses:              map[string]clause.Clause{},
		Distinct:             stmt.Distinct,
		Selects:              stmt.Selects,
		Omits:                stmt.Omits,
		Preloads:             map[string][]interface{}{},
		ConnPool:             stmt.ConnPool,
		Schema:               stmt.Schema,
		Context:              stmt.Context,
		RaiseErrorOnNotFound: stmt.RaiseErrorOnNotFound,
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

	stmt.Settings.Range(func(k, v interface{}) bool {
		newStmt.Settings.Store(k, v)
		return true
	})

	return newStmt
}

// Helpers
// SetColumn set column's value
func (stmt *Statement) SetColumn(name string, value interface{}) {
	if v, ok := stmt.Dest.(map[string]interface{}); ok {
		v[name] = value
	} else if stmt.Schema != nil {
		if field := stmt.Schema.LookUpField(name); field != nil {
			switch stmt.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				field.Set(stmt.ReflectValue.Index(stmt.CurDestIndex), value)
			case reflect.Struct:
				field.Set(stmt.ReflectValue, value)
			}
		} else {
			stmt.AddError(ErrInvalidField)
		}
	} else {
		stmt.AddError(ErrInvalidField)
	}
}

// Changed check model changed or not when updating
func (stmt *Statement) Changed(fields ...string) bool {
	modelValue := reflect.ValueOf(stmt.Model)
	for modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}

	switch modelValue.Kind() {
	case reflect.Slice, reflect.Array:
		modelValue = stmt.ReflectValue.Index(stmt.CurDestIndex)
	}

	selectColumns, restricted := stmt.SelectAndOmitColumns(false, true)
	changed := func(field *schema.Field) bool {
		fieldValue, _ := field.ValueOf(modelValue)
		if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
			if v, ok := stmt.Dest.(map[string]interface{}); ok {
				if fv, ok := v[field.Name]; ok {
					return !utils.AssertEqual(fv, fieldValue)
				} else if fv, ok := v[field.DBName]; ok {
					return !utils.AssertEqual(fv, fieldValue)
				}
			} else {
				changedValue, _ := field.ValueOf(stmt.ReflectValue)
				return !utils.AssertEqual(changedValue, fieldValue)
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

// SelectAndOmitColumns get select and omit columns, select -> true, omit -> false
func (stmt *Statement) SelectAndOmitColumns(requireCreate, requireUpdate bool) (map[string]bool, bool) {
	results := map[string]bool{}
	notRestricted := false

	// select columns
	for _, column := range stmt.Selects {
		if column == "*" {
			notRestricted = true
			for _, dbName := range stmt.Schema.DBNames {
				results[dbName] = true
			}
		} else if column == clause.Associations && stmt.Schema != nil {
			for _, rel := range stmt.Schema.Relationships.Relations {
				results[rel.Name] = true
			}
		} else if field := stmt.Schema.LookUpField(column); field != nil && field.DBName != "" {
			results[field.DBName] = true
		} else {
			results[column] = true
		}
	}

	// omit columns
	for _, omit := range stmt.Omits {
		if omit == clause.Associations {
			if stmt.Schema != nil {
				for _, rel := range stmt.Schema.Relationships.Relations {
					results[rel.Name] = false
				}
			}
		} else if field := stmt.Schema.LookUpField(omit); field != nil && field.DBName != "" {
			results[field.DBName] = false
		} else {
			results[omit] = false
		}
	}

	if stmt.Schema != nil {
		for _, field := range stmt.Schema.Fields {
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
