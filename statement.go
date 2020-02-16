package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
)

// Instance db instance
type Instance struct {
	Error        error
	RowsAffected int64
	Context      context.Context
	Statement    *Statement
}

func (instance *Instance) ToSQL(clauses ...string) (string, []interface{}) {
	if len(clauses) > 0 {
		instance.Statement.Build(clauses...)
	}
	return strings.TrimSpace(instance.Statement.SQL.String()), instance.Statement.Vars
}

// AddError add error to instance
func (inst *Instance) AddError(err error) {
	if inst.Error == nil {
		inst.Error = err
	} else {
		inst.Error = fmt.Errorf("%v; %w", inst.Error, err)
	}
}

// Statement statement
type Statement struct {
	Table    string
	Model    interface{}
	Dest     interface{}
	Clauses  map[string]clause.Clause
	Selects  []string // selected columns
	Omits    []string // omit columns
	Settings sync.Map
	DB       *DB
	Schema   *schema.Schema

	// SQL Builder
	SQL       strings.Builder
	Vars      []interface{}
	NamedVars []sql.NamedArg
}

// StatementOptimizer statement optimizer interface
type StatementOptimizer interface {
	OptimizeStatement(*Statement)
}

// Write write string
func (stmt *Statement) Write(sql ...string) (err error) {
	for _, s := range sql {
		_, err = stmt.SQL.WriteString(s)
	}
	return
}

// Write write string
func (stmt *Statement) WriteByte(c byte) (err error) {
	return stmt.SQL.WriteByte(c)
}

// WriteQuoted write quoted field
func (stmt *Statement) WriteQuoted(field interface{}) (err error) {
	_, err = stmt.SQL.WriteString(stmt.Quote(field))
	return
}

// Quote returns quoted value
func (stmt Statement) Quote(field interface{}) string {
	var str strings.Builder
	str.WriteByte(stmt.DB.quoteChars[0])

	switch v := field.(type) {
	case clause.Table:
		if v.Name == clause.CurrentTable {
			str.WriteString(stmt.Table)
		} else {
			str.WriteString(v.Name)
		}

		if v.Alias != "" {
			str.WriteByte(stmt.DB.quoteChars[1])
			str.WriteString(" AS ")
			str.WriteByte(stmt.DB.quoteChars[0])
			str.WriteString(v.Alias)
			str.WriteByte(stmt.DB.quoteChars[1])
		}
	case clause.Column:
		if v.Table != "" {
			if v.Table == clause.CurrentTable {
				str.WriteString(stmt.Table)
			} else {
				str.WriteString(v.Table)
			}
			str.WriteByte(stmt.DB.quoteChars[1])
			str.WriteByte('.')
			str.WriteByte(stmt.DB.quoteChars[0])
		}

		if v.Name == clause.PrimaryKey {
			if stmt.Schema != nil && stmt.Schema.PrioritizedPrimaryField != nil {
				str.WriteString(stmt.Schema.PrioritizedPrimaryField.DBName)
			}
		} else {
			str.WriteString(v.Name)
		}

		if v.Alias != "" {
			str.WriteByte(stmt.DB.quoteChars[1])
			str.WriteString(" AS ")
			str.WriteByte(stmt.DB.quoteChars[0])
			str.WriteString(v.Alias)
			str.WriteByte(stmt.DB.quoteChars[1])
		}
	default:
		str.WriteString(fmt.Sprint(field))
	}

	str.WriteByte(stmt.DB.quoteChars[1])
	return str.String()
}

// Write write string
func (stmt *Statement) AddVar(vars ...interface{}) string {
	var placeholders strings.Builder
	for idx, v := range vars {
		if idx > 0 {
			placeholders.WriteByte(',')
		}

		switch v := v.(type) {
		case sql.NamedArg:
			if len(v.Name) > 0 {
				stmt.NamedVars = append(stmt.NamedVars, v)
				placeholders.WriteByte('@')
				placeholders.WriteString(v.Name)
			} else {
				stmt.Vars = append(stmt.Vars, v.Value)
				placeholders.WriteString(stmt.DB.Dialector.BindVar(stmt, v.Value))
			}
		case clause.Column:
			placeholders.WriteString(stmt.Quote(v))
		case []interface{}:
			if len(v) > 0 {
				placeholders.WriteByte('(')
				placeholders.WriteString(stmt.AddVar(v...))
				placeholders.WriteByte(')')
			} else {
				placeholders.WriteString("(NULL)")
			}
		default:
			stmt.Vars = append(stmt.Vars, v)
			placeholders.WriteString(stmt.DB.Dialector.BindVar(stmt, v))
		}
	}
	return placeholders.String()
}

// AddClause add clause
func (stmt *Statement) AddClause(v clause.Interface) {
	if optimizer, ok := v.(StatementOptimizer); ok {
		optimizer.OptimizeStatement(stmt)
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
	if _, ok := stmt.Clauses[v.Name()]; !ok {
		stmt.AddClause(v)
	}
}

// BuildCondtion build condition
func (stmt Statement) BuildCondtion(query interface{}, args ...interface{}) (conditions []clause.Expression) {
	if sql, ok := query.(string); ok {
		if i, err := strconv.Atoi(sql); err != nil {
			query = i
		} else if len(args) == 0 || (len(args) > 0 && strings.Contains(sql, "?")) || strings.Contains(sql, "@") {
			return []clause.Expression{clause.Expr{SQL: sql, Vars: args}}
		}
	}

	args = append([]interface{}{query}, args...)
	for _, arg := range args {
		if valuer, ok := arg.(driver.Valuer); ok {
			arg, _ = valuer.Value()
		}

		switch v := arg.(type) {
		case clause.Expression:
			conditions = append(conditions, v)
		case *DB:
			if v.Statement == nil {
				if cs, ok := v.Statement.Clauses["WHERE"]; ok {
					conditions = append(conditions, cs.Expression)
				}
			}
		case map[interface{}]interface{}:
			var clauseMap = clause.Map{}
			for i, j := range v {
				clauseMap[i] = j
			}
			conditions = append(conditions, clauseMap)
		case map[string]string:
			var clauseMap = clause.Map{}
			for i, j := range v {
				clauseMap[i] = j
			}
			conditions = append(conditions, clauseMap)
		case map[string]interface{}:
			var clauseMap = clause.Map{}
			for i, j := range v {
				clauseMap[i] = j
			}
			conditions = append(conditions, clauseMap)
		default:
			// TODO check is struct
			// struct, slice -> ids
		}
	}

	if len(conditions) == 0 {
		conditions = append(conditions, clause.IN{Column: clause.PrimaryColumn, Values: args})
	}

	return conditions
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
				b.Build(c, stmt)
			} else {
				c.Build(stmt)
			}
		}
	}
	// TODO handle named vars
}
