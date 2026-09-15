package gorm

import (
	"reflect"
	"testing"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// addVarTestDialector is a minimal Dialector implementation used by AddVar
// tests and benchmarks. It cannot live in gorm/utils/tests because that
// package depends on gorm.io/gorm, which would create an import cycle when
// referenced from inside package gorm.
type addVarTestDialector struct{}

func (addVarTestDialector) Name() string {
	return "addvar_test"
}

func (addVarTestDialector) Initialize(*DB) error {
	return nil
}

func (addVarTestDialector) Migrator(*DB) Migrator {
	return nil
}

func (addVarTestDialector) DataTypeOf(*schema.Field) string {
	return ""
}

func (addVarTestDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (addVarTestDialector) BindVarTo(writer clause.Writer, stmt *Statement, v interface{}) {
	_ = writer.WriteByte('?')
}

func (addVarTestDialector) QuoteTo(writer clause.Writer, str string) {
	_, _ = writer.WriteString(str)
}

func (addVarTestDialector) Explain(sql string, vars ...interface{}) string {
	return sql
}

func newAddVarStatement() *Statement {
	db := &DB{Config: &Config{Dialector: addVarTestDialector{}}}
	db.Statement = &Statement{DB: db}
	return db.Statement
}

func TestAddVarIntSlice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []int{1, 2, 3})

	if got, want := stmt.SQL.String(), "(?,?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	if got, want := len(stmt.Vars), 3; got != want {
		t.Fatalf("Vars length mismatch: got %d, want %d", got, want)
	}
	for i, v := range stmt.Vars {
		iv, ok := v.(int)
		if !ok {
			t.Errorf("Vars[%d] type = %T, want int", i, v)
			continue
		}
		if iv != i+1 {
			t.Errorf("Vars[%d] = %d, want %d", i, iv, i+1)
		}
	}
}

func TestAddVarInt64Slice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []int64{10, 20, 30})

	if got, want := stmt.SQL.String(), "(?,?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	for i, v := range stmt.Vars {
		if _, ok := v.(int64); !ok {
			t.Errorf("Vars[%d] type = %T, want int64", i, v)
		}
	}
}

func TestAddVarUintSlice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []uint{1, 2})

	if got, want := stmt.SQL.String(), "(?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	for i, v := range stmt.Vars {
		if _, ok := v.(uint); !ok {
			t.Errorf("Vars[%d] type = %T, want uint", i, v)
		}
	}
}

func TestAddVarFloat64Slice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []float64{1.5, 2.5})

	if got, want := stmt.SQL.String(), "(?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	if v, ok := stmt.Vars[0].(float64); !ok || v != 1.5 {
		t.Errorf("Vars[0] = %v (%T), want float64(1.5)", stmt.Vars[0], stmt.Vars[0])
	}
}

func TestAddVarStringSlice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []string{"a", "b"})

	if got, want := stmt.SQL.String(), "(?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	for i, v := range stmt.Vars {
		if _, ok := v.(string); !ok {
			t.Errorf("Vars[%d] type = %T, want string", i, v)
		}
	}
}

func TestAddVarBoolSlice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []bool{true, false})

	if got, want := stmt.SQL.String(), "(?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	for i, v := range stmt.Vars {
		if _, ok := v.(bool); !ok {
			t.Errorf("Vars[%d] type = %T, want bool", i, v)
		}
	}
}

func TestAddVarEmptyTypedSlice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []int{})

	if got, want := stmt.SQL.String(), "(NULL)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	if len(stmt.Vars) != 0 {
		t.Errorf("Vars should be empty, got %v", stmt.Vars)
	}
}

