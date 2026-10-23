package schema_test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNamerPreloadRegression(t *testing.T) {
	type Company struct {
		ID   int `gorm:"primaryKey;column:company_id"`
		Name string
	}
	type User struct {
		ID        int
		CompanyID int
		Company   Company `gorm:"foreignKey:CompanyID;references:ID"`
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	for _, query := range []string{
		"CREATE TABLE companies (company_id integer PRIMARY KEY, name text)",
		"CREATE TABLE users (id integer PRIMARY KEY, company_id integer)",
		"INSERT INTO companies VALUES (42, 'expected-company')",
		"INSERT INTO users VALUES (7, 42)",
	} {
		if err := db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	var user User
	if err := db.Preload("Company").First(&user, 7).Error; err != nil {
		t.Fatal(err)
	}
	if user.Company.ID != 42 || user.Company.Name != "expected-company" {
		t.Fatalf("preload followed wrong key: %#v", user)
	}
}

func TestNamerReturnedColumnScan(t *testing.T) {
	type Record struct {
		Value string `gorm:"column:VALUE"`
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{NamingStrategy: upperColumnNamer{}})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	var record Record
	if err := db.Raw("SELECT 'oracle-compatible' AS value").Scan(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.Value != "oracle-compatible" {
		t.Fatalf("column fallback lost: %#v", record)
	}
}
