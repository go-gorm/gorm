package tests_test

import (
	"testing"

	"gorm.io/gorm"
)

func TestDefaultValue(t *testing.T) {
	type Harumph struct {
		gorm.Model
		Email string `gorm:"not null;"`
		Name  string `gorm:"not null;default:foo"`
		Name2 string `gorm:"not null;default:'foo'"`
		Age   int    `gorm:"default:18"`
	}

	DB.Migrator().DropTable(&Harumph{})

	if err := DB.AutoMigrate(&Harumph{}); err != nil {
		t.Fatalf("Failed to migrate with default value, got error: %v", err)
	}

	var harumph = Harumph{Email: "hello@gorm.io"}
	if err := DB.Create(&harumph).Error; err != nil {
		t.Fatalf("Failed to create data with default value, got error: %v", err)
	} else if harumph.Name != "foo" || harumph.Name2 != "foo" || harumph.Age != 18 {
		t.Fatalf("Failed to create data with default value, got: %+v", harumph)
	}

	var result Harumph
	if err := DB.First(&result, "email = ?", "hello@gorm.io").Error; err != nil {
		t.Fatalf("Failed to find created data, got error: %v", err)
	} else if result.Name != "foo" || result.Name2 != "foo" || result.Age != 18 {
		t.Fatalf("Failed to find created data with default data, got %+v", result)
	}
}
