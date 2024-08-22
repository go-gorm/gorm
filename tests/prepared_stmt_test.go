package tests_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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
	AssertEqual(t, len(conn.Stmts), 2)
	for _, stmt := range conn.Stmts {
		if stmt == nil {
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

func TestPreparedStmtReset(t *testing.T) {
	tx := DB.Session(&gorm.Session{PrepareStmt: true})

	user := *GetUser("prepared_stmt_reset", Config{})
	tx = tx.Create(&user)

	pdb, ok := tx.ConnPool.(*gorm.PreparedStmtDB)
	if !ok {
		t.Fatalf("should assign PreparedStatement Manager back to database when using PrepareStmt mode")
	}

	pdb.Mux.Lock()
	if len(pdb.Stmts) == 0 {
		pdb.Mux.Unlock()
		t.Fatalf("prepared stmt can not be empty")
	}
	pdb.Mux.Unlock()

	pdb.Reset()
	pdb.Mux.Lock()
	defer pdb.Mux.Unlock()
	if len(pdb.Stmts) != 0 {
		t.Fatalf("prepared stmt should be empty")
	}
}

func isUsingClosedConnError(err error) bool {
	// https://github.com/golang/go/blob/e705a2d16e4ece77e08e80c168382cdb02890f5b/src/database/sql/sql.go#L2717
	return err.Error() == "sql: statement is closed"
}

// TestPreparedStmtConcurrentReset test calling reset and executing SQL concurrently
// this test making sure that the gorm would not get a Segmentation Fault, and the only error cause by this is using a closed Stmt
func TestPreparedStmtConcurrentReset(t *testing.T) {
	name := "prepared_stmt_concurrent_reset"
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
		pdb.Reset()
	}()

	wg.Wait()

	if unexpectedError {
		t.Fatalf("should is a unexpected error")
	}
}

// TestPreparedStmtConcurrentClose test calling close and executing SQL concurrently
// for example: one goroutine found error and just close the database, and others are executing SQL
// this test making sure that the gorm would not get a Segmentation Fault,
// and the only error cause by this is using a closed Stmt or gorm.ErrInvalidDB
// and all of the goroutine must got gorm.ErrInvalidDB after database close
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
	var lastErr error
	closeValid := make(chan struct{}, loopCount)
	closeStartIdx := loopCount / 2 // close the database at the middle of the execution
	var lastRunIndex int
	var closeFinishedAt int64

	wg.Add(1)
	go func(id uint) {
		defer wg.Done()
		defer close(closeValid)
		for lastRunIndex = 1; lastRunIndex <= loopCount; lastRunIndex++ {
			if lastRunIndex == closeStartIdx {
				closeValid <- struct{}{}
			}
			var tmp User
			now := time.Now().UnixNano()
			err := tx.Session(&gorm.Session{}).First(&tmp, id).Error
			if err == nil {
				closeFinishedAt := atomic.LoadInt64(&closeFinishedAt)
				if (closeFinishedAt != 0) && (now > closeFinishedAt) {
					lastErr = errors.New("must got error after database closed")
					break
				}
				continue
			}
			lastErr = err
			break
		}
	}(user.ID)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range closeValid {
			for i := 0; i < loopCount; i++ {
				pdb.Close() // the Close method must can be call multiple times
				atomic.CompareAndSwapInt64(&closeFinishedAt, 0, time.Now().UnixNano())
			}
		}
	}()

	wg.Wait()
	var tmp User
	err = tx.Session(&gorm.Session{}).First(&tmp, user.ID).Error
	if err != gorm.ErrInvalidDB {
		t.Fatalf("must got a gorm.ErrInvalidDB while execution after db close, got %+v instead", err)
	}

	// must be error
	if lastErr != gorm.ErrInvalidDB && !isUsingClosedConnError(lastErr) {
		t.Fatalf("exp error gorm.ErrInvalidDB, got %+v instead", lastErr)
	}
	if lastRunIndex >= loopCount || lastRunIndex < closeStartIdx {
		t.Fatalf("exp loop times between (closeStartIdx %d <=) and (< loopCount %d), got %d instead", closeStartIdx, loopCount, lastRunIndex)
	}
	if pdb.Stmts != nil {
		t.Fatalf("stmts must be nil")
	}
}