func TestAddVarByteSlice(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []byte("abc"))

	if got, want := stmt.SQL.String(), "?"; got != want {
		t.Errorf("SQL mismatch for []byte: got %q, want %q", got, want)
	}
	if len(stmt.Vars) != 1 {
		t.Fatalf("Vars length for []byte = %d, want 1", len(stmt.Vars))
	}
	if _, ok := stmt.Vars[0].([]byte); !ok {
		t.Errorf("Vars[0] type = %T, want []byte", stmt.Vars[0])
	}
}

func TestAddVarCustomSliceAliasFallback(t *testing.T) {
	type userIDs []int
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, userIDs{1, 2, 3})

	if got, want := stmt.SQL.String(), "(?,?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	for i, v := range stmt.Vars {
		if _, ok := v.(int); !ok {
			t.Errorf("Vars[%d] type = %T, want int (preserved by reflect fallback)", i, v)
		}
	}
}

func TestAddVarInterfaceSliceUnchanged(t *testing.T) {
	stmt := newAddVarStatement()
	stmt.AddVar(&stmt.SQL, []interface{}{1, "two", 3.0})

	if got, want := stmt.SQL.String(), "(?,?,?)"; got != want {
		t.Errorf("SQL mismatch: got %q, want %q", got, want)
	}
	if len(stmt.Vars) != 3 {
		t.Fatalf("Vars length = %d, want 3", len(stmt.Vars))
	}
	if reflect.TypeOf(stmt.Vars[0]).Kind() != reflect.Int {
		t.Errorf("Vars[0] kind = %v, want int", reflect.TypeOf(stmt.Vars[0]).Kind())
	}
	if _, ok := stmt.Vars[1].(string); !ok {
		t.Errorf("Vars[1] type = %T, want string", stmt.Vars[1])
	}
	if _, ok := stmt.Vars[2].(float64); !ok {
		t.Errorf("Vars[2] type = %T, want float64", stmt.Vars[2])
	}
}

// addVarSink prevents the compiler from optimizing away benchmark results.
var addVarSink string

func benchmarkAddVar(b *testing.B, v interface{}) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stmt := newAddVarStatement()
		stmt.AddVar(&stmt.SQL, v)
		addVarSink = stmt.SQL.String()
	}
}

func BenchmarkStatementAddVar(b *testing.B) {
	const n = 100

	b.Run("int_slice", func(b *testing.B) {
		v := make([]int, n)
		for i := range v {
			v[i] = i
		}
		benchmarkAddVar(b, v)
	})

	b.Run("int64_slice", func(b *testing.B) {
		v := make([]int64, n)
		for i := range v {
			v[i] = int64(i)
		}
		benchmarkAddVar(b, v)
	})

	b.Run("uint_slice", func(b *testing.B) {
		v := make([]uint, n)
		for i := range v {
			v[i] = uint(i) //nolint:gosec // benchmark loop index is bounded by n
		}
		benchmarkAddVar(b, v)
	})

	b.Run("float64_slice", func(b *testing.B) {
		v := make([]float64, n)
		for i := range v {
			v[i] = float64(i)
		}
		benchmarkAddVar(b, v)
	})

	b.Run("string_slice", func(b *testing.B) {
		v := make([]string, n)
		for i := range v {
			v[i] = "v"
		}
		benchmarkAddVar(b, v)
	})

	b.Run("bool_slice", func(b *testing.B) {
		v := make([]bool, n)
		for i := range v {
			v[i] = i%2 == 0
		}
		benchmarkAddVar(b, v)
	})

	b.Run("interface_slice", func(b *testing.B) {
		v := make([]interface{}, n)
		for i := range v {
			v[i] = i
		}
		benchmarkAddVar(b, v)
	})

	b.Run("custom_alias_slice", func(b *testing.B) {
		type userIDs []int
		v := make(userIDs, n)
		for i := range v {
			v[i] = i
		}
		benchmarkAddVar(b, v)
	})

	b.Run("byte_slice_scalar", func(b *testing.B) {
		v := make([]byte, n)
		for i := range v {
			v[i] = byte(i)
		}
		benchmarkAddVar(b, v)
	})
}
