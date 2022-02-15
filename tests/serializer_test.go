package tests_test

import (
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

type SerializerStruct struct {
	gorm.Model
	Name  []byte `gorm:"json"`
	Roles Roles  `gorm:"json"`
}

type Roles []string

func TestSerializerJSON(t *testing.T) {
	DB.Migrator().DropTable(&SerializerStruct{})
	if err := DB.Migrator().AutoMigrate(&SerializerStruct{}); err != nil {
		t.Fatalf("no error should happen when migrate scanner, valuer struct, got error %v", err)
	}

	data := SerializerStruct{
		Name:  []byte("jinzhu"),
		Roles: []string{"r1", "r2"},
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("failed to create data, got error %v", err)
	}

	var result SerializerStruct
	DB.First(&result, data.ID)

	AssertEqual(t, result, data)
}
