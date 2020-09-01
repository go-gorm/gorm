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
