package tests_test

import (
	"database/sql"
	"errors"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestTransaction(t *testing.T) {
	tx := DB.Begin()
	user := *GetUser("transcation", Config{})

	if err := tx.Save(&user).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err := tx.First(&User{}, "name = ?", "transcation").Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	if sqlTx, ok := tx.Statement.ConnPool.(*sql.Tx); !ok || sqlTx == nil {
		t.Fatalf("Should return the underlying sql.Tx")
	}

	tx.Rollback()

	if err := DB.First(&User{}, "name = ?", "transcation").Error; err == nil {
		t.Fatalf("Should not find record after rollback, but got %v", err)
	}

	tx2 := DB.Begin()
	user2 := *GetUser("transcation-2", Config{})
	if err := tx2.Save(&user2).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err := tx2.First(&User{}, "name = ?", "transcation-2").Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	tx2.Commit()

	if err := DB.First(&User{}, "name = ?", "transcation-2").Error; err != nil {
		t.Fatalf("Should be able to find committed record, but got %v", err)
	}
}

func TestTransactionWithBlock(t *testing.T) {
	assertPanic := func(f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("The code did not panic")
			}
		}()
		f()
	}

	// rollback
	err := DB.Transaction(func(tx *gorm.DB) error {
		user := *GetUser("transcation-block", Config{})
		if err := tx.Save(&user).Error; err != nil {
			t.Fatalf("No error should raise")
		}

		if err := tx.First(&User{}, "name = ?", user.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}

		return errors.New("the error message")
	})

	if err.Error() != "the error message" {
		t.Fatalf("Transaction return error will equal the block returns error")
	}

	if err := DB.First(&User{}, "name = ?", "transcation-block").Error; err == nil {
		t.Fatalf("Should not find record after rollback")
	}

	// commit
	DB.Transaction(func(tx *gorm.DB) error {
		user := *GetUser("transcation-block-2", Config{})
		if err := tx.Save(&user).Error; err != nil {
			t.Fatalf("No error should raise")
		}

		if err := tx.First(&User{}, "name = ?", user.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}
		return nil
	})

	if err := DB.First(&User{}, "name = ?", "transcation-block-2").Error; err != nil {
		t.Fatalf("Should be able to find committed record")
	}

	// panic will rollback
	assertPanic(func() {
		DB.Transaction(func(tx *gorm.DB) error {
			user := *GetUser("transcation-block-3", Config{})
			if err := tx.Save(&user).Error; err != nil {
				t.Fatalf("No error should raise")
			}

			if err := tx.First(&User{}, "name = ?", user.Name).Error; err != nil {
				t.Fatalf("Should find saved record")
			}

			panic("force panic")
		})
	})

	if err := DB.First(&User{}, "name = ?", "transcation-block-3").Error; err == nil {
		t.Fatalf("Should not find record after panic rollback")
	}
}

func TestTransactionRaiseErrorOnRollbackAfterCommit(t *testing.T) {
	tx := DB.Begin()
	user := User{Name: "transcation"}
	if err := tx.Save(&user).Error; err != nil {
		t.Fatalf("No error should raise")
	}

	if err := tx.Commit().Error; err != nil {
		t.Fatalf("Commit should not raise error")
	}

	if err := tx.Rollback().Error; err == nil {
		t.Fatalf("Rollback after commit should raise error")
	}
}
