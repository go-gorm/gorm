package gorm

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jinzhu/gorm/clause"
)

// Statement statement
type Statement struct {
	Model    interface{}
	Dest     interface{}
	Table    string
	Clauses  map[string][]clause.Condition
	Settings sync.Map
	Context  context.Context
	DB       *DB
	StatementBuilder
}

// StatementBuilder statement builder
type StatementBuilder struct {
	SQL       bytes.Buffer
	Vars      []interface{}
	NamedVars []sql.NamedArg
}

// Write write string
func (stmt Statement) Write(sql ...string) (err error) {
	for _, s := range sql {
		_, err = stmt.SQL.WriteString(s)
	}
	return
}

// WriteQuoted write quoted field
func (stmt Statement) WriteQuoted(field interface{}) (err error) {
	_, err = stmt.SQL.WriteString(stmt.Quote(field))
	return
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

// Quote returns quoted value
func (stmt Statement) Quote(field interface{}) (str string) {
	return fmt.Sprint(field)
}

// AddClause add clause
func (s Statement) AddClause(clause clause.Interface) {
	s.Clauses[clause.Name()] = append(s.Clauses[clause.Name()], clause)
}

// BuildCondtions build conditions
func (s Statement) BuildCondtions(query interface{}, args ...interface{}) (conditions []clause.Condition) {
	if sql, ok := query.(string); ok {
		if i, err := strconv.Atoi(sql); err != nil {
			query = i
		} else if len(args) == 0 || (len(args) > 0 && strings.Contains(sql, "?")) || strings.Contains(sql, "@") {
			return []clause.Condition{clause.Raw{SQL: sql, Values: args}}
		}
	}

	args = append([]interface{}{query}, args...)
	for _, arg := range args {
		if valuer, ok := arg.(driver.Valuer); ok {
			arg, _ = valuer.Value()
		}

		switch v := arg.(type) {
		case clause.Builder:
			conditions = append(conditions, v)
		case *DB:
			if v.Statement == nil {
				if cs, ok := v.Statement.Clauses["WHERE"]; ok {
					conditions = append(conditions, cs...)
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

func (s Statement) AddError(err error) {
}
