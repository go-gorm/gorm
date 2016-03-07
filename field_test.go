package gorm_test

import (
	"testing"

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
	} else if _, ok := field.TagSettings["NOT NULL"]; !ok {
		t.Errorf("should find embedded field's tag settings")
	}
}
