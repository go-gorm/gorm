package gorm_test

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestFindInBatchesBatchSize(t *testing.T) {
	for _, test := range []struct {
		name        string
		batchSize   int
		wantError   bool
		wantQueries int
		wantBatches int
	}{
		{name: "zero", batchSize: 0, wantError: true},
		{name: "negative", batchSize: -1, wantError: true},
		{name: "positive", batchSize: 1, wantQueries: 2, wantBatches: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, err := gorm.Open(tests.DummyDialector{}, nil)
			if err != nil {
				t.Fatal(err)
			}

			queries := 0
			if err := db.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
				queries++
				var users []tests.User
				if queries == 1 {
					users = []tests.User{{Model: gorm.Model{ID: 1}}}
				}
				*tx.Statement.Dest.(*[]tests.User) = users
				tx.RowsAffected = int64(len(users))
			}); err != nil {
				t.Fatal(err)
			}

			defer func() {
				if value := recover(); value != nil {
					t.Errorf("FindInBatches panicked: %v", value)
				}
			}()

			batches := 0
			var users []tests.User
			result := db.FindInBatches(&users, test.batchSize, func(*gorm.DB, int) error {
				batches++
				return nil
			})
			if (result.Error != nil) != test.wantError {
				t.Errorf("expected error %v, got %v", test.wantError, result.Error)
			}
			if queries != test.wantQueries {
				t.Errorf("expected %d queries, got %d", test.wantQueries, queries)
			}
			if batches != test.wantBatches {
				t.Errorf("expected %d batches, got %d", test.wantBatches, batches)
			}
		})
	}
}
