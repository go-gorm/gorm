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
	fields := DB.NewScope(&field).Fields()
	if fields["children"].Relationship == nil || fields["category"].Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}
}
