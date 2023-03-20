package tests_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestReturningWithNullToZeroValues(t *testing.T) {
	dialect := DB.Dialector.Name()
	switch dialect {
	case "mysql", "sqlserver":
		// these dialects do not support the "returning" clause
		return
	default:
		// This user struct will leverage the existing users table, but override
		// the Name field to default to null.
		type user struct {
			gorm.Model
			Name string `gorm:"default:null"`
		}
		u1 := user{}

		if results := DB.Create(&u1); results.Error != nil {
			t.Fatalf("errors happened on create: %v", results.Error)
		} else if results.RowsAffected != 1 {
			t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
		} else if u1.ID == 0 {
			t.Fatalf("ID expects : not equal 0, got %v", u1.ID)
		}

		got := user{}
		results := DB.First(&got, "id = ?", u1.ID)
		if results.Error != nil {
			t.Fatalf("errors happened on first: %v", results.Error)
		} else if results.RowsAffected != 1 {
			t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
		} else if got.ID != u1.ID {
			t.Fatalf("first expects: %v, got %v", u1, got)
		}

		results = DB.Select("id, name").Find(&got)
		if results.Error != nil {
			t.Fatalf("errors happened on first: %v", results.Error)
		} else if results.RowsAffected != 1 {
			t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
		} else if got.ID != u1.ID {
			t.Fatalf("select expects: %v, got %v", u1, got)
		}

		u1.Name = "jinzhu"
		if results := DB.Save(&u1); results.Error != nil {
			t.Fatalf("errors happened on update: %v", results.Error)
		} else if results.RowsAffected != 1 {
			t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
		}

		u1 = user{} // important to reinitialize this before creating it again
		u2 := user{}
		db := DB.Session(&gorm.Session{CreateBatchSize: 10})

		if results := db.Create([]*user{&u1, &u2}); results.Error != nil {
			t.Fatalf("errors happened on create: %v", results.Error)
		} else if results.RowsAffected != 2 {
			t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
		} else if u1.ID == 0 {
			t.Fatalf("ID expects : not equal 0, got %v", u1.ID)
		} else if u2.ID == 0 {
			t.Fatalf("ID expects : not equal 0, got %v", u2.ID)
		}

		var gotUsers []user
		results = DB.Where("id in (?, ?)", u1.ID, u2.ID).Order("id asc").Select("id, name").Find(&gotUsers)
		if results.Error != nil {
			t.Fatalf("errors happened on first: %v", results.Error)
		} else if results.RowsAffected != 2 {
			t.Fatalf("rows affected expects: %v, got %v", 2, results.RowsAffected)
		} else if gotUsers[0].ID != u1.ID {
			t.Fatalf("select expects: %v, got %v", u1.ID, gotUsers[0].ID)
		} else if gotUsers[1].ID != u2.ID {
			t.Fatalf("select expects: %v, got %v", u2.ID, gotUsers[1].ID)
		}

		u1.Name = "Jinzhu"
		u2.Name = "Zhang"
		if results := DB.Save([]*user{&u1, &u2}); results.Error != nil {
			t.Fatalf("errors happened on update: %v", results.Error)
		} else if results.RowsAffected != 2 {
			t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
		}

	}
}

func TestEaserSameQueryTwice(t *testing.T) {
	db, _ := gorm.Open(tests.DummyDialector{}, &gorm.Config{
		Ease: true,
	})

	wg := &sync.WaitGroup{}
	wg.Add(2)

	var incr uint32

	testQuery := func(d *gorm.DB) {
		time.Sleep(time.Second)
		atomic.AddUint32(&incr, 1)
	}

	go func() {
		db.Ease(testQuery)
		wg.Done()
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		db.Ease(testQuery)
		wg.Done()
	}()

	wg.Wait()

	if incr != 1 {
		t.Error("easer had to run the query only once")
	}
}

func TestEaserTwoDifferentQueries(t *testing.T) {
	db, _ := gorm.Open(tests.DummyDialector{}, &gorm.Config{
		Ease: true,
	})

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	wg.Add(2)

	var incr uint32 = 0

	testQuery := func(d *gorm.DB) {
		time.Sleep(time.Second)
		atomic.AddUint32(&incr, 1)
	}

	go func() {
		mu.Lock()
		db.Statement.SQL.WriteString("q1")
		db.Ease(testQuery)
		mu.Unlock()
		wg.Done()
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		mu.Lock()
		db.Statement.SQL.WriteString("q2")
		db.Ease(testQuery)
		mu.Unlock()
		wg.Done()
	}()

	wg.Wait()

	if incr != 2 {
		t.Error("easer had to run two separate queries")
	}
}
