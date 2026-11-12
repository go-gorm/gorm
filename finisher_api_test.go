package gorm

import (
	"errors"
	"testing"
)

func TestFindInBatchesRejectsNonPositiveBatchSize(t *testing.T) {
	for _, size := range []int{0, -1} {
		db := &DB{Config: &Config{}}
		called := false
		var dest []int
		result := db.FindInBatches(&dest, size, func(tx *DB, batch int) error {
			called = true
			return nil
		})
		if !errors.Is(result.Error, ErrInvalidBatchSize) {
			t.Fatalf("batchSize=%d: error = %v, want ErrInvalidBatchSize", size, result.Error)
		}
		if called {
			t.Fatalf("batchSize=%d: callback should not be invoked", size)
		}
	}
}
