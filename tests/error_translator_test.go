package tests_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestPostgresErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	type Product struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	DB.Migrator().DropTable(&Product{})

	if err := DB.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	err := DB.Create(&Product{Name: "Milk"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated

	err = DB.Create(&Product{Name: "Milk"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated

	var product Product

	err = DB.Find(&product, "name = ?", "coffee").Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}

func TestMysqlErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		t.Skip()
	}

	type Product struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	DB.Migrator().DropTable(&Product{})

	if err := DB.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	err := DB.Create(&Product{Name: "Milk"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated

	err = DB.Create(&Product{Name: "Milk"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated

	var product Product

	err = DB.Find(&product, "name = ?", "coffee").Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}

func TestMssqlErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "mssql" {
		t.Skip()
	}

	type Product struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	DB.Migrator().DropTable(&Product{})

	if err := DB.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	err := DB.Create(&Product{Name: "Milk"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated

	err = DB.Create(&Product{Name: "Milk"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated

	var product Product

	err = DB.Find(&product, "name = ?", "coffee").Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}

func TestSqliteErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "sqlite" {
		t.Skip()
	}

	type Product struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	DB.Migrator().DropTable(&Product{})

	if err := DB.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	err := DB.Create(&Product{Name: "Milk"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated

	err = DB.Create(&Product{Name: "Milk"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated

	var product Product

	err = DB.Find(&product, "name = ?", "coffee").Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}
