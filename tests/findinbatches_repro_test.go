package tests_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

// Regression test for go-gorm/gorm#7737:
// Model(A).FindInBatches(&[]B) must page correctly when B embeds A but has a
// different field layout. Before the fix, the batch cursor read the primary key
// through the Model's schema (A) against B's rows, yielding the wrong field
// (panic on older versions, "unsupported type" SQL error on current master),
// which broke pagination across batches.

type userExtended struct {
	User
	ExtraFoo int64
}

func TestFindInBatchesModelDestMismatch(t *testing.T) {
	const total = 5
	created := make(map[uint]bool, total)
	for i := 0; i < total; i++ {
		u := User{Name: "fib_7737"}
		if err := DB.Create(&u).Error; err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		created[u.ID] = true
	}

	// query sets a Model that differs from the dest element type, and selects an
	// extra column that only exists on the dest — forcing the schema mismatch.
	query := func() *gorm.DB {
		return DB.Model(&User{}).Where("name = ?", "fib_7737").Select("*, 42 AS extra_foo")
	}

	// assertPagedOnce verifies every created row was visited exactly once across
	// batches (no duplicate, no skip) — i.e. the primary-key cursor is correct.
	assertPagedOnce := func(t *testing.T, seen map[uint]bool, rowsAffected int64) {
		t.Helper()
		if int(rowsAffected) != total {
			t.Fatalf("expected RowsAffected=%d, got %d", total, rowsAffected)
		}
		if len(seen) != total {
			t.Fatalf("expected to page over %d unique rows, saw %d", total, len(seen))
		}
		for id := range created {
			if !seen[id] {
				t.Fatalf("row id %d was skipped by the cursor", id)
			}
		}
	}

	// batchSize (2) < total (5) forces the primary-key cursor path across batches.
	t.Run("slice of struct", func(t *testing.T) {
		var batch []userExtended
		seen := make(map[uint]bool, total)
		result := query().FindInBatches(&batch, 2, func(tx *gorm.DB, _ int) error {
			for _, r := range batch {
				if r.ExtraFoo != 42 {
					return fmt.Errorf("ExtraFoo not scanned into dest: got %d", r.ExtraFoo)
				}
				if seen[r.ID] {
					return fmt.Errorf("row id %d visited twice (broken cursor)", r.ID)
				}
				seen[r.ID] = true
			}
			return nil
		})
		if result.Error != nil {
			t.Fatalf("FindInBatches returned error: %v", result.Error)
		}
		assertPagedOnce(t, seen, result.RowsAffected)
	})

	t.Run("slice of pointer", func(t *testing.T) {
		var batch []*userExtended
		seen := make(map[uint]bool, total)
		result := query().FindInBatches(&batch, 2, func(tx *gorm.DB, _ int) error {
			for _, r := range batch {
				if seen[r.ID] {
					return fmt.Errorf("row id %d visited twice (broken cursor)", r.ID)
				}
				seen[r.ID] = true
			}
			return nil
		})
		if result.Error != nil {
			t.Fatalf("FindInBatches returned error: %v", result.Error)
		}
		assertPagedOnce(t, seen, result.RowsAffected)
	})
}
