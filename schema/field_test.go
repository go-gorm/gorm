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
			t.Errorf("no error should happen when assign value to field %v", k)
		}
	}
	checkField(t, userSchema, reflectValue, newValues)
}

func TestPointerFieldValuerAndSetter(t *testing.T) {
	var (
		cacheMap      = sync.Map{}
		userSchema, _ = schema.Parse(&User{}, &cacheMap, schema.NamingStrategy{})
		name          = "pointer_field_valuer_and_setter"
		age           = 18
		active        = true
		user          = User{
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
}

type User struct {
	*gorm.Model
	Name      *string
	Age       *int
	Birthday  *time.Time
	Account   *tests.Account
	Pets      []*tests.Pet
	Toys      []tests.Toy `gorm:"polymorphic:Owner"`
	CompanyID *int
	Company   *tests.Company
	ManagerID *int
	Manager   *User
	Team      []User           `gorm:"foreignkey:ManagerID"`
	Languages []tests.Language `gorm:"many2many:UserSpeak"`
	Friends   []*User          `gorm:"many2many:user_friends"`
	Active    *bool
}
