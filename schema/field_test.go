package schema_test

import (
	"database/sql"
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
		userSchema, _, _ = schema.Parse(&tests.User{}, &sync.Map{}, schema.NamingStrategy{})
		user             = tests.User{
			Model: gorm.Model{
				ID:        10,
				CreatedAt: time.Now(),
				DeletedAt: tests.Now(),
			},
			Name:     "valuer_and_setter",
			Age:      18,
			Birthday: tests.Now(),
			Active:   true,
		}
		reflectValue = reflect.ValueOf(&user)
	)

	// test valuer
	values := map[string]interface{}{
		"name":       user.Name,
		"id":         user.ID,
		"created_at": user.CreatedAt,
		"deleted_at": user.DeletedAt,
		"age":        user.Age,
		"birthday":   user.Birthday,
		"active":     true,
	}
	checkField(t, userSchema, reflectValue, values)

	// test setter
	newValues := map[string]interface{}{
		"name":       "valuer_and_setter_2",
		"id":         2,
		"created_at": time.Now(),
		"deleted_at": tests.Now(),
		"age":        20,
		"birthday":   time.Now(),
		"active":     false,
	}

	for k, v := range newValues {
		if err := userSchema.FieldsByDBName[k].Set(reflectValue, v); err != nil {
			t.Errorf("no error should happen when assign value to field %v, but got %v", k, err)
		}
	}
	checkField(t, userSchema, reflectValue, newValues)

	// test valuer and other type
	age := myint(10)
	newValues2 := map[string]interface{}{
		"name":       sql.NullString{String: "valuer_and_setter_3", Valid: true},
		"id":         &sql.NullInt64{Int64: 3, Valid: true},
		"created_at": tests.Now(),
		"deleted_at": time.Now(),
		"age":        &age,
		"birthday":   mytime(time.Now()),
		"active":     mybool(true),
	}

	for k, v := range newValues2 {
		if err := userSchema.FieldsByDBName[k].Set(reflectValue, v); err != nil {
			t.Errorf("no error should happen when assign value to field %v, but got %v", k, err)
		}
	}
	checkField(t, userSchema, reflectValue, newValues2)
}

func TestPointerFieldValuerAndSetter(t *testing.T) {
	var (
		userSchema, _, _      = schema.Parse(&User{}, &sync.Map{}, schema.NamingStrategy{})
		name                  = "pointer_field_valuer_and_setter"
		age              uint = 18
		active                = true
		user                  = User{
			Model: &gorm.Model{
				ID:        10,
				CreatedAt: time.Now(),
				DeletedAt: tests.Now(),
			},
			Name:     &name,
			Age:      &age,
			Birthday: tests.Now(),
			Active:   &active,
		}
		reflectValue = reflect.ValueOf(&user)
	)

	// test valuer
	values := map[string]interface{}{
		"name":       user.Name,
		"id":         user.ID,
		"created_at": user.CreatedAt,
		"deleted_at": user.DeletedAt,
		"age":        user.Age,
		"birthday":   user.Birthday,
		"active":     true,
	}
	checkField(t, userSchema, reflectValue, values)

	// test setter
	newValues := map[string]interface{}{
		"name":       "valuer_and_setter_2",
		"id":         2,
		"created_at": time.Now(),
		"deleted_at": tests.Now(),
		"age":        20,
		"birthday":   time.Now(),
		"active":     false,
	}

	for k, v := range newValues {
		if err := userSchema.FieldsByDBName[k].Set(reflectValue, v); err != nil {
			t.Errorf("no error should happen when assign value to field %v, but got %v", k, err)
		}
	}
	checkField(t, userSchema, reflectValue, newValues)

	// test valuer and other type
	age2 := myint(10)
	newValues2 := map[string]interface{}{
		"name":       sql.NullString{String: "valuer_and_setter_3", Valid: true},
		"id":         &sql.NullInt64{Int64: 3, Valid: true},
		"created_at": tests.Now(),
		"deleted_at": time.Now(),
		"age":        &age2,
		"birthday":   mytime(time.Now()),
		"active":     mybool(true),
	}

	for k, v := range newValues2 {
		if err := userSchema.FieldsByDBName[k].Set(reflectValue, v); err != nil {
			t.Errorf("no error should happen when assign value to field %v, but got %v", k, err)
		}
	}
	checkField(t, userSchema, reflectValue, newValues2)
}

func TestAdvancedDataTypeValuerAndSetter(t *testing.T) {
	var (
		userSchema, _, _ = schema.Parse(&AdvancedDataTypeUser{}, &sync.Map{}, schema.NamingStrategy{})
		name             = "advanced_data_type_valuer_and_setter"
		deletedAt        = mytime(time.Now())
		isAdmin          = mybool(false)
		user             = AdvancedDataTypeUser{
			ID:           sql.NullInt64{Int64: 10, Valid: true},
			Name:         &sql.NullString{String: name, Valid: true},
			Birthday:     sql.NullTime{Time: time.Now(), Valid: true},
			RegisteredAt: mytime(time.Now()),
			DeletedAt:    &deletedAt,
			Active:       mybool(true),
			Admin:        &isAdmin,
		}
		reflectValue = reflect.ValueOf(&user)
	)

	// test valuer
	values := map[string]interface{}{
		"id":            user.ID,
		"name":          user.Name,
		"birthday":      user.Birthday,
		"registered_at": user.RegisteredAt,
		"deleted_at":    user.DeletedAt,
		"active":        user.Active,
		"admin":         user.Admin,
	}
	checkField(t, userSchema, reflectValue, values)

	// test setter
	newDeletedAt := mytime(time.Now())
	newIsAdmin := mybool(true)
	newValues := map[string]interface{}{
		"id":            sql.NullInt64{Int64: 1, Valid: true},
		"name":          &sql.NullString{String: name + "rename", Valid: true},
		"birthday":      time.Now(),
		"registered_at": mytime(time.Now()),
		"deleted_at":    &newDeletedAt,
		"active":        mybool(false),
		"admin":         &newIsAdmin,
	}

	for k, v := range newValues {
		if err := userSchema.FieldsByDBName[k].Set(reflectValue, v); err != nil {
			t.Errorf("no error should happen when assign value to field %v, but got %v", k, err)
		}
	}
	checkField(t, userSchema, reflectValue, newValues)

	newValues2 := map[string]interface{}{
		"id":            5,
		"name":          name + "rename2",
		"birthday":      time.Now(),
		"registered_at": time.Now(),
		"deleted_at":    time.Now(),
		"active":        true,
		"admin":         false,
	}

	for k, v := range newValues2 {
		if err := userSchema.FieldsByDBName[k].Set(reflectValue, v); err != nil {
			t.Errorf("no error should happen when assign value to field %v, but got %v", k, err)
		}
	}
	checkField(t, userSchema, reflectValue, newValues2)
}
