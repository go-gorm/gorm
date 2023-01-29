package tests_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

type City struct {
	gorm.Model
	Name string `gorm:"unique"`
}

func TestPostgresErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	DB.Migrator().DropTable(&City{})

	if err := DB.AutoMigrate(&City{}); err != nil {
		t.Fatalf("Failed to migrate cities table, got error: %v", err)
	}

	err := DB.Create(&City{Name: "Amsterdam"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated
	err = DB.Create(&City{Name: "Amsterdam"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated
	err = DB.Where("name = ?", "Kabul").First(&City{}).Error
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}

func TestMysqlErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		t.Skip()
	}

	DB.Migrator().DropTable(&City{})

	if err := DB.AutoMigrate(&City{}); err != nil {
		t.Fatalf("Failed to migrate cities table, got error: %v", err)
	}

	err := DB.Create(&City{Name: "Berlin"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated
	err = DB.Create(&City{Name: "Berlin"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated
	err = DB.Where("name = ?", "Istanbul").First(&City{}).Error
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}

func TestMssqlErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "mssql" {
		t.Skip()
	}

	DB.Migrator().DropTable(&City{})

	if err := DB.AutoMigrate(&City{}); err != nil {
		t.Fatalf("Failed to migrate cities table, got error: %v", err)
	}

	err := DB.Create(&City{Name: "Paris"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated
	err = DB.Create(&City{Name: "Paris"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated
	err = DB.Where("name = ?", "Prague").First(&City{}).Error
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}

func TestSqliteErrorTranslator(t *testing.T) {
	if DB.Dialector.Name() != "sqlite" {
		t.Skip()
	}

	DB.Migrator().DropTable(&City{})

	if err := DB.AutoMigrate(&City{}); err != nil {
		t.Fatalf("Failed to migrate cities table, got error: %v", err)
	}

	err := DB.Create(&City{Name: "Madrid"}).Error
	if err != nil {
		t.Fatalf("errors happened on create: %v", err)
	}

	// test errors to be translated
	err = DB.Create(&City{Name: "Madrid"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}

	// test default errors to not be translated
	err = DB.Where("name = ?", "Rome").First(&City{}).Error
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrRecordNotFound, err)
	}
}
