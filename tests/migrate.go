package tests

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestMigrate(t *testing.T, db *gorm.DB) {
	allModels := []interface{}{&User{}, &Account{}, &Pet{}, &Toy{}, &Company{}, &Language{}}

	db.AutoMigrate(allModels...)

	for _, m := range allModels {
		if !db.Migrator().HasTable(m) {
			t.Errorf("Failed to create table for %+v", m)
		}
	}
}
