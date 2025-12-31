package tests

import (
	"context"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// testDialector is a minimal Dialector implementation used only for unit tests in-memory.
type testDialector struct{}

func (d testDialector) Name() string                                   { return "test" }
func (d testDialector) Initialize(*gorm.DB) error                      { return nil }
func (d testDialector) Migrator(db *gorm.DB) gorm.Migrator             { return nil }
func (d testDialector) DataTypeOf(*schema.Field) string                { return "" }
func (d testDialector) DefaultValueOf(*schema.Field) clause.Expression { return clause.Expr{} }
func (d testDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	// write a simple placeholder
	writer.WriteByte('?')
}
func (d testDialector) QuoteTo(writer clause.Writer, s string)         { writer.WriteString(s) }
func (d testDialector) Explain(sql string, vars ...interface{}) string { return sql }

// newTestDB returns a minimal *DB with an initialized Statement suitable for unit tests
func newTestDB() *gorm.DB {
	d := testDialector{}
	cfg := &gorm.Config{Dialector: d}
	db := &gorm.DB{Config: cfg}
	stmt := &gorm.Statement{
		DB:       db,
		Clauses:  map[string]clause.Clause{},
		Preloads: map[string][]interface{}{},
		Context:  context.Background(),
		Vars:     make([]interface{}, 0),
	}
	db.Statement = stmt
	return db
}

func TestChainableAPI(t *testing.T) {
	db := newTestDB()

	// Model
	m := &struct{ ID int }{}
	tx := db.Model(m)
	if tx.Statement.Model != m {
		t.Fatalf("Model not set, got %v", tx.Statement.Model)
	}

	// Table
	tx = tx.Table("users")
	if tx.Statement.Table != "users" {
		t.Fatalf("Table not set, got %v", tx.Statement.Table)
	}
	if tx.Statement.TableExpr == nil {
		t.Fatalf("TableExpr expected to be set")
	}

	// Distinct + Select
	tx = tx.Distinct("name", "age")
	if !tx.Statement.Distinct {
		t.Fatalf("Distinct expected true")
	}
	if len(tx.Statement.Selects) != 2 || tx.Statement.Selects[0] != "name" {
		t.Fatalf("Selects expected [name age], got %v", tx.Statement.Selects)
	}

	// Where
	tx = tx.Where("age = ?", 20)
	c, ok := tx.Statement.Clauses["WHERE"]
	if !ok {
		t.Fatalf("WHERE clause expected")
	}
	if where, ok := c.Expression.(clause.Where); !ok || len(where.Exprs) == 0 {
		t.Fatalf("WHERE expressions expected, got %v", c.Expression)
	}

	// Order
	tx = tx.Order("name DESC")
	if _, ok := tx.Statement.Clauses["ORDER BY"]; !ok {
		t.Fatalf("ORDER BY clause expected")
	}

	// Limit / Offset
	tx = tx.Limit(10).Offset(5)
	if cl, ok := tx.Statement.Clauses["LIMIT"]; !ok {
		t.Fatalf("LIMIT clause expected")
	} else {
		if limit, ok := cl.Expression.(clause.Limit); !ok || limit.Limit == nil || *limit.Limit != 10 || limit.Offset != 5 {
			t.Fatalf("LIMIT/Offset values unexpected: %v", cl.Expression)
		}
	}

	// Joins
	tx = tx.Joins("JOIN accounts ON accounts.user_id = users.id")
	if len(tx.Statement.Joins) == 0 {
		t.Fatalf("Joins expected")
	}
	if tx.Statement.Joins[0].Name != "JOIN accounts ON accounts.user_id = users.id" {
		t.Fatalf("Join name mismatch: %v", tx.Statement.Joins[0].Name)
	}

	// Preload
	tx = tx.Preload("Orders", "state != ?", "cancelled")
	args, ok := tx.Statement.Preloads["Orders"]
	if !ok || len(args) != 2 {
		t.Fatalf("Preload expected with args, got %v", tx.Statement.Preloads)
	}

	// Scopes
	tx = tx.Scopes(func(d *gorm.DB) *gorm.DB { return d.Where("status = ?", "ok") })
	if len(tx.Statement.scopes) == 0 {
		t.Fatalf("Scopes expected to be recorded")
	}

	// executeScopes should apply the recorded scope and add another WHERE
	tx = tx.executeScopes()
	if _, ok := tx.Statement.Clauses["WHERE"]; !ok {
		t.Fatalf("WHERE clause expected after executeScopes")
	}

	// Unscoped
	tx = tx.Unscoped()
	if !tx.Statement.Unscoped {
		t.Fatalf("Unscoped expected to be true")
	}

	// Raw
	tx = tx.Raw("SELECT ? as x", 1)
	if tx.Statement.SQL.Len() == 0 {
		t.Fatalf("Raw SQL expected to be built")
	}
	if len(tx.Statement.Vars) != 1 || tx.Statement.Vars[0] != 1 {
		t.Fatalf("Raw Vars expected to contain 1, got %v", tx.Statement.Vars)
	}
}
