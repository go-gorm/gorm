package gorm

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"

	"github.com/jinzhu/now"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// DeletedAt is a nullable deletion timestamp that installs GORM soft-delete clauses
// when used as a model field. An invalid value represents SQL NULL and an active
// record. Normal queries/updates exclude deleted records; Delete writes the current
// time instead of removing the row. Unscoped bypasses these clauses.
// It implements sql.Scanner, driver.Valuer, and JSON marshaling.
// See https://gorm.io/docs/delete.html#Soft-Delete.
type DeletedAt sql.NullTime

// Scan reads a database value using sql.NullTime conversion rules, including NULL. It
// changes the timestamp and validity and returns conversion failures.
func (n *DeletedAt) Scan(value interface{}) error {
	return (*sql.NullTime)(n).Scan(value)
}

// Value returns SQL NULL when Valid is false and the stored time otherwise. It does not
// itself change deletion state or execute SQL.
func (n DeletedAt) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

// MarshalJSON encodes a valid deletion timestamp as a JSON time string and an invalid
// timestamp as null.
func (n DeletedAt) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Time)
	}
	return json.Marshal(nil)
}

// UnmarshalJSON clears Valid for JSON null; otherwise it parses a time and marks it
// valid on success. Invalid input returns an error, so callers must not assume a
// successful state change.
func (n *DeletedAt) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Time)
	if err == nil {
		n.Valid = true
	}
	return err
}

// QueryClauses installs the active-row predicate for field f. The optional ZEROVALUE
// tag selects a valid timestamp sentinel in place of SQL NULL.
func (DeletedAt) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{SoftDeleteQueryClause{Field: f, ZeroValue: parseZeroValueTag(f)}}
}

// parseZeroValueTag accepts a parseable ZEROVALUE timestamp tag; a missing or invalid
// tag yields an invalid NullString, representing SQL NULL in the predicate.
func parseZeroValueTag(f *schema.Field) sql.NullString {
	if v, ok := f.TagSettings["ZEROVALUE"]; ok {
		if _, err := now.Parse(v); err == nil {
			return sql.NullString{String: v, Valid: true}
		}
	}
	return sql.NullString{Valid: false}
}

// SoftDeleteQueryClause adds the active-row predicate once per scoped statement. It
// groups existing OR predicates before adding the deletion filter so they cannot bypass
// it.
type SoftDeleteQueryClause struct {
	// ZeroValue identifies an active row: an invalid NullString means SQL NULL; a valid
	// value is the configured timestamp sentinel.
	ZeroValue sql.NullString
	// Field is the parsed model field whose database column stores the deletion timestamp.
	Field *schema.Field
}

// Name returns an empty name because this clause modifies the statement rather than
// emitting a standalone named SQL clause.
func (sd SoftDeleteQueryClause) Name() string {
	return ""
}

// Build emits no SQL directly; ModifyStatement installs the clauses used by the
// operation builder.
func (sd SoftDeleteQueryClause) Build(clause.Builder) {
}

// MergeClause is a no-op because this modifier does not merge a standalone named
// clause.
func (sd SoftDeleteQueryClause) MergeClause(*clause.Clause) {
}

// ModifyStatement adds the active-row condition unless Unscoped is set or the statement
// is already marked soft_delete_enabled. It groups existing OR terms before adding the
// predicate.
func (sd SoftDeleteQueryClause) ModifyStatement(stmt *Statement) {
	if _, ok := stmt.Clauses["soft_delete_enabled"]; !ok && !stmt.Statement.Unscoped {
		if c, ok := stmt.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause.Where); ok && len(where.Exprs) >= 1 {
				for _, expr := range where.Exprs {
					if orCond, ok := expr.(clause.OrConditions); ok && len(orCond.Exprs) == 1 {
						where.Exprs = []clause.Expression{clause.And(where.Exprs...)}
						c.Expression = where
						stmt.Clauses["WHERE"] = c
						break
					}
				}
			}
		}

		stmt.AddClause(clause.Where{Exprs: []clause.Expression{
			clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: sd.Field.DBName}, Value: sd.ZeroValue},
		}})
		stmt.Clauses["soft_delete_enabled"] = clause.Clause{}
	}
}

