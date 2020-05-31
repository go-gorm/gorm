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
		t.Fatalf("Failed to drop table, got error %v", err)
	}

	if err := DB.AutoMigrate(allModels...); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}

	for _, m := range allModels {
		if !DB.Migrator().HasTable(m) {
			t.Fatalf("Failed to create table for %#v", m)
		}
	}
}

func TestTable(t *testing.T) {
	type TableStruct struct {
		gorm.Model
		Name string
	}

	DB.Migrator().DropTable(&TableStruct{})
	DB.AutoMigrate(&TableStruct{})

	if !DB.Migrator().HasTable(&TableStruct{}) {
		t.Fatalf("should found created table")
	}

	type NewTableStruct struct {
		gorm.Model
		Name string
	}

	if err := DB.Migrator().RenameTable(&TableStruct{}, &NewTableStruct{}); err != nil {
		t.Fatalf("Failed to rename table, got error %v", err)
	}

	if !DB.Migrator().HasTable("new_table_structs") {
		t.Fatal("should found renamed table")
	}

	DB.Migrator().DropTable("new_table_structs")

	if DB.Migrator().HasTable(&NewTableStruct{}) {
		t.Fatal("should not found droped table")
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
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Got error when tried to create index: %+v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Fatalf("Failed to find index for user's name")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Fatalf("Should not find index for user's name after delete")
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Got error when tried to create index: %+v", err)
	}

	if err := DB.Migrator().RenameIndex(&IndexStruct{}, "idx_index_structs_name", "idx_users_name_1"); err != nil {
		t.Fatalf("no error should happen when rename index, but got %v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Fatalf("Should find index for user's name after rename")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "idx_users_name_1"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Fatalf("Should not find index for user's name after delete")
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
		t.Fatalf("Failed to add column, got %v", err)
	}

	if !DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "NewName") {
		t.Fatalf("Failed to find added column")
	}

	if err := DB.Table("column_structs").Migrator().DropColumn(&NewColumnStruct{}, "NewName"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "NewName") {
		t.Fatalf("Found deleted column")
	}

	if err := DB.Table("column_structs").Migrator().AddColumn(&NewColumnStruct{}, "NewName"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if err := DB.Table("column_structs").Migrator().RenameColumn(&NewColumnStruct{}, "NewName", "new_new_name"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if !DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "new_new_name") {
		t.Fatalf("Failed to found renamed column")
	}

	if err := DB.Table("column_structs").Migrator().DropColumn(&NewColumnStruct{}, "new_new_name"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "new_new_name") {
		t.Fatalf("Found deleted column")
	}
}
