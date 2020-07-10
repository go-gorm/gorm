package gorm

import (
	"database/sql"
	"database/sql/driver"
	"reflect"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type DeletedAt sql.NullTime

// Scan implements the Scanner interface.
func (n *DeletedAt) Scan(value interface{}) error {
	return (*sql.NullTime)(n).Scan(value)
}

// Value implements the driver Valuer interface.
func (n DeletedAt) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

func (DeletedAt) QueryClauses() []clause.Interface {
	return []clause.Interface{
		clause.Where{Exprs: []clause.Expression{
			clause.Eq{
				Column: clause.Column{Table: clause.CurrentTable, Name: "deleted_at"},
				Value:  nil,
			},
		}},
	}
}

func (DeletedAt) DeleteClauses() []clause.Interface {
	return []clause.Interface{SoftDeleteClause{}}
}

type SoftDeleteClause struct {
}

func (SoftDeleteClause) Name() string {
	return ""
}

func (SoftDeleteClause) Build(clause.Builder) {
}

func (SoftDeleteClause) MergeClause(*clause.Clause) {
}

func (SoftDeleteClause) ModifyStatement(stmt *Statement) {
	if stmt.SQL.String() == "" {
		stmt.AddClause(clause.Set{{Column: clause.Column{Name: "deleted_at"}, Value: stmt.DB.NowFunc()}})

		if stmt.Schema != nil {
			_, queryValues := schema.GetIdentityFieldValuesMap(stmt.ReflectValue, stmt.Schema.PrimaryFields)
			column, values := schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

			if len(values) > 0 {
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
			}

			if stmt.Dest != stmt.Model && stmt.Model != nil {
				_, queryValues = schema.GetIdentityFieldValuesMap(reflect.ValueOf(stmt.Model), stmt.Schema.PrimaryFields)
				column, values = schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}
			}
		}

		if _, ok := stmt.Clauses["WHERE"]; !ok {
			stmt.DB.AddError(ErrMissingWhereClause)
			return
		}

		stmt.AddClauseIfNotExists(clause.Update{})
		stmt.Build("UPDATE", "SET", "WHERE")
	}
}
