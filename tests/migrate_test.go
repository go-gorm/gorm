package tests_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	. "github.com/jinzhu/gorm/tests"
)

func TestMigrate(t *testing.T) {
	allModels := []interface{}{&User{}, &Account{}, &Pet{}, &Company{}, &Toy{}, &Language{}}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(allModels), func(i, j int) { allModels[i], allModels[j] = allModels[j], allModels[i] })

	if err := DB.Migrator().DropTable(allModels...); err != nil {
		t.Errorf("Failed to drop table, got error %v", err)
	}

	if err := DB.AutoMigrate(allModels...); err != nil {
		t.Errorf("Failed to auto migrate, but got error %v", err)
	}

	for _, m := range allModels {
		if !DB.Migrator().HasTable(m) {
			t.Errorf("Failed to create table for %#v", m)
		}
	}
}

func TestIndexes(t *testing.T) {
	type IndexStruct struct {
		gorm.Model
		Name string `gorm:"size:255;index"`
	}

	DB.Migrator().DropTable(&IndexStruct{})
	DB.AutoMigrate(&IndexStruct{})

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "Name"); err != nil {
		t.Errorf("Failed to drop index for user's name, got err %v", err)
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Errorf("Failed to find index for user's name")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "Name"); err != nil {
		t.Errorf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Errorf("Should not find index for user's name after delete")
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	if err := DB.Migrator().RenameIndex(&IndexStruct{}, "idx_index_structs_name", "idx_users_name_1"); err != nil {
		t.Errorf("no error should happen when rename index, but got %v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Errorf("Should find index for user's name after rename")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "idx_users_name_1"); err != nil {
		t.Errorf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Errorf("Should not find index for user's name after delete")
	}
}

func TestColumns(t *testing.T) {
	type ColumnStruct struct {
		gorm.Model
		Name string
	}

	DB.Migrator().DropTable(&ColumnStruct{})

	if err := DB.AutoMigrate(&ColumnStruct{}); err != nil {
		t.Errorf("Failed to migrate, got %v", err)
	}

	type NewColumnStruct struct {
		gorm.Model
		Name    string
		NewName string
	}

	if err := DB.Table("column_structs").Migrator().AddColumn(&NewColumnStruct{}, "NewName"); err != nil {
		t.Errorf("Failed to add column, got %v", err)
	}

	if !DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "NewName") {
		t.Errorf("Failed to find added column")
	}

	if err := DB.Table("column_structs").Migrator().DropColumn(&NewColumnStruct{}, "NewName"); err != nil {
		t.Errorf("Failed to add column, got %v", err)
	}

	if DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "NewName") {
		t.Errorf("Found deleted column")
	}
}
