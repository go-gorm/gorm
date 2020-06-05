package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Statement statement
type Statement struct {
	*DB
	Table                string
	Model                interface{}
	Unscoped             bool
	Dest                 interface{}
	ReflectValue         reflect.Value
	Clauses              map[string]clause.Clause
	Distinct             bool
	Selects              []string // selected columns
	Omits                []string // omit columns
	Joins                map[string][]interface{}
	Preloads             map[string][]interface{}
	Settings             sync.Map
	ConnPool             ConnPool
	Schema               *schema.Schema
	Context              context.Context
	RaiseErrorOnNotFound bool
	UpdatingColumn       bool
	SQL                  strings.Builder
	Vars                 []interface{}
	NamedVars            []sql.NamedArg
	attrs                []interface{}
	assigns              []interface{}
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
func (stmt *Statement) WriteQuoted(value interface{}) error {
	stmt.QuoteTo(&stmt.SQL, value)
	return nil
}

// QuoteTo write quoted value to writer
func (stmt Statement) QuoteTo(writer clause.Writer, field interface{}) {
	switch v := field.(type) {
	case clause.Table:
		if v.Name == clause.CurrentTable {
			stmt.DB.Dialector.QuoteTo(writer, stmt.Table)
		} else if v.Raw {
			writer.WriteString(v.Name)
		} else {
			stmt.DB.Dialector.QuoteTo(writer, v.Name)
		}

		if v.Alias != "" {
			writer.WriteString(" AS ")
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
			if stmt.Schema != nil && stmt.Schema.PrioritizedPrimaryField != nil {
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
	case string:
		stmt.DB.Dialector.QuoteTo(writer, v)
	case []string:
		writer.WriteByte('(')
		for idx, d := range v {
			if idx != 0 {
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
func (stmt Statement) Quote(field interface{}) string {
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
			if len(v.Name) > 0 {
				stmt.NamedVars = append(stmt.NamedVars, v)
				writer.WriteByte('@')
				writer.WriteString(v.Name)
			} else {
				stmt.Vars = append(stmt.Vars, v.Value)
				stmt.DB.Dialector.BindVarTo(writer, stmt, v.Value)
			}
		case clause.Column, clause.Table:
			stmt.QuoteTo(writer, v)
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
		case []interface{}:
			if len(v) > 0 {
				writer.WriteByte('(')
				stmt.AddVar(writer, v...)
				writer.WriteByte(')')
			} else {
				writer.WriteString("(NULL)")
			}
		case *DB:
			subdb := v.Session(&Session{DryRun: true, WithConditions: true}).getInstance()
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
	}

	c, ok := stmt.Clauses[v.Name()]
	if !ok {
		c.Name = v.Name()
	}
	v.MergeClause(&c)
	stmt.Clauses[v.Name()] = c
}

// AddClauseIfNotExists add clause if not exists
func (stmt *Statement) AddClauseIfNotExists(v clause.Interface) {
	if c, ok := stmt.Clauses[v.Name()]; !ok && c.Expression == nil {
		stmt.AddClause(v)
	}
}

// BuildCondtion build condition
func (stmt Statement) BuildCondtion(query interface{}, args ...interface{}) (conds []clause.Expression) {
	if sql, ok := query.(string); ok {
		// if it is a number, then treats it as primary key
		if _, err := strconv.Atoi(sql); err != nil {
			if sql == "" && len(args) == 0 {
				return
			} else if len(args) == 0 || (len(args) > 0 && strings.Contains(sql, "?")) || strings.Contains(sql, "@") {
				return []clause.Expression{clause.Expr{SQL: sql, Vars: args}}
			} else if len(args) == 1 {
				return []clause.Expression{clause.Eq{Column: sql, Value: args[0]}}
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
			if v.Statement != nil {
				if cs, ok := v.Statement.Clauses["WHERE"]; ok {
					conds = append(conds, cs.Expression)
				}
			}
		case map[interface{}]interface{}:
			for i, j := range v {
				conds = append(conds, clause.Eq{Column: i, Value: j})
			}
		case map[string]string:
			for i, j := range v {
				conds = append(conds, clause.Eq{Column: i, Value: j})
			}
		case map[string]interface{}:
			for i, j := range v {
				conds = append(conds, clause.Eq{Column: i, Value: j})
			}
		default:
			reflectValue := reflect.Indirect(reflect.ValueOf(arg))
			if s, err := schema.Parse(arg, stmt.DB.cacheStore, stmt.DB.NamingStrategy); err == nil {
				switch reflectValue.Kind() {
				case reflect.Struct:
					for _, field := range s.Fields {
						if v, isZero := field.ValueOf(reflectValue); !isZero {
							if field.DBName == "" {
								conds = append(conds, clause.Eq{Column: clause.Column{Table: s.Table, Name: field.Name}, Value: v})
							} else {
								conds = append(conds, clause.Eq{Column: clause.Column{Table: s.Table, Name: field.DBName}, Value: v})
							}
						}
					}
				case reflect.Slice, reflect.Array:
					for i := 0; i < reflectValue.Len(); i++ {
						for _, field := range s.Fields {
							if v, isZero := field.ValueOf(reflectValue.Index(i)); !isZero {
								if field.DBName == "" {
									conds = append(conds, clause.Eq{Column: clause.Column{Table: s.Table, Name: field.Name}, Value: v})
								} else {
									conds = append(conds, clause.Eq{Column: clause.Column{Table: s.Table, Name: field.DBName}, Value: v})
								}
							}
						}
					}
				}
			} else if len(conds) == 0 {
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
	// TODO handle named vars
}

func (stmt *Statement) Parse(value interface{}) (err error) {
	if stmt.Schema, err = schema.Parse(value, stmt.DB.cacheStore, stmt.DB.NamingStrategy); err == nil && stmt.Table == "" {
		stmt.Table = stmt.Schema.Table
	}
	return err
}

func (stmt *Statement) clone() *Statement {
	newStmt := &Statement{
		DB:                   stmt.DB,
		Table:                stmt.Table,
		Model:                stmt.Model,
		Dest:                 stmt.Dest,
		ReflectValue:         stmt.ReflectValue,
		Clauses:              map[string]clause.Clause{},
		Distinct:             stmt.Distinct,
		Selects:              stmt.Selects,
		Omits:                stmt.Omits,
		Joins:                map[string][]interface{}{},
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

	for k, j := range stmt.Joins {
		newStmt.Joins[k] = j
	}

	return newStmt
}

func (stmt *Statement) reinit() {
	// stmt.Table = ""
	// stmt.Model = nil
	// stmt.Selects = nil
	// stmt.Omits = nil
	// stmt.ConnPool = stmt.DB.Config.ConnPool
	// stmt.Context = context.Background()
	// stmt.RaiseErrorOnNotFound = false

	// for k := range stmt.Clauses {
	// 	delete(stmt.Clauses, k)
	// }

	// for k := range stmt.Joins {
	// 	delete(stmt.Joins, k)
	// }

	// for k := range stmt.Preloads {
	// 	delete(stmt.Preloads, k)
	// }

	// stmt.Settings.Range(func(k, _ interface{}) bool {
	// 	stmt.Settings.Delete(k)
	// 	return true
	// })

	// stmt.Schema = nil
	if !stmt.DB.DryRun {
		stmt.SQL.Reset()
		stmt.Vars = nil
		stmt.NamedVars = nil
	}
}
