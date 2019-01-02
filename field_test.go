package gorm_test

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type CalculateField struct {
	gorm.Model
	Name     string
	Children []CalculateFieldChild
	Category CalculateFieldCategory
	EmbeddedField
}

type EmbeddedField struct {
	EmbeddedName string `sql:"NOT NULL;DEFAULT:'hello'"`
}

type CalculateFieldChild struct {
	gorm.Model
	CalculateFieldID uint
	Name             string
}

type CalculateFieldCategory struct {
	gorm.Model
	CalculateFieldID uint
	Name             string
}

func TestCalculateField(t *testing.T) {
	var field CalculateField
	var scope = DB.NewScope(&field)
	if field, ok := scope.FieldByName("Children"); !ok || field.Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}

	if field, ok := scope.FieldByName("Category"); !ok || field.Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}

	if field, ok := scope.FieldByName("embedded_name"); !ok {
		t.Errorf("should find embedded field")
	} else if _, ok := field.TagSettingsGet("NOT NULL"); !ok {
		t.Errorf("should find embedded field's tag settings")
	}
}

func TestFieldSet(t *testing.T) {
	type TestFieldSetNullUUID struct {
		NullUUID uuid.NullUUID
	}
	scope := DB.NewScope(&TestFieldSetNullUUID{})
	field := scope.Fields()[0]
	err := field.Set(uuid.FromStringOrNil("3034d44a-da03-11e8-b366-4a00070b9f00"))
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := field.Field.Addr().Interface().(*uuid.NullUUID); !ok {
		t.Fatal()
	} else if !id.Valid || id.UUID.String() != "3034d44a-da03-11e8-b366-4a00070b9f00" {
		t.Fatal(id)
	}
}
