package tests_test

import (
	"gorm.io/gorm"
	"sync"
	"testing"
	"time"
)

func TestEaser(t *testing.T) {
	t.Run("once", func(t *testing.T) {
		db1 := DB.Unscoped()
		db1.Config.EaseQueue = &sync.Map{}

		wg := &sync.WaitGroup{}
		wg.Add(2)

		incr := 0

		testQuery := func(d *gorm.DB) {
			time.Sleep(time.Second)
			incr++
		}

		go func() {
			db1.Ease(testQuery)
			wg.Done()
		}()

		go func() {
			time.Sleep(500 * time.Millisecond)
			db1.Ease(testQuery)
			wg.Done()
		}()

		wg.Wait()

		if incr != 1 {
			t.Error("easer had to run the query only once")
		}
	})
	t.Run("twice", func(t *testing.T) {
		db1 := DB.Unscoped()
		db1.Config.EaseQueue = &sync.Map{}

		wg := &sync.WaitGroup{}
		wg.Add(2)

		incr := 0

		testQuery := func(d *gorm.DB) {
			time.Sleep(time.Second)
			incr++
		}

		go func() {
			db1.Statement.SQL.WriteString("q1")
			db1.Ease(testQuery)
			wg.Done()
		}()

		go func() {
			time.Sleep(500 * time.Millisecond)
			db1.Statement.SQL.WriteString("q2")
			db1.Ease(testQuery)
			wg.Done()
		}()

		wg.Wait()

		if incr != 2 {
			t.Error("easer had to run two separate queries")
		}
	})
}
