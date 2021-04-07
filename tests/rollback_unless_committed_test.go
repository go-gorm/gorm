package tests_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestRollbackUnlessCommitted(t *testing.T) {
	type Product struct {
		gorm.Model
		Code  string
		Price uint
	}
	DB.Migrator().DropTable(&Product{})
	if err := DB.Migrator().AutoMigrate(&Product{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}
	err := DB.RollbackUnlessCommitted().Error
	if !errors.Is(err, gorm.ErrInvalidTransaction) {
		t.Fatalf("want err %v, but get err %v", gorm.ErrInvalidTransaction, err)
	}

	tx := DB.Begin()
	tx.Create(&Product{Code: "D42", Price: 100})
	err = tx.RollbackUnlessCommitted().Error
	if err != nil {
		t.Fatalf("RollbackUnlessCommitted failed, got err %v", err)
	}
	var count int64
	DB.Model(&Product{}).Where("price = ?", 100).Count(&count)
	if count != 0 {
		t.Fatalf("count should be 0, but get %d", count)
	}

	tx1 := DB.Begin()
	tx1.Create(&Product{Code: "D42", Price: 100})
	tx1.Commit()
	err = tx.RollbackUnlessCommitted().Error
	if err != nil {
		t.Fatalf("RollbackUnlessCommitted failed, got err %v", err)
	}
	DB.Model(&Product{}).Where("price = ?", 100).Count(&count)
	if count != 1 {
		t.Fatalf("count should be 1, but get %d", count)
	}
}
