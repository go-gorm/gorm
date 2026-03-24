package callbacks_test

import (
	"regexp"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
)

func TestUpdateReturningUsesExplicitColumnsWhenQueryFieldsEnabled(t *testing.T) {
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}

	stmt := db.Session(&gorm.Session{DryRun: true, QueryFields: true}).
		Model(&tests.User{}).
		Clauses(clause.Returning{}).
		Where("name = ?", "update-returning-query-fields").
		Update("age", 88).Statement

	sql := stmt.SQL.String()
	if regexp.MustCompile(`(?i)RETURNING \*`).MatchString(sql) {
		t.Fatalf("SQL should not include wildcard returning when QueryFields is enabled, got %v", sql)
	}

	returningColumns := `(?i)RETURNING .*id.*created_at.*updated_at.*deleted_at.*name.*age.*birthday.*company_id.*manager_id.*active`
	if !regexp.MustCompile(returningColumns).MatchString(sql) {
		t.Fatalf("SQL should include explicit returning columns when QueryFields is enabled, got %v", sql)
	}
}
