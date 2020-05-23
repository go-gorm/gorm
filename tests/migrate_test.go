package tests_test

import (
	"math/rand"
	"testing"
	"time"

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
