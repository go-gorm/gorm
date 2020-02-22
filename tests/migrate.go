package tests

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestMigrate(t *testing.T, db *gorm.DB) {
	allModels := []interface{}{&User{}, &Account{}, &Pet{}, &Company{}, &Toy{}, &Language{}}

	for _, m := range allModels {
		if db.Migrator().HasTable(m) {
			if err := db.Migrator().DropTable(m); err != nil {
				t.Errorf("Failed to drop table, got error %v", err)
			}
		}
	}

	if err := db.AutoMigrate(allModels...); err != nil {
		t.Errorf("Failed to auto migrate, but got error %v", err)
	}

	for _, m := range allModels {
		if !db.Migrator().HasTable(m) {
			t.Errorf("Failed to create table for %#v", m)
		}
	}
}
