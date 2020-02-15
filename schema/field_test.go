package schema_test

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/tests"
)

func TestFieldValuerAndSetter(t *testing.T) {
	var (
		cacheMap      = sync.Map{}
		userSchema, _ = schema.Parse(&tests.User{}, &cacheMap, schema.NamingStrategy{})
		user          = tests.User{
			Model: gorm.Model{
				ID:        10,
				CreatedAt: time.Now(),
				DeletedAt: tests.Now(),
			},
			Name:     "valuer_and_setter",
			Age:      18,
			Birthday: tests.Now(),
		}
		reflectValue = reflect.ValueOf(user)
	)

	values := map[string]interface{}{
		"name":       user.Name,
		"id":         user.ID,
		"created_at": user.CreatedAt,
		"deleted_at": user.DeletedAt,
		"age":        user.Age,
		"birthday":   user.Birthday,
	}

	for k, v := range values {
		if rv := userSchema.FieldsByDBName[k].ValueOf(reflectValue); rv != v {
			t.Errorf("user's %v value should equal %+v, but got %+v", k, v, rv)
		}
	}

	newValues := map[string]interface{}{
		"name":       "valuer_and_setter_2",
		"id":         "2",
		"created_at": time.Now(),
		"deleted_at": tests.Now(),
		"age":        20,
		"birthday":   time.Now(),
	}

	for k, v := range newValues {
		if err := userSchema.FieldsByDBName[k].Set(reflectValue, v); err != nil {
			t.Errorf("no error should happen when assign value to field %v", k)
		}

		if rv := userSchema.FieldsByDBName[k].ValueOf(reflectValue); rv != v {
			t.Errorf("user's %v value should equal %+v after assign new value, but got %+v", k, v, rv)
		}
	}
}
