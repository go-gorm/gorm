package gorm

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// pgStyleDialector emits PostgreSQL-style "$N" bind variables. It is used to
// exercise the subquery bind-variable normalization path in Statement.AddVar
// without depending on a real driver.
type pgStyleDialector struct{}

func (pgStyleDialector) Name() string                    { return "pgstyle" }
func (pgStyleDialector) Initialize(*DB) error            { return nil }
func (pgStyleDialector) Migrator(*DB) Migrator           { return nil }
func (pgStyleDialector) DataTypeOf(*schema.Field) string { return "" }
func (pgStyleDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (pgStyleDialector) BindVarTo(w clause.Writer, s *Statement, _ interface{}) {
	_ = w.WriteByte('$')
	_, _ = w.WriteString(strconv.Itoa(len(s.Vars)))
}
func (pgStyleDialector) QuoteTo(w clause.Writer, str string)         { _, _ = w.WriteString(str) }
func (pgStyleDialector) Explain(sql string, _ ...interface{}) string { return sql }

// qStyleDialector emits the simple "?" placeholder, exercising the no-op
// branch of replaceBindVarsWithQuestion (no rewrite needed).
type qStyleDialector struct{}

func (qStyleDialector) Name() string                    { return "qstyle" }
func (qStyleDialector) Initialize(*DB) error            { return nil }
func (qStyleDialector) Migrator(*DB) Migrator           { return nil }
func (qStyleDialector) DataTypeOf(*schema.Field) string { return "" }
func (qStyleDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (qStyleDialector) BindVarTo(w clause.Writer, _ *Statement, _ interface{}) {
	_ = w.WriteByte('?')
}
func (qStyleDialector) QuoteTo(w clause.Writer, str string)         { _, _ = w.WriteString(str) }
func (qStyleDialector) Explain(sql string, _ ...interface{}) string { return sql }

func newSubqueryDB(d Dialector) *DB {
	cfg := &Config{
		Dialector:  d,
		NowFunc:    func() time.Time { return time.Time{} },
		Logger:     logger.Discard,
		cacheStore: &sync.Map{},
		Plugins:    map[string]Plugin{},
	}
	db := &DB{Config: cfg, clone: 0}
	db.Statement = &Statement{
		DB:      db,
		Context: context.Background(),
		Clauses: map[string]clause.Clause{},
		Vars:    make([]interface{}, 0, 8),
	}
	return db
}

// buildSubquerySub builds a sub-DB whose pre-rendered SQL contains n
// placeholders in the form expected by d, mirroring the state of a subquery
// DB after Raw() has populated Statement.SQL/Vars.
func buildSubquerySub(d Dialector, n int) *DB {
	sub := newSubqueryDB(d)
	var b strings.Builder
	_, _ = b.WriteString("SELECT id FROM users WHERE ")
	for i := 1; i <= n; i++ {
		if i > 1 {
			_, _ = b.WriteString(" AND ")
		}
		_, _ = b.WriteString("c")
		_, _ = b.WriteString(strconv.Itoa(i))
		_, _ = b.WriteString(" = ")
		var bv strings.Builder
		sub.Statement.Vars = append(sub.Statement.Vars, i)
		d.BindVarTo(&bv, sub.Statement, i)
		_, _ = b.WriteString(bv.String())
	}
	_, _ = sub.Statement.SQL.WriteString(b.String())
	return sub
}

func TestReplaceBindVarsWithQuestion(t *testing.T) {
	t.Run("pg placeholders are normalized in one pass", func(t *testing.T) {
		stmt := newSubqueryDB(pgStyleDialector{}).Statement
		vars := []interface{}{1, 2, 3, 4}
		sql := "VALUES ($1,$2,$3,$4)"
		if got, want := replaceBindVarsWithQuestion(stmt, sql, vars), "VALUES (?,?,?,?)"; got != want {
			t.Fatalf("normalized SQL = %q, want %q", got, want)
		}
		if got, want := len(stmt.Vars), len(vars); got != want {
			t.Fatalf("stmt Vars len = %d, want %d", got, want)
		}
	})

	t.Run("multi digit pg placeholders keep ordering", func(t *testing.T) {
		stmt := newSubqueryDB(pgStyleDialector{}).Statement
		vars := make([]interface{}, 12)
		for i := range vars {
			vars[i] = i + 1
		}
		sub := buildSubquerySub(pgStyleDialector{}, 12)
		sql := sub.Statement.SQL.String()
		got := replaceBindVarsWithQuestion(stmt, sql, vars)
		want := "SELECT id FROM users WHERE c1 = ? AND c2 = ? AND c3 = ? AND c4 = ? AND c5 = ? AND c6 = ? AND c7 = ? AND c8 = ? AND c9 = ? AND c10 = ? AND c11 = ? AND c12 = ?"
		if got != want {
			t.Fatalf("normalized SQL = %q, want %q", got, want)
		}
	})

	t.Run("question placeholders are already normalized", func(t *testing.T) {
		stmt := newSubqueryDB(qStyleDialector{}).Statement
		vars := []interface{}{1, 2, 3}
		sql := "WHERE c1 = ? AND c2 = ? AND c3 = ?"
		if got := replaceBindVarsWithQuestion(stmt, sql, vars); got != sql {
			t.Fatalf("normalized SQL = %q, want original %q", got, sql)
		}
	})

	t.Run("preserves previous textual replacement behavior", func(t *testing.T) {
		stmt := newSubqueryDB(pgStyleDialector{}).Statement
		vars := []interface{}{42}
		sql := "SELECT '$1' AS label, id FROM users WHERE id = $1"
		got := replaceBindVarsWithQuestion(stmt, sql, vars)
		want := "SELECT '?' AS label, id FROM users WHERE id = $1"
		if got != want {
			t.Fatalf("normalized SQL = %q, want %q", got, want)
		}
	})
}

func TestStatementAddVarSubqueryNormalize(t *testing.T) {
	cases := []struct {
		name string
		d    Dialector
		n    int
		want string
	}{
		{
			"pg/n=3",
			pgStyleDialector{},
			3,
			"SELECT id FROM users WHERE c1 = $1 AND c2 = $2 AND c3 = $3",
		},
		{
			"pg/n=12 multi-digit placeholders",
			pgStyleDialector{},
			12,
			"SELECT id FROM users WHERE c1 = $1 AND c2 = $2 AND c3 = $3 AND c4 = $4 AND c5 = $5 AND c6 = $6 AND c7 = $7 AND c8 = $8 AND c9 = $9 AND c10 = $10 AND c11 = $11 AND c12 = $12",
		},
		{
			"q/n=3 already normalized",
			qStyleDialector{},
			3,
			"SELECT id FROM users WHERE c1 = ? AND c2 = ? AND c3 = ?",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sub := buildSubquerySub(tc.d, tc.n)
			outer := newSubqueryDB(tc.d)

			var w strings.Builder
			outer.Statement.AddVar(&w, sub)

			if got := w.String(); got != tc.want {
				t.Fatalf("rendered SQL = %q, want %q", got, tc.want)
			}
			if got, want := len(outer.Statement.Vars), tc.n; got != want {
				t.Fatalf("outer Vars len = %d, want %d", got, want)
			}
			for i := 0; i < tc.n; i++ {
				if outer.Statement.Vars[i] != i+1 {
					t.Fatalf("outer Vars[%d] = %v, want %d", i, outer.Statement.Vars[i], i+1)
				}
			}
		})
	}
}

var benchSubqueryAddVarSink string

func BenchmarkSubqueryAddVar(b *testing.B) {
	cases := []struct {
		name string
		d    Dialector
		n    int
	}{
		{"pg/n=3", pgStyleDialector{}, 3},
		{"pg/n=10", pgStyleDialector{}, 10},
		{"pg/n=30", pgStyleDialector{}, 30},
		{"q/n=10", qStyleDialector{}, 10},
	}
	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			sub := buildSubquerySub(tc.d, tc.n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				outer := newSubqueryDB(tc.d)
				var w strings.Builder
				outer.Statement.AddVar(&w, sub)
				benchSubqueryAddVarSink = w.String()
			}
		})
	}
}
