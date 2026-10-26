package callbacks_test

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestUpdatesFromReusableModelSessionDoesNotLeakPrimaryKey(t *testing.T) {
	db := openDryRunDB(t)
	base := db.Model(new(tests.User)).Session(&gorm.Session{})

	update := func(id uint, age uint) *gorm.DB {
		tx := base.Where("id = ?", id).Updates(&tests.User{
			Model: gorm.Model{ID: id},
			Age:   age,
		})
		if model := base.Statement.Model.(*tests.User); model.ID != 0 {
			t.Errorf("reusable base model should remain zero after updating ID %d, got ID %d", id, model.ID)
		}
		return tx
	}

	first := update(111, 666)
	second := update(222, 777)
	third := update(333, 888)

	for _, tx := range []*gorm.DB{second, third} {
		if strings.Contains(tx.Statement.SQL.String(), "AND `id` = ?") {
			t.Fatalf("update leaked stale primary key condition, sql: %s; first vars: %v; current vars: %v",
				tx.Statement.SQL.String(), first.Statement.Vars, tx.Statement.Vars)
		}

		if len(tx.Statement.Vars) != len(first.Statement.Vars) {
			t.Fatalf("update should not carry extra vars, first vars: %v; current vars: %v", first.Statement.Vars, tx.Statement.Vars)
		}
	}
}

func TestUpdatesWithNonZeroModelKeepsPrimaryKeyCondition(t *testing.T) {
	db := openDryRunDB(t)

	tx := db.Model(&tests.User{Model: gorm.Model{ID: 999}}).Where("id = ?", 111).Updates(&tests.User{
		Model: gorm.Model{ID: 111},
		Age:   666,
	})

	if !strings.Contains(tx.Statement.SQL.String(), "AND `id` = ?") {
		t.Fatalf("non-zero model primary key condition was not preserved, sql: %s; vars: %v",
			tx.Statement.SQL.String(), tx.Statement.Vars)
	}

	if got := tx.Statement.Vars[len(tx.Statement.Vars)-1]; got != uint(999) {
		t.Fatalf("non-zero model primary key condition should use model ID 999, got %v; vars: %v", got, tx.Statement.Vars)
	}
}

func openDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{
		DryRun:  true,
		NowFunc: func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}

	return db
}
