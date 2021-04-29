package tests_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestPreparedStmt(t *testing.T) {
	tx := DB.Session(&gorm.Session{PrepareStmt: true})

	if _, ok := tx.ConnPool.(*gorm.PreparedStmtDB); !ok {
		t.Fatalf("should assign PreparedStatement Manager back to database when using PrepareStmt mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	txCtx := tx.WithContext(ctx)

	user := *GetUser("prepared_stmt", Config{})

	txCtx.Create(&user)

	var result1 User
	if err := txCtx.Find(&result1, user.ID).Error; err != nil {
		t.Fatalf("no error should happen but got %v", err)
	}

	time.Sleep(time.Second)

	var result2 User
	if err := tx.Find(&result2, user.ID).Error; err != nil {
		t.Fatalf("no error should happen but got %v", err)
	}

	user2 := *GetUser("prepared_stmt2", Config{})
	if err := txCtx.Create(&user2).Error; err == nil {
		t.Fatalf("should failed to create with timeout context")
	}

	if err := tx.Create(&user2).Error; err != nil {
		t.Fatalf("failed to create, got error %v", err)
	}

	var result3 User
	if err := tx.Find(&result3, user2.ID).Error; err != nil {
		t.Fatalf("no error should happen but got %v", err)
	}
}

func TestPreparedStmtFromTransaction(t *testing.T) {
	db := DB.Session(&gorm.Session{PrepareStmt: true, SkipDefaultTransaction: true})

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		t.Errorf("Failed to start transaction, got error %v\n", err)
	}

	if err := tx.Where("name=?", "zzjin").Delete(&User{}).Error; err != nil {
		tx.Rollback()
		t.Errorf("Failed to run one transaction, got error %v\n", err)
	}

	if err := tx.Create(&User{Name: "zzjin"}).Error; err != nil {
		tx.Rollback()
		t.Errorf("Failed to run one transaction, got error %v\n", err)
	}

	if err := tx.Commit().Error; err != nil {
		t.Errorf("Failed to commit transaction, got error %v\n", err)
	}

	if result := db.Where("name=?", "zzjin").Delete(&User{}); result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("Failed, got error: %v, rows affected: %v", result.Error, result.RowsAffected)
	}

	tx2 := db.Begin()
	if result := tx2.Where("name=?", "zzjin").Delete(&User{}); result.Error != nil || result.RowsAffected != 0 {
		t.Fatalf("Failed, got error: %v, rows affected: %v", result.Error, result.RowsAffected)
	}
	tx2.Commit()
}
