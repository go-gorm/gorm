package gorm_test

import (
	"fmt"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type relationRaceTarget struct {
	ID uint `gorm:"primaryKey"`
	// named <HolderSchema>ID so guessRelation resolves the has-many relation
	RelationRaceHolderID uint
}

type relationRaceHolder struct {
	ID uint `gorm:"primaryKey"`
	Ts []relationRaceTarget
}

// The first (lazy) parse of a schema holding a has-one/has-many association
// registers a reverse relation into the *target* schema's shared
// Relationships.Relations map, while runtime readers (preload, scan,
// association, migrator, ...) read that same map without any lock.
// Concurrently parsing the holder for the first time while querying the
// target used to trigger `fatal error: concurrent map read and map write`
// (or a DATA RACE report under -race).
//
// Run with -race to verify.
func TestRelationRegisterDataRace(t *testing.T) {
	for i := 0; i < 300; i++ {
		// fresh schema cache per iteration => Holder's lazy parse (and with it
		// the reverse relation write into relationRaceTarget's Relations map)
		// happens inside the spawned writer goroutine every round
		db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:relation_race_%d?mode=memory&cache=shared", i)), &gorm.Config{Logger: logger.Discard})
		if err != nil {
			t.Fatalf("open db failed: %v", err)
		}
		if err := db.AutoMigrate(&relationRaceTarget{}); err != nil {
			t.Fatalf("migrate failed: %v", err)
		}
		if err := db.Create(&relationRaceTarget{RelationRaceHolderID: 1}).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(2)

		// writer: first lazy parse of relationRaceHolder registers the reverse
		// relation "_relationRaceHolder_Ts" into relationRaceTarget's Relations map.
		// DryRun so the goroutine performs no SQL: cgo transitions are explicit
		// happens-before edges for the race detector, and any statement
		// execution here would order the map write before the reader's access
		// through the connection pool and mask the race.
		go func() {
			defer wg.Done()
			var n int64
			db.Session(&gorm.Session{NewDB: true, DryRun: true}).Model(&relationRaceHolder{}).Count(&n)
		}()

		// reader: Preload(clause.Associations) iterates the same Relations map
		// (callbacks/preload.go) while it is being written
		go func() {
			defer wg.Done()
			var ts []relationRaceTarget
			if err := db.Session(&gorm.Session{NewDB: true}).Preload(clause.Associations).Find(&ts).Error; err != nil {
				t.Errorf("query failed: %v", err)
			}
		}()

		wg.Wait()
	}
}
