package tests_test

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	. "gorm.io/gorm/utils/tests"
)

func TestSchemaAccessibleFromScopes(t *testing.T) {
	users := []User{
		*GetUser("schema-scope-1", Config{}),
		*GetUser("schema-scope-2", Config{}),
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create users: %v", err)
	}

	var schema *schema.Schema
	var tableName string
	scope := func(db *gorm.DB) *gorm.DB {
		schema = db.Statement.Schema
		tableName = db.Statement.Table
		return db
	}

	var results []User
	if err := DB.Scopes(scope).Select("name", "age").Where("name like ?", "schema-scope-%").Find(&results).Error; err != nil {
		t.Errorf("failed to query users, got error: %v", err)
	}

	expects := []User{
		{Name: "schema-scope-1", Age: 18},
		{Name: "schema-scope-2", Age: 18},
	}

	if len(results) != 2 {
		t.Fatalf("invalid results length found, expects: %v, got %v", len(expects), len(results))
	}

	expectedTableName := "users"
	if tableName != expectedTableName {
		t.Errorf("invalid table name found, expects: %v, got %v", expectedTableName, tableName)
	}

	if schema == nil {
		t.Errorf("invalid schema found, expected non-nil schema")
	}
}
