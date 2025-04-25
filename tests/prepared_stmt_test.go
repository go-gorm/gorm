package tests_test

import (
	"context"
	"errors"
	"sync"
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

func TestPreparedStmtLruFromTransaction(t *testing.T) {
	db, _ := OpenTestConnection(&gorm.Config{PrepareStmt: true, PrepareStmtMaxSize: 10, PrepareStmtTTL: 20 * time.Second})

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
	// Attempt to convert the connection pool of tx to the *gorm.PreparedStmtDB type.
	// If the conversion is successful, ok will be true and conn will be the converted object;
	// otherwise, ok will be false and conn will be nil.
	conn, ok := tx.ConnPool.(*gorm.PreparedStmtDB)
	// Get the number of statement keys stored in the PreparedStmtDB.
	lens := len(conn.Stmts.Keys())
	// Check if the number of stored statement keys is 0.
	if lens == 0 {
		// If the number is 0, it means there are no statements stored in the LRU cache.
		// The test fails and an error message is output.
		t.Fatalf("lru should not be empty")
	}
	// Wait for 40 seconds to give the statements in the cache enough time to expire.
	time.Sleep(time.Second * 40)
	// Assert whether the connection pool of tx is successfully converted to the *gorm.PreparedStmtDB type.
	AssertEqual(t, ok, true)
	// Assert whether the number of statement keys stored in the PreparedStmtDB is 0 after 40 seconds.
	// If it is not 0, it means the statements in the cache have not expired as expected.
	AssertEqual(t, len(conn.Stmts.Keys()), 0)

}

func TestPreparedStmtDeadlock(t *testing.T) {
	tx, err := OpenTestConnection(&gorm.Config{})
	AssertEqual(t, err, nil)

	sqlDB, _ := tx.DB()
	sqlDB.SetMaxOpenConns(1)

	tx = tx.Session(&gorm.Session{PrepareStmt: true})

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			user := User{Name: "jinzhu"}
			tx.Create(&user)

			var result User
			tx.First(&result)
			wg.Done()
		}()
	}
	wg.Wait()

	conn, ok := tx.ConnPool.(*gorm.PreparedStmtDB)
	AssertEqual(t, ok, true)
	AssertEqual(t, len(conn.Stmts.Keys()), 2)
	for _, stmt := range conn.Stmts.Keys() {
		if stmt == "" {
			t.Fatalf("stmt cannot bee nil")
		}
	}

	AssertEqual(t, sqlDB.Stats().InUse, 0)
}

func TestPreparedStmtInTransaction(t *testing.T) {
	user := User{Name: "jinzhu"}

	if err := DB.Transaction(func(tx *gorm.DB) error {
		tx.Session(&gorm.Session{PrepareStmt: true}).Create(&user)
		return errors.New("test")
	}); err == nil {
		t.Error(err)
	}

	var result User
	if err := DB.First(&result, user.ID).Error; err == nil {
		t.Errorf("Failed, got error: %v", err)
	}
}

func TestPreparedStmtClose(t *testing.T) {
	tx := DB.Session(&gorm.Session{PrepareStmt: true})

	user := *GetUser("prepared_stmt_close", Config{})
	tx = tx.Create(&user)

	pdb, ok := tx.ConnPool.(*gorm.PreparedStmtDB)
	if !ok {
		t.Fatalf("should assign PreparedStatement Manager back to database when using PrepareStmt mode")
	}

	pdb.Mux.Lock()
	if len(pdb.Stmts.Keys()) == 0 {
		pdb.Mux.Unlock()
		t.Fatalf("prepared stmt can not be empty")
	}
	pdb.Mux.Unlock()

	pdb.Close()
	pdb.Mux.Lock()
	defer pdb.Mux.Unlock()
	if len(pdb.Stmts.Keys()) != 0 {
		t.Fatalf("prepared stmt should be empty")
	}
}

func isUsingClosedConnError(err error) bool {
	// https://github.com/golang/go/blob/e705a2d16e4ece77e08e80c168382cdb02890f5b/src/database/sql/sql.go#L2717
	return err.Error() == "sql: statement is closed"
}

// TestPreparedStmtConcurrentClose test calling close and executing SQL concurrently
// this test making sure that the gorm would not get a Segmentation Fault, and the only error cause by this is using a closed Stmt
func TestPreparedStmtConcurrentClose(t *testing.T) {
	name := "prepared_stmt_concurrent_close"
	user := *GetUser(name, Config{})
	createTx := DB.Session(&gorm.Session{}).Create(&user)
	if createTx.Error != nil {
		t.Fatalf("failed to prepare record due to %s, test cannot be continue", createTx.Error)
	}

	// create a new connection to keep away from other tests
	tx, err := OpenTestConnection(&gorm.Config{PrepareStmt: true})
	if err != nil {
		t.Fatalf("failed to open test connection due to %s", err)
	}
	pdb, ok := tx.ConnPool.(*gorm.PreparedStmtDB)
	if !ok {
		t.Fatalf("should assign PreparedStatement Manager back to database when using PrepareStmt mode")
	}

	loopCount := 100
	var wg sync.WaitGroup
	var unexpectedError bool
	writerFinish := make(chan struct{})

	wg.Add(1)
	go func(id uint) {
		defer wg.Done()
		defer close(writerFinish)

		for j := 0; j < loopCount; j++ {
			var tmp User
			err := tx.Session(&gorm.Session{}).First(&tmp, id).Error
			if err == nil || isUsingClosedConnError(err) {
				continue
			}
			t.Errorf("failed to read user of id %d due to %s, there should not be error", id, err)
			unexpectedError = true
			break
		}
	}(user.ID)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-writerFinish
		pdb.Close()
	}()

	wg.Wait()

	if unexpectedError {
		t.Fatalf("should is a unexpected error")
	}
}
