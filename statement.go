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

// AddError add error to instance
func (inst Instance) AddError(err error) {
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
	OptimizeStatement(Statement)
}

// Write write string
func (stmt Statement) Write(sql ...string) (err error) {
	for _, s := range sql {
		_, err = stmt.SQL.WriteString(s)
	}
	return
}

// Write write string
func (stmt Statement) WriteByte(c byte) (err error) {
	return stmt.SQL.WriteByte(c)
}

// WriteQuoted write quoted field
func (stmt Statement) WriteQuoted(field interface{}) (err error) {
	_, err = stmt.SQL.WriteString(stmt.Quote(field))
	return
}

// Quote returns quoted value
func (stmt Statement) Quote(field interface{}) string {
	var str strings.Builder

	switch v := field.(type) {
	case clause.Table:
		str.WriteString(v.Table)
		if v.Alias != "" {
			str.WriteString(" AS ")
			str.WriteString(v.Alias)
		}
	case clause.Column:
		if v.Table != "" {
			str.WriteString(v.Table)
			str.WriteByte('.')
		}

		str.WriteString(v.Name)
		if v.Alias != "" {
			str.WriteString(" AS ")
			str.WriteString(v.Alias)
		}
	default:
		fmt.Sprint(field)
	}

	return str.String()
}

// Write write string
func (stmt Statement) AddVar(vars ...interface{}) string {
	var placeholders strings.Builder
	for idx, v := range vars {
		if idx > 0 {
			placeholders.WriteByte(',')
		}

		if namedArg, ok := v.(sql.NamedArg); ok && len(namedArg.Name) > 0 {
			stmt.NamedVars = append(stmt.NamedVars, namedArg)
			placeholders.WriteByte('@')
			placeholders.WriteString(namedArg.Name)
		} else if arrs, ok := v.([]interface{}); ok {
			placeholders.WriteByte('(')
			if len(arrs) > 0 {
				placeholders.WriteString(stmt.AddVar(arrs...))
			} else {
				placeholders.WriteString("NULL")
			}
			placeholders.WriteByte(')')
		} else {
			placeholders.WriteString(stmt.DB.Dialector.BindVar(stmt, v))
		}
	}
	return placeholders.String()
}

// AddClause add clause
func (stmt Statement) AddClause(v clause.Interface) {
	if optimizer, ok := v.(StatementOptimizer); ok {
		optimizer.OptimizeStatement(stmt)
	}

	c, _ := stmt.Clauses[v.Name()]
	if namer, ok := v.(clause.OverrideNameInterface); ok {
		c.Name = namer.OverrideName()
	} else {
		c.Name = v.Name()
	}

	if c.Expression != nil {
		v.MergeExpression(c.Expression)
	}

	c.Expression = v
	stmt.Clauses[v.Name()] = c
}

// BuildCondtion build condition
func (stmt Statement) BuildCondtion(query interface{}, args ...interface{}) (conditions []clause.Expression) {
	if sql, ok := query.(string); ok {
		if i, err := strconv.Atoi(sql); err != nil {
			query = i
		} else if len(args) == 0 || (len(args) > 0 && strings.Contains(sql, "?")) || strings.Contains(sql, "@") {
			return []clause.Expression{clause.String{SQL: sql, Values: args}}
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
		conditions = append(conditions, clause.ID{Value: args})
	}

	return conditions
}

// Build build sql with clauses names
func (stmt Statement) Build(clauses ...string) {
	var includeSpace bool

	for _, name := range clauses {
		if c, ok := stmt.Clauses[name]; ok {
			if includeSpace {
				stmt.WriteByte(' ')
			}

			includeSpace = true
			c.Build(stmt)
		}
	}
}
