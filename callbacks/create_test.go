package callbacks

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)
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
	assert.EqualValues(t, clause.Values{
		// column has value + defaultValue column has value (which should have a stable order)
		Columns: []clause.Column{{Name: "name"}, {Name: "email"}, {Name: "age"}, {Name: "id"}},
		Values: [][]interface{}{
			{"alice", "email", 18, 1},
			{"bob", "email", 19, 2},
		},
	}, values)
}
