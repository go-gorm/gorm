package tests

import (
	"math/rand"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestMigrate(t *testing.T, db *gorm.DB) {
	allModels := []interface{}{&User{}, &Account{}, &Pet{}, &Company{}, &Toy{}, &Language{}}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(allModels), func(i, j int) { allModels[i], allModels[j] = allModels[j], allModels[i] })

	if err := db.Migrator().DropTable(allModels...); err != nil {
		t.Errorf("Failed to drop table, got error %v", err)
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
