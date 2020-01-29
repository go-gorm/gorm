package gorm

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/jinzhu/gorm/clause"
)

// Statement statement
type Statement struct {
	Dest     interface{}
	Table    interface{}
	Clauses  map[string][]clause.Interface
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
	var placeholders []string
	for _, v := range vars {
		if namedArg, ok := v.(sql.NamedArg); ok && len(namedArg.Name) > 0 {
			stmt.NamedVars = append(stmt.NamedVars, namedArg)
			placeholders = append(placeholders, "@"+namedArg.Name)
		} else {
			placeholders = append(placeholders, stmt.DB.Dialector.BindVar(stmt, v))
		}
	}
	return strings.Join(placeholders, ",")
}

// Quote returns quoted value
func (stmt Statement) Quote(field interface{}) (str string) {
	return fmt.Sprint(field)
}

// AddClause add clause
func (s Statement) AddClause(clause clause.Interface) {
	s.Clauses[clause.Name()] = append(s.Clauses[clause.Name()], clause)
}