// UpdateClauses installs the active-row filter for generated UPDATE statements,
// preventing ordinary updates of soft-deleted rows.
func (DeletedAt) UpdateClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{SoftDeleteUpdateClause{Field: f, ZeroValue: parseZeroValueTag(f)}}
}

// SoftDeleteUpdateClause applies the query soft-delete predicate before building a
// scoped UPDATE. It leaves prebuilt SQL and Unscoped statements unchanged.
type SoftDeleteUpdateClause struct {
	// ZeroValue identifies an active row: an invalid NullString means SQL NULL; a valid
	// value is the configured timestamp sentinel.
	ZeroValue sql.NullString
	// Field is the parsed model field whose database column stores the deletion timestamp.
	Field *schema.Field
}

// Name returns an empty name because this clause modifies the statement rather than
// emitting a standalone named SQL clause.
func (sd SoftDeleteUpdateClause) Name() string {
	return ""
}

// Build emits no SQL directly; ModifyStatement installs the clauses used by the
// operation builder.
func (sd SoftDeleteUpdateClause) Build(clause.Builder) {
}

// MergeClause is a no-op because this modifier does not merge a standalone named
// clause.
func (sd SoftDeleteUpdateClause) MergeClause(*clause.Clause) {
}

// ModifyStatement applies the soft-delete query predicate only when SQL has not been
// built and the statement is scoped.
func (sd SoftDeleteUpdateClause) ModifyStatement(stmt *Statement) {
	if stmt.SQL.Len() == 0 && !stmt.Statement.Unscoped {
		SoftDeleteQueryClause(sd).ModifyStatement(stmt)
	}
}

// DeleteClauses installs the modifier that turns scoped deletes into timestamp updates
// for field f.
func (DeletedAt) DeleteClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{SoftDeleteDeleteClause{Field: f, ZeroValue: parseZeroValueTag(f)}}
}

// SoftDeleteDeleteClause rewrites a generated scoped DELETE into an UPDATE of the
// deletion timestamp, preserving primary-key predicates and the active-row filter.
// Unscoped statements retain physical DELETE behavior.
type SoftDeleteDeleteClause struct {
	// ZeroValue identifies an active row: an invalid NullString means SQL NULL; a valid
	// value is the configured timestamp sentinel.
	ZeroValue sql.NullString
	// Field is the parsed model field whose database column stores the deletion timestamp.
	Field *schema.Field
}

// Name returns an empty name because this clause modifies the statement rather than
// emitting a standalone named SQL clause.
func (sd SoftDeleteDeleteClause) Name() string {
	return ""
}

// Build emits no SQL directly; ModifyStatement installs the clauses used by the
// operation builder.
func (sd SoftDeleteDeleteClause) Build(clause.Builder) {
}

// MergeClause is a no-op because this modifier does not merge a standalone named
// clause.
func (sd SoftDeleteDeleteClause) MergeClause(*clause.Clause) {
}

// ModifyStatement sets the deletion timestamp in the model and SQL, adds available
// primary-key conditions, and builds UPDATE clauses for a scoped unbuilt statement. The
// delete callback still enforces the missing-WHERE guard.
func (sd SoftDeleteDeleteClause) ModifyStatement(stmt *Statement) {
	if stmt.SQL.Len() == 0 && !stmt.Statement.Unscoped {
		curTime := stmt.DB.NowFunc()
		stmt.AddClause(clause.Set{{Column: clause.Column{Name: sd.Field.DBName}, Value: curTime}})
		stmt.SetColumn(sd.Field.DBName, curTime, true)

		if stmt.Schema != nil {
			_, queryValues := schema.GetIdentityFieldValuesMap(stmt.Context, stmt.ReflectValue, stmt.Schema.PrimaryFields)
			column, values := schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

			if len(values) > 0 {
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
			}

			if stmt.ReflectValue.CanAddr() && stmt.Dest != stmt.Model && stmt.Model != nil {
				_, queryValues = schema.GetIdentityFieldValuesMap(stmt.Context, reflect.ValueOf(stmt.Model), stmt.Schema.PrimaryFields)
				column, values = schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}
			}
		}

		SoftDeleteQueryClause(sd).ModifyStatement(stmt)
		stmt.AddClauseIfNotExists(clause.Update{})
		stmt.Build(stmt.DB.Callback().Update().Clauses...)
	}
}
