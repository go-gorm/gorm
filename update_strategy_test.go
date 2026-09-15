package gorm_test

import (
	"strings"
	"testing"

	"gorm.io/gorm"
	gormtests "gorm.io/gorm/utils/tests"
)

type updateStrategyModel struct {
	ID           uint
	Always       string  `gorm:"updateStrategy:ALWAYS"`
	NotNil       *string `gorm:"updateStrategy:NOT NIL"`
	NotNilScalar int     `gorm:"updateStrategy:NOT NIL"`
	NotZero      string  `gorm:"updateStrategy:NOT ZERO"`
	Default      string  `gorm:"updateStrategy:DEFAULT"`
	Never        string  `gorm:"updateStrategy:NEVER"`
	Unspecified  string
}

func openUpdateStrategyDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(gormtests.DummyDialector{}, &gorm.Config{
		DryRun:                 true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func assertUpdateColumns(t *testing.T, sql string, included, excluded []string) {
	t.Helper()

	for _, column := range included {
		if !strings.Contains(sql, "`"+column+"`=?") {
			t.Errorf("expected column %q in SQL: %s", column, sql)
		}
	}
	for _, column := range excluded {
		if strings.Contains(sql, "`"+column+"`=?") {
			t.Errorf("did not expect column %q in SQL: %s", column, sql)
		}
	}
}

func TestUpdatesStructUpdateStrategy(t *testing.T) {
	empty := ""
	db := openUpdateStrategyDB(t)

	tx := db.Model(&updateStrategyModel{ID: 1}).Updates(updateStrategyModel{
		NotNil:       &empty,
		NotNilScalar: 0,
		NotZero:      "not-zero",
		Never:        "never",
	})
	if tx.Error != nil {
		t.Fatalf("Updates returned an error: %v", tx.Error)
	}

	assertUpdateColumns(t, tx.Statement.SQL.String(),
		[]string{"always", "not_nil", "not_nil_scalar", "not_zero"},
		[]string{"default", "never", "unspecified"},
	)

	if _, ok := tx.Statement.Settings.Load("gorm:update_strategy"); ok {
		t.Fatal("update strategy marker leaked into the returned statement")
	}
}

func TestUpdateStrategyAppliesToDirectStructUpdates(t *testing.T) {
	db := openUpdateStrategyDB(t)

	t.Run("UpdatesMap", func(t *testing.T) {
		tx := db.Model(&updateStrategyModel{ID: 1}).Updates(map[string]interface{}{
			"never": "updated",
		})
		assertUpdateColumns(t, tx.Statement.SQL.String(), []string{"never"}, nil)
	})

	t.Run("UpdateColumnsMap", func(t *testing.T) {
		tx := db.Model(&updateStrategyModel{ID: 1}).UpdateColumns(map[string]interface{}{
			"never": "updated",
		})
		assertUpdateColumns(t, tx.Statement.SQL.String(), []string{"never"}, nil)
	})

	t.Run("UpdateColumnsStruct", func(t *testing.T) {
		tx := db.Model(&updateStrategyModel{ID: 1}).UpdateColumns(updateStrategyModel{
			Never: "updated",
		})
		assertUpdateColumns(t, tx.Statement.SQL.String(), []string{"always", "not_nil_scalar"}, []string{"never"})
	})
}

func TestUpdateStrategyDoesNotApplyToSave(t *testing.T) {
	db := openUpdateStrategyDB(t)

	tx := db.Save(&updateStrategyModel{
		ID:    1,
		Never: "updated",
	})
	if tx.Error != nil {
		t.Fatalf("Save returned an error: %v", tx.Error)
	}

	assertUpdateColumns(t, tx.Statement.SQL.String(),
		[]string{"always", "not_nil", "not_nil_scalar", "not_zero", "default", "never", "unspecified"},
		nil,
	)
}
