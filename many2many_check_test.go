package gorm_test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMany2ManyDoesNotInheritCheckConstraints(t *testing.T) {
	type CheckedTag struct {
		Code string `gorm:"primaryKey;check:checked_tag_code,code <> ''"`
	}
	type CheckedUser struct {
		Email string       `gorm:"primaryKey;check:checked_user_email,email <> ''"`
		Tags  []CheckedTag `gorm:"many2many:checked_user_tags;foreignKey:Email;references:Code"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(&CheckedUser{}); err != nil {
		t.Fatal(err)
	}
	relation := stmt.Schema.Relationships.Relations["Tags"]
	if checks := relation.JoinTable.ParseCheckConstraints(); len(checks) != 0 {
		t.Errorf("join table inherited check constraints: %v", checks)
	}
	if len(stmt.Schema.ParseCheckConstraints()) != 1 || len(relation.FieldSchema.ParseCheckConstraints()) != 1 {
		t.Error("source table check constraints must be preserved")
	}

	if err := db.AutoMigrate(&CheckedUser{}, &CheckedTag{}); err != nil {
		t.Fatalf("migrate many-to-many tables: %v", err)
	}
	user := CheckedUser{Email: "user@example.com", Tags: []CheckedTag{{Code: "valid"}}}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create association: %v", err)
	}
	var count int64
	if err := db.Table("checked_user_tags").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected one join row, got %d, error: %v", count, err)
	}
	if err := db.Create(&CheckedUser{Email: ""}).Error; err == nil {
		t.Error("user check constraint was not enforced")
	}
	if err := db.Create(&CheckedTag{Code: ""}).Error; err == nil {
		t.Error("tag check constraint was not enforced")
	}
}
