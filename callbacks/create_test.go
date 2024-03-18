package callbacks

import (
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
