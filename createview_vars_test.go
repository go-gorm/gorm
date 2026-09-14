package gorm_test

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type createViewItem struct {
	ID  uint `gorm:"primarykey"`
	Txt string
}

// Migrator.CreateView used to collect the view query's vars on the shared
// m.DB.Statement: the second CreateView call on the same Migrator inherited
// the first call's vars (AddVar prefixes the stale ones), so the new view's
// `?` placeholders were filled with the OLD values. The second view then
// silently filtered on the first view's parameters.
func TestMigratorCreateViewVarsPollution(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}

	// file::memory:?cache=shared is a single process-wide database; clear its
	// state so repeated runs (go test -count=N) start from scratch
	for _, view := range []string{"cv_alpha", "cv_beta", "cv_all"} {
		if err := db.Migrator().DropView(view); err != nil {
			t.Fatalf("drop view %s failed: %v", view, err)
		}
	}
	if err := db.Migrator().DropTable(&createViewItem{}); err != nil {
		t.Fatalf("drop table failed: %v", err)
	}

	if err := db.AutoMigrate(&createViewItem{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	for _, txt := range []string{"alpha", "beta"} {
		if err := db.Create(&createViewItem{Txt: txt}).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	mig := db.Migrator() // the SAME migrator (and statement) for every call

	q1 := db.Session(&gorm.Session{NewDB: true}).Model(&createViewItem{}).Where("txt = ?", "alpha")
	if err := mig.CreateView("cv_alpha", gorm.ViewOption{Query: q1}); err != nil {
		t.Fatalf("create cv_alpha failed: %v", err)
	}

	q2 := db.Session(&gorm.Session{NewDB: true}).Model(&createViewItem{}).Where("txt = ?", "beta")
	if err := mig.CreateView("cv_beta", gorm.ViewOption{Query: q2}); err != nil {
		t.Fatalf("create cv_beta failed: %v", err)
	}

	assertViewRows := func(view string, want int64) {
		t.Helper()
		var got int64
		if err := db.Table(view).Count(&got).Error; err != nil {
			t.Fatalf("count %s failed: %v", view, err)
		}
		if got != want {
			t.Errorf("view %s: expected %d rows, got %d", view, want, got)
		}
	}
	assertViewRows("cv_alpha", 1)
	// bug: cv_beta was created with alpha's leaked var and returned the alpha row
	assertViewRows("cv_beta", 1)

	var got string
	if err := db.Raw("SELECT txt FROM cv_beta LIMIT 1").Scan(&got).Error; err != nil {
		t.Fatalf("query cv_beta failed: %v", err)
	}
	if got != "beta" {
		t.Errorf("cv_beta should filter on beta, got %q (vars leaked from the previous CreateView call)", got)
	}

	// DDL-level check: the stale value is visible in the view definition
	var ddl string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE name = 'cv_beta'").Scan(&ddl).Error; err != nil {
		t.Fatalf("read cv_beta ddl failed: %v", err)
	}
	if !strings.Contains(ddl, "beta") || strings.Contains(ddl, "alpha") {
		t.Errorf("cv_beta ddl should reference beta, got %q", ddl)
	}

	// a view with two placeholders keeps its positional substitution intact
	q3 := db.Session(&gorm.Session{NewDB: true}).Model(&createViewItem{}).Where("txt IN ?", []string{"alpha", "beta"})
	if err := mig.CreateView("cv_all", gorm.ViewOption{Query: q3}); err != nil {
		t.Fatalf("create cv_all failed: %v", err)
	}
	assertViewRows("cv_all", 2)
}
