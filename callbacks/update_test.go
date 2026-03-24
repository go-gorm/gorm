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

	returningColumns := "(?i)RETURNING .*`users`\\.`id`.*`users`\\.`created_at`.*`users`\\.`updated_at`.*`users`\\.`deleted_at`.*`users`\\.`name`.*`users`\\.`age`.*`users`\\.`birthday`.*`users`\\.`company_id`.*`users`\\.`manager_id`.*`users`\\.`active`"
	if !regexp.MustCompile(returningColumns).MatchString(sql) {
		t.Fatalf("SQL should include explicit qualified returning columns when QueryFields is enabled, got %v", sql)
	}
}
