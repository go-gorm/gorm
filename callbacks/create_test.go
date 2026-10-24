package callbacks

import (
	"context"
	"database/sql"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

var schemaCache = &sync.Map{}

func TestConvertToCreateValues_DestType_Slice(t *testing.T) {
	type user struct {
		ID    int `gorm:"primaryKey"`
		Name  string
		Email string `gorm:"default:(-)"`
		Age   int    `gorm:"default:(-)"`
	}

	s, err := schema.Parse(&user{}, schemaCache, schema.NamingStrategy{})
	if err != nil {
		t.Errorf("parse schema error: %v, is not expected", err)
		return
	}
	dest := []*user{
		{
			ID:    1,
			Name:  "alice",
			Email: "email",
			Age:   18,
		},
		{
			ID:    2,
			Name:  "bob",
			Email: "email",
			Age:   19,
		},
	}
	stmt := &gorm.Statement{
		DB: &gorm.DB{
			Config: &gorm.Config{
				NowFunc: func() time.Time { return time.Time{} },
			},
			Statement: &gorm.Statement{
				Settings: sync.Map{},
				Schema:   s,
			},
		},
		ReflectValue: reflect.ValueOf(dest),
		Dest:         dest,
	}

	stmt.Schema = s

	values := ConvertToCreateValues(stmt)
	expected := clause.Values{
		// column has value + defaultValue column has value (which should have a stable order)
		Columns: []clause.Column{{Name: "name"}, {Name: "email"}, {Name: "age"}, {Name: "id"}},
		Values: [][]interface{}{
			{"alice", "email", 18, 1},
			{"bob", "email", 19, 2},
		},
	}
	if !reflect.DeepEqual(expected, values) {
		t.Errorf("expected: %v got %v", expected, values)
	}
}

type lastInsertIDPool struct {
	id int64
}

func (lastInsertIDPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, nil
}

func (p lastInsertIDPool) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return lastInsertIDResult(p), nil
}

func (lastInsertIDPool) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (lastInsertIDPool) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return nil
}

type lastInsertIDResult struct {
	id int64
}

func (r lastInsertIDResult) LastInsertId() (int64, error) { return r.id, nil }
func (lastInsertIDResult) RowsAffected() (int64, error)   { return 1, nil }

// An auto-increment above MaxInt64 in a BIGINT UNSIGNED column comes back from
// LastInsertId as a negative int64, and the primary key was left at 0.
func TestCreate_UnsignedPrimaryKeyAboveMaxInt64(t *testing.T) {
	type record struct {
		ID   uint64 `gorm:"primaryKey;autoIncrement"`
		Name string
	}

	// math.MinInt64 is how a driver hands over an id of 1 << 63: the same bits
	// read as signed.
	const lastInsertID = int64(math.MinInt64)
	const want = uint64(1) << 63

	s, err := schema.Parse(&record{}, schemaCache, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	dest := &record{Name: "test"}
	db := &gorm.DB{Config: &gorm.Config{}}
	db.Statement = &gorm.Statement{
		DB:           db,
		Context:      context.Background(),
		ConnPool:     lastInsertIDPool{id: lastInsertID},
		Schema:       s,
		Dest:         dest,
		ReflectValue: reflect.ValueOf(dest).Elem(),
		Settings:     sync.Map{},
	}
	db.Statement.SQL.WriteString("INSERT INTO records (name) VALUES (?)")
	db.Statement.Vars = []interface{}{dest.Name}

	Create(&Config{})(db)

	if db.Error != nil {
		t.Fatalf("create: %v", db.Error)
	}
	if dest.ID != want {
		t.Errorf("primary key = %d, want %d", dest.ID, want)
	}
}

// A signed primary key has no such reading: a negative id there is the error
// it always was, and it must stay ignored.
func TestCreate_SignedPrimaryKeyIgnoresNegativeID(t *testing.T) {
	type record struct {
		ID   int64 `gorm:"primaryKey;autoIncrement"`
		Name string
	}

	s, err := schema.Parse(&record{}, schemaCache, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	dest := &record{Name: "test"}
	db := &gorm.DB{Config: &gorm.Config{}}
	db.Statement = &gorm.Statement{
		DB:           db,
		Context:      context.Background(),
		ConnPool:     lastInsertIDPool{id: -1},
		Schema:       s,
		Dest:         dest,
		ReflectValue: reflect.ValueOf(dest).Elem(),
		Settings:     sync.Map{},
	}
	db.Statement.SQL.WriteString("INSERT INTO records (name) VALUES (?)")
	db.Statement.Vars = []interface{}{dest.Name}

	Create(&Config{})(db)

	if dest.ID != 0 {
		t.Errorf("primary key = %d, want it left alone", dest.ID)
	}
}
