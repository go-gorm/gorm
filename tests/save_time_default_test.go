package tests_test

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// Regression tests for go-gorm/gorm#7540: when saving a LIST of models, a
// time.Time field with a (parseable) literal default must be updated on
// conflict, just like an int field with a default is. Before the fix, the
// quoted default failed to parse, DefaultValueInterface stayed nil, and the
// column was dropped from the ON CONFLICT ... DO UPDATE SET.
type priceTimeDefault struct {
	ProductID int       `gorm:"primaryKey"`
	Price     int       `gorm:"default:1"`
	SomeTime  time.Time `gorm:"default:'2015-10-22T14:00:00Z'"`
}

// SQL-level check: the time column must appear in the update clause. Only
// meaningful on dialects that emit "DO UPDATE SET" (sqlite, postgres); MySQL
// (ON DUPLICATE KEY UPDATE) and SQL Server (MERGE) use other syntax.
func TestSaveListTimeDefaultInUpdateClause(t *testing.T) {
	if name := DB.Dialector.Name(); name != "sqlite" && name != "postgres" {
		t.Skipf("assertion targets DO UPDATE SET dialects; skipping on %s", name)
	}

	sql := DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		prices := []priceTimeDefault{
			{ProductID: 1, Price: 100, SomeTime: time.Now()},
			{ProductID: 2, Price: 150, SomeTime: time.Now()},
		}
		return tx.Save(&prices)
	})

	i := strings.Index(sql, "DO UPDATE SET")
	if i < 0 {
		t.Fatalf("expected an ON CONFLICT DO UPDATE, got:\n%s", sql)
	}
	setClause := sql[i:]
	if j := strings.Index(setClause, "RETURNING"); j >= 0 {
		setClause = setClause[:j]
	}

	if !strings.Contains(setClause, "price") {
		t.Fatalf("sanity: expected price in DO UPDATE SET, got:\n%s", sql)
	}
	if !strings.Contains(setClause, "some_time") {
		t.Fatalf("expected some_time (time.Time with default) in DO UPDATE SET, got:\n%s", sql)
	}
}

// Behavioral check (dialect-agnostic): saving a list a second time with changed
// values must update the time-with-default column on conflict.
func TestSaveListUpdatesTimeDefaultValue(t *testing.T) {
	if err := DB.Migrator().DropTable(&priceTimeDefault{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	if err := DB.AutoMigrate(&priceTimeDefault{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t0 := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	prices := []priceTimeDefault{
		{ProductID: 1, Price: 100, SomeTime: t0},
		{ProductID: 2, Price: 150, SomeTime: t0},
	}
	if err := DB.Save(&prices).Error; err != nil {
		t.Fatalf("first save: %v", err)
	}

	t1 := time.Date(2021, 6, 7, 8, 9, 10, 0, time.UTC)
	prices[0].SomeTime = t1
	prices[1].SomeTime = t1
	if err := DB.Save(&prices).Error; err != nil {
		t.Fatalf("second save (upsert): %v", err)
	}

	var got []priceTimeDefault
	if err := DB.Order("product_id").Find(&got).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	// Year comparison is robust to sqlite time round-trip / timezone shifts:
	// without the fix, some_time is never updated and stays at t0 (2020).
	for _, p := range got {
		if p.SomeTime.Year() != t1.Year() {
			t.Fatalf("product %d: some_time not updated on conflict, want year %d, got %v",
				p.ProductID, t1.Year(), p.SomeTime)
		}
	}
}
