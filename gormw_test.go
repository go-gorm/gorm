package gorm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Employee struct {
	ID     int
	Name   string
	Salary int
}

func setupGormw(t *testing.T) (db gorm.Gormw) {
	db, err := gorm.Openw("sqlite3", filepath.Join(os.TempDir(), "gorm.db"))
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	if db == nil {
		t.Fatal("db should not be nil")
	}
	return db
}

func teardownGormw(t *testing.T, db gorm.Gormw) {
	db.Close()
	for _, err := range db.GetErrors() {
		t.Error(err)
	}
}

func TestDDL(t *testing.T) {
	db := setupGormw(t)
	db.CreateTable(&Employee{})
	if !db.HasTable("employees") {
		t.Error(`table "employees" should exist`)
	}
	db.DropTableIfExists(&Employee{})

	db.SingularTable(true)
	db.CreateTable(&Employee{})
	name := db.NewScope(&Employee{}).TableName()
	if name != "employee" {
		t.Errorf(`expected table name "employee"; got "%s"`, name)
	}
	db.DropTable(&Employee{})
	teardownGormw(t, db)
}

func TestBasicDML(t *testing.T) {
	db := setupGormw(t)
	db.CreateTable(&Employee{})

	emp := &Employee{1, "jinzhu", 1000000}
	db.Create(emp)
	emp1 := &Employee{0, "littledot", 0}
	if !db.NewRecord(emp1) {
		t.Errorf(`NewRecord() should return true`)
	}
	if !db.Where(emp1).First(emp1).RecordNotFound() {
		t.Errorf(`non-existent row should not be found`)
	}

	db.FirstOrInit(emp1, emp1)
	emp1.Salary = 1
	affected := db.Model(emp1).Updates(emp1).RowsAffected()
	if affected == 0 {
		t.Errorf(`expected 1 affected row; got %d`, affected)
	}

	if err := db.Raw("bad syntax burp").Scan(&Employee{}).Error(); err == nil {
		t.Error(`expected error; got nil`)
	}

	db.DropTable(&Employee{})
	teardownGormw(t, db)
}
