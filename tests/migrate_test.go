package tests_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"github.com/brucewangviki/gorm"
	"github.com/brucewangviki/gorm/schema"
	. "github.com/brucewangviki/gorm/utils/tests"
)

func TestMigrate(t *testing.T) {
	allModels := []interface{}{&User{}, &Account{}, &Pet{}, &Company{}, &Toy{}, &Language{}}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(allModels), func(i, j int) { allModels[i], allModels[j] = allModels[j], allModels[i] })
	DB.Migrator().DropTable("user_speaks", "user_friends", "ccc")

	if err := DB.Migrator().DropTable(allModels...); err != nil {
		t.Fatalf("Failed to drop table, got error %v", err)
	}

	if err := DB.AutoMigrate(allModels...); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}

	if tables, err := DB.Migrator().GetTables(); err != nil {
		t.Fatalf("Failed to get database all tables, but got error %v", err)
	} else {
		for _, t1 := range []string{"users", "accounts", "pets", "companies", "toys", "languages"} {
			hasTable := false
			for _, t2 := range tables {
				if t2 == t1 {
					hasTable = true
					break
				}
			}
			if !hasTable {
				t.Fatalf("Failed to get table %v when GetTables", t1)
			}
		}
	}

	for _, m := range allModels {
		if !DB.Migrator().HasTable(m) {
			t.Fatalf("Failed to create table for %#v", m)
		}
	}

	DB.Scopes(func(db *gorm.DB) *gorm.DB {
		return db.Table("ccc")
	}).Migrator().CreateTable(&Company{})

	if !DB.Migrator().HasTable("ccc") {
		t.Errorf("failed to create table ccc")
	}

	for _, indexes := range [][2]string{
		{"user_speaks", "fk_user_speaks_user"},
		{"user_speaks", "fk_user_speaks_language"},
		{"user_friends", "fk_user_friends_user"},
		{"user_friends", "fk_user_friends_friends"},
		{"accounts", "fk_users_account"},
		{"users", "fk_users_team"},
		{"users", "fk_users_company"},
	} {
		if !DB.Migrator().HasConstraint(indexes[0], indexes[1]) {
			t.Fatalf("Failed to find index for many2many for %v %v", indexes[0], indexes[1])
		}
	}
}

func TestAutoMigrateSelfReferential(t *testing.T) {
	type MigratePerson struct {
		ID        uint
		Name      string
		ManagerID *uint
		Manager   *MigratePerson
	}

	DB.Migrator().DropTable(&MigratePerson{})

	if err := DB.AutoMigrate(&MigratePerson{}); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}

	if !DB.Migrator().HasConstraint("migrate_people", "fk_migrate_people_manager") {
		t.Fatalf("Failed to find has one constraint between people and managers")
	}
}

func TestSmartMigrateColumn(t *testing.T) {
	fullSupported := map[string]bool{"mysql": true, "postgres": true}[DB.Dialector.Name()]

	type UserMigrateColumn struct {
		ID       uint
		Name     string
		Salary   float64
		Birthday time.Time `gorm:"precision:4"`
	}

	DB.Migrator().DropTable(&UserMigrateColumn{})

	DB.AutoMigrate(&UserMigrateColumn{})

	type UserMigrateColumn2 struct {
		ID                  uint
		Name                string    `gorm:"size:128"`
		Salary              float64   `gorm:"precision:2"`
		Birthday            time.Time `gorm:"precision:2"`
		NameIgnoreMigration string    `gorm:"size:100"`
	}

	if err := DB.Table("user_migrate_columns").AutoMigrate(&UserMigrateColumn2{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err := DB.Table("user_migrate_columns").Migrator().ColumnTypes(&UserMigrateColumn{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}

	for _, columnType := range columnTypes {
		switch columnType.Name() {
		case "name":
			if length, _ := columnType.Length(); (fullSupported || length != 0) && length != 128 {
				t.Fatalf("name's length should be 128, but got %v", length)
			}
		case "salary":
			if precision, o, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 2 {
				t.Fatalf("salary's precision should be 2, but got %v %v", precision, o)
			}
		case "birthday":
			if precision, _, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 2 {
				t.Fatalf("birthday's precision should be 2, but got %v", precision)
			}
		}
	}

	type UserMigrateColumn3 struct {
		ID                  uint
		Name                string    `gorm:"size:256"`
		Salary              float64   `gorm:"precision:3"`
		Birthday            time.Time `gorm:"precision:3"`
		NameIgnoreMigration string    `gorm:"size:128;-:migration"`
	}

	if err := DB.Table("user_migrate_columns").AutoMigrate(&UserMigrateColumn3{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err = DB.Table("user_migrate_columns").Migrator().ColumnTypes(&UserMigrateColumn{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}

	for _, columnType := range columnTypes {
		switch columnType.Name() {
		case "name":
			if length, _ := columnType.Length(); (fullSupported || length != 0) && length != 256 {
				t.Fatalf("name's length should be 128, but got %v", length)
			}
		case "salary":
			if precision, _, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 3 {
				t.Fatalf("salary's precision should be 2, but got %v", precision)
			}
		case "birthday":
			if precision, _, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 3 {
				t.Fatalf("birthday's precision should be 2, but got %v", precision)
			}
		case "name_ignore_migration":
			if length, _ := columnType.Length(); (fullSupported || length != 0) && length != 100 {
				t.Fatalf("name_ignore_migration's length should still be 100 but got %v", length)
			}
		}
	}
}

func TestMigrateWithColumnComment(t *testing.T) {
	type UserWithColumnComment struct {
		gorm.Model
		Name string `gorm:"size:111;comment:this is a 字段"`
	}

	if err := DB.Migrator().DropTable(&UserWithColumnComment{}); err != nil {
		t.Fatalf("Failed to drop table, got error %v", err)
	}

	if err := DB.AutoMigrate(&UserWithColumnComment{}); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}
}

func TestMigrateWithIndexComment(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		t.Skip()
	}

	type UserWithIndexComment struct {
		gorm.Model
		Name string `gorm:"size:111;index:,comment:这是一个index"`
	}

	if err := DB.Migrator().DropTable(&UserWithIndexComment{}); err != nil {
		t.Fatalf("Failed to drop table, got error %v", err)
	}

	if err := DB.AutoMigrate(&UserWithIndexComment{}); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}
}

func TestMigrateWithUniqueIndex(t *testing.T) {
	type UserWithUniqueIndex struct {
		ID   int
		Name string    `gorm:"size:20;index:idx_name,unique"`
		Date time.Time `gorm:"index:idx_name,unique"`
	}

	DB.Migrator().DropTable(&UserWithUniqueIndex{})
	if err := DB.AutoMigrate(&UserWithUniqueIndex{}); err != nil {
		t.Fatalf("failed to migrate, got %v", err)
	}

	if !DB.Migrator().HasIndex(&UserWithUniqueIndex{}, "idx_name") {
		t.Errorf("Failed to find created index")
	}
}

func TestMigrateTable(t *testing.T) {
	type TableStruct struct {
		gorm.Model
		Name string
	}

	DB.Migrator().DropTable(&TableStruct{})
	DB.AutoMigrate(&TableStruct{})

	if !DB.Migrator().HasTable(&TableStruct{}) {
		t.Fatalf("should found created table")
	}

	type NewTableStruct struct {
		gorm.Model
		Name string
	}

	if err := DB.Migrator().RenameTable(&TableStruct{}, &NewTableStruct{}); err != nil {
		t.Fatalf("Failed to rename table, got error %v", err)
	}

	if !DB.Migrator().HasTable("new_table_structs") {
		t.Fatal("should found renamed table")
	}

	DB.Migrator().DropTable("new_table_structs")

	if DB.Migrator().HasTable(&NewTableStruct{}) {
		t.Fatal("should not found dropped table")
	}
}

func TestMigrateWithQuotedIndex(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		t.Skip()
	}

	type QuotedIndexStruct struct {
		gorm.Model
		Name string `gorm:"size:255;index:AS"` // AS is one of MySQL reserved words
	}

	if err := DB.Migrator().DropTable(&QuotedIndexStruct{}); err != nil {
		t.Fatalf("Failed to drop table, got error %v", err)
	}

	if err := DB.AutoMigrate(&QuotedIndexStruct{}); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}
}

func TestMigrateIndexes(t *testing.T) {
	type IndexStruct struct {
		gorm.Model
		Name string `gorm:"size:255;index"`
	}

	DB.Migrator().DropTable(&IndexStruct{})
	DB.AutoMigrate(&IndexStruct{})

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Got error when tried to create index: %+v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Fatalf("Failed to find index for user's name")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Fatalf("Should not find index for user's name after delete")
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Got error when tried to create index: %+v", err)
	}

	if err := DB.Migrator().RenameIndex(&IndexStruct{}, "idx_index_structs_name", "idx_users_name_1"); err != nil {
		t.Fatalf("no error should happen when rename index, but got %v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Fatalf("Should find index for user's name after rename")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "idx_users_name_1"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Fatalf("Should not find index for user's name after delete")
	}
}

func TestMigrateColumns(t *testing.T) {
	sqlite := DB.Dialector.Name() == "sqlite"
	sqlserver := DB.Dialector.Name() == "sqlserver"

	type ColumnStruct struct {
		gorm.Model
		Name  string
		Age   int    `gorm:"default:18;comment:my age"`
		Code  string `gorm:"unique;comment:my code;"`
		Code2 string
		Code3 string `gorm:"unique"`
	}

	DB.Migrator().DropTable(&ColumnStruct{})

	if err := DB.AutoMigrate(&ColumnStruct{}); err != nil {
		t.Errorf("Failed to migrate, got %v", err)
	}

	type ColumnStruct2 struct {
		gorm.Model
		Name  string `gorm:"size:100"`
		Code  string `gorm:"unique;comment:my code2;default:hello"`
		Code2 string `gorm:"unique"`
		// Code3 string
	}

	if err := DB.Table("column_structs").Migrator().AlterColumn(&ColumnStruct{}, "Name"); err != nil {
		t.Fatalf("no error should happened when alter column, but got %v", err)
	}

	if err := DB.Table("column_structs").AutoMigrate(&ColumnStruct2{}); err != nil {
		t.Fatalf("no error should happened when auto migrate column, but got %v", err)
	}

	if columnTypes, err := DB.Migrator().ColumnTypes(&ColumnStruct{}); err != nil {
		t.Fatalf("no error should returns for ColumnTypes")
	} else {
		stmt := &gorm.Statement{DB: DB}
		stmt.Parse(&ColumnStruct2{})

		for _, columnType := range columnTypes {
			switch columnType.Name() {
			case "id":
				if v, ok := columnType.PrimaryKey(); !ok || !v {
					t.Fatalf("column id primary key should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				}
			case "name":
				dataType := DB.Dialector.DataTypeOf(stmt.Schema.LookUpField(columnType.Name()))
				if !strings.Contains(strings.ToUpper(dataType), strings.ToUpper(columnType.DatabaseTypeName())) {
					t.Fatalf("column name type should be correct, name: %v, length: %v, expects: %v, column: %#v", columnType.Name(), columnType.DatabaseTypeName(), dataType, columnType)
				}
				if length, ok := columnType.Length(); !sqlite && (!ok || length != 100) {
					t.Fatalf("column name length should be correct, name: %v, length: %v, expects: %v, column: %#v", columnType.Name(), length, 100, columnType)
				}
			case "age":
				if v, ok := columnType.DefaultValue(); !ok || v != "18" {
					t.Fatalf("column age default value should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				}
				if v, ok := columnType.Comment(); !sqlite && !sqlserver && (!ok || v != "my age") {
					t.Fatalf("column age comment should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				}
			case "code":
				if v, ok := columnType.Unique(); !ok || !v {
					t.Fatalf("column code unique should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				}
				if v, ok := columnType.DefaultValue(); !sqlserver && (!ok || v != "hello") {
					t.Fatalf("column code default value should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				}
				if v, ok := columnType.Comment(); !sqlite && !sqlserver && (!ok || v != "my code2") {
					t.Fatalf("column code comment should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				}
			case "code2":
				if v, ok := columnType.Unique(); !sqlserver && (!ok || !v) {
					t.Fatalf("column code2 unique should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				}
			case "code3":
				// TODO
				// if v, ok := columnType.Unique(); !ok || v {
				// 	t.Fatalf("column code unique should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				// }
			}
		}
	}

	type NewColumnStruct struct {
		gorm.Model
		Name    string
		NewName string
	}

	if err := DB.Table("column_structs").Migrator().AddColumn(&NewColumnStruct{}, "NewName"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if !DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "NewName") {
		t.Fatalf("Failed to find added column")
	}

	if err := DB.Table("column_structs").Migrator().DropColumn(&NewColumnStruct{}, "NewName"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "NewName") {
		t.Fatalf("Found deleted column")
	}

	if err := DB.Table("column_structs").Migrator().AddColumn(&NewColumnStruct{}, "NewName"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if err := DB.Table("column_structs").Migrator().RenameColumn(&NewColumnStruct{}, "NewName", "new_new_name"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if !DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "new_new_name") {
		t.Fatalf("Failed to found renamed column")
	}

	if err := DB.Table("column_structs").Migrator().DropColumn(&NewColumnStruct{}, "new_new_name"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if DB.Table("column_structs").Migrator().HasColumn(&NewColumnStruct{}, "new_new_name") {
		t.Fatalf("Found deleted column")
	}
}

func TestMigrateConstraint(t *testing.T) {
	names := []string{"Account", "fk_users_account", "Pets", "fk_users_pets", "Company", "fk_users_company", "Team", "fk_users_team", "Languages", "fk_users_languages"}

	for _, name := range names {
		if !DB.Migrator().HasConstraint(&User{}, name) {
			DB.Migrator().CreateConstraint(&User{}, name)
		}

		if err := DB.Migrator().DropConstraint(&User{}, name); err != nil {
			t.Fatalf("failed to drop constraint %v, got error %v", name, err)
		}

		if DB.Migrator().HasConstraint(&User{}, name) {
			t.Fatalf("constraint %v should been deleted", name)
		}

		if err := DB.Migrator().CreateConstraint(&User{}, name); err != nil {
			t.Fatalf("failed to create constraint %v, got error %v", name, err)
		}

		if !DB.Migrator().HasConstraint(&User{}, name) {
			t.Fatalf("failed to found constraint %v", name)
		}
	}
}

type DynamicUser struct {
	gorm.Model
	Name      string
	CompanyID string `gorm:"index"`
}

// To test auto migrate crate indexes for dynamic table name
// https://github.com/go-gorm/gorm/issues/4752
func TestMigrateIndexesWithDynamicTableName(t *testing.T) {
	// Create primary table
	if err := DB.AutoMigrate(&DynamicUser{}); err != nil {
		t.Fatalf("AutoMigrate create table error: %#v", err)
	}

	// Create sub tables
	for _, v := range []string{"01", "02", "03"} {
		tableName := "dynamic_users_" + v
		m := DB.Scopes(func(db *gorm.DB) *gorm.DB {
			return db.Table(tableName)
		}).Migrator()

		if err := m.AutoMigrate(&DynamicUser{}); err != nil {
			t.Fatalf("AutoMigrate create table error: %#v", err)
		}

		if !m.HasTable(tableName) {
			t.Fatalf("AutoMigrate expected %#v exist, but not.", tableName)
		}

		if !m.HasIndex(&DynamicUser{}, "CompanyID") {
			t.Fatalf("Should have index on %s", "CompanyI.")
		}

		if !m.HasIndex(&DynamicUser{}, "DeletedAt") {
			t.Fatalf("Should have index on deleted_at.")
		}
	}
}

// check column order after migration, flaky test
// https://github.com/go-gorm/gorm/issues/4351
func TestMigrateColumnOrder(t *testing.T) {
	type UserMigrateColumn struct {
		ID uint
	}
	DB.Migrator().DropTable(&UserMigrateColumn{})
	DB.AutoMigrate(&UserMigrateColumn{})

	type UserMigrateColumn2 struct {
		ID  uint
		F1  string
		F2  string
		F3  string
		F4  string
		F5  string
		F6  string
		F7  string
		F8  string
		F9  string
		F10 string
		F11 string
		F12 string
		F13 string
		F14 string
		F15 string
		F16 string
		F17 string
		F18 string
		F19 string
		F20 string
		F21 string
		F22 string
		F23 string
		F24 string
		F25 string
		F26 string
		F27 string
		F28 string
		F29 string
		F30 string
		F31 string
		F32 string
		F33 string
		F34 string
		F35 string
	}
	if err := DB.Table("user_migrate_columns").AutoMigrate(&UserMigrateColumn2{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err := DB.Table("user_migrate_columns").Migrator().ColumnTypes(&UserMigrateColumn2{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}
	typ := reflect.Indirect(reflect.ValueOf(&UserMigrateColumn2{})).Type()
	numField := typ.NumField()
	if numField != len(columnTypes) {
		t.Fatalf("column's number not match struct and ddl, %d != %d", numField, len(columnTypes))
	}
	namer := schema.NamingStrategy{}
	for i := 0; i < numField; i++ {
		expectName := namer.ColumnName("", typ.Field(i).Name)
		if columnTypes[i].Name() != expectName {
			t.Fatalf("column order not match struct and ddl, idx %d: %s != %s",
				i, columnTypes[i].Name(), expectName)
		}
	}
}

// https://github.com/go-gorm/gorm/issues/5047
func TestMigrateSerialColumn(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	type Event struct {
		ID  uint `gorm:"primarykey"`
		UID uint32
	}

	type Event1 struct {
		ID  uint   `gorm:"primarykey"`
		UID uint32 `gorm:"not null;autoIncrement"`
	}

	type Event2 struct {
		ID  uint   `gorm:"primarykey"`
		UID uint16 `gorm:"not null;autoIncrement"`
	}

	var err error
	err = DB.Migrator().DropTable(&Event{})
	if err != nil {
		t.Errorf("DropTable err:%v", err)
	}

	// create sequence
	err = DB.Table("events").AutoMigrate(&Event1{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	// delete sequence
	err = DB.Table("events").AutoMigrate(&Event{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	// update sequence
	err = DB.Table("events").AutoMigrate(&Event1{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}
	err = DB.Table("events").AutoMigrate(&Event2{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	DB.Table("events").Save(&Event2{})
	DB.Table("events").Save(&Event2{})
	DB.Table("events").Save(&Event2{})

	events := make([]*Event, 0)
	DB.Table("events").Find(&events)

	AssertEqual(t, 3, len(events))
	for _, v := range events {
		AssertEqual(t, v.ID, v.UID)
	}
}

// https://github.com/go-gorm/gorm/issues/5300
func TestMigrateWithSpecialName(t *testing.T) {
	var err error
	err = DB.AutoMigrate(&Coupon{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}
	err = DB.Table("coupon_product_1").AutoMigrate(&CouponProduct{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}
	err = DB.Table("coupon_product_2").AutoMigrate(&CouponProduct{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	AssertEqual(t, true, DB.Migrator().HasTable("coupons"))
	AssertEqual(t, true, DB.Migrator().HasTable("coupon_product_1"))
	AssertEqual(t, true, DB.Migrator().HasTable("coupon_product_2"))
}

// https://github.com/go-gorm/gorm/issues/5320
func TestPrimarykeyID(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	type MissPKLanguage struct {
		ID   string `gorm:"type:uuid;default:uuid_generate_v4()"`
		Name string
	}

	type MissPKUser struct {
		ID              string           `gorm:"type:uuid;default:uuid_generate_v4()"`
		MissPKLanguages []MissPKLanguage `gorm:"many2many:miss_pk_user_languages;"`
	}

	var err error
	err = DB.Migrator().DropTable(&MissPKUser{}, &MissPKLanguage{})
	if err != nil {
		t.Fatalf("DropTable err:%v", err)
	}

	DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)

	err = DB.AutoMigrate(&MissPKUser{}, &MissPKLanguage{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	// patch
	err = DB.AutoMigrate(&MissPKUser{}, &MissPKLanguage{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}
}

func TestUniqueColumn(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		return
	}

	type UniqueTest struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique"`
	}

	type UniqueTest2 struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique;default:NULL"`
	}

	type UniqueTest3 struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique;default:''"`
	}

	type UniqueTest4 struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique;default:'123'"`
	}

	var err error
	err = DB.Migrator().DropTable(&UniqueTest{})
	if err != nil {
		t.Errorf("DropTable err:%v", err)
	}

	err = DB.AutoMigrate(&UniqueTest{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	// null -> null
	err = DB.AutoMigrate(&UniqueTest{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err := findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok := ct.DefaultValue()
	AssertEqual(t, "", value)
	AssertEqual(t, false, ok)

	// null -> null
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest2{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	// not trigger alert column
	AssertEqual(t, true, DB.Migrator().HasIndex(&UniqueTest{}, "name"))
	AssertEqual(t, false, DB.Migrator().HasIndex(&UniqueTest{}, "name_1"))
	AssertEqual(t, false, DB.Migrator().HasIndex(&UniqueTest{}, "name_2"))

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	AssertEqual(t, "", value)
	AssertEqual(t, false, ok)

	// null -> empty string
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest3{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	AssertEqual(t, "", value)
	AssertEqual(t, true, ok)

	//  empty string -> 123
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest4{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	AssertEqual(t, "123", value)
	AssertEqual(t, true, ok)

	//  123 -> null
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest2{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	AssertEqual(t, "", value)
	AssertEqual(t, false, ok)
}

func findColumnType(dest interface{}, columnName string) (
	foundColumn gorm.ColumnType, err error,
) {
	columnTypes, err := DB.Migrator().ColumnTypes(dest)
	if err != nil {
		err = fmt.Errorf("ColumnTypes err:%v", err)
		return
	}

	for _, c := range columnTypes {
		if c.Name() == columnName {
			foundColumn = c
			break
		}
	}
	return
}

func TestInvalidCachedPlan(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	db, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{})
	if err != nil {
		t.Errorf("Open err:%v", err)
	}

	type Object1 struct{}
	type Object2 struct {
		Field1 string
	}
	type Object3 struct {
		Field2 string
	}
	db.Migrator().DropTable("objects")

	err = db.Table("objects").AutoMigrate(&Object1{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	err = db.Table("objects").AutoMigrate(&Object2{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	err = db.Table("objects").AutoMigrate(&Object3{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}
}

func TestDifferentTypeWithoutDeclaredLength(t *testing.T) {
	type DiffType struct {
		ID   uint
		Name string `gorm:"type:varchar(20)"`
	}

	type DiffType1 struct {
		ID   uint
		Name string `gorm:"type:text"`
	}

	var err error
	DB.Migrator().DropTable(&DiffType{})

	err = DB.AutoMigrate(&DiffType{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	ct, err := findColumnType(&DiffType{}, "name")
	if err != nil {
		t.Errorf("findColumnType err:%v", err)
	}

	AssertEqual(t, "varchar", strings.ToLower(ct.DatabaseTypeName()))

	err = DB.Table("diff_types").AutoMigrate(&DiffType1{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&DiffType{}, "name")
	if err != nil {
		t.Errorf("findColumnType err:%v", err)
	}

	AssertEqual(t, "text", strings.ToLower(ct.DatabaseTypeName()))
}

func TestMigrateArrayTypeModel(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	type ArrayTypeModel struct {
		ID              uint
		Number          string     `gorm:"type:varchar(51);NOT NULL"`
		TextArray       []string   `gorm:"type:text[];NOT NULL"`
		NestedTextArray [][]string `gorm:"type:text[][]"`
		NestedIntArray  [][]int64  `gorm:"type:integer[3][3]"`
	}

	var err error
	DB.Migrator().DropTable(&ArrayTypeModel{})

	err = DB.AutoMigrate(&ArrayTypeModel{})
	AssertEqual(t, nil, err)

	ct, err := findColumnType(&ArrayTypeModel{}, "number")
	AssertEqual(t, nil, err)
	AssertEqual(t, "varchar", ct.DatabaseTypeName())

	ct, err = findColumnType(&ArrayTypeModel{}, "text_array")
	AssertEqual(t, nil, err)
	AssertEqual(t, "text[]", ct.DatabaseTypeName())

	ct, err = findColumnType(&ArrayTypeModel{}, "nested_text_array")
	AssertEqual(t, nil, err)
	AssertEqual(t, "text[]", ct.DatabaseTypeName())

	ct, err = findColumnType(&ArrayTypeModel{}, "nested_int_array")
	AssertEqual(t, nil, err)
	AssertEqual(t, "integer[]", ct.DatabaseTypeName())
}

func TestMigrateSameEmbeddedFieldName(t *testing.T) {
	type UserStat struct {
		GroundDestroyCount int
	}

	type GameUser struct {
		gorm.Model
		StatAb UserStat `gorm:"embedded;embeddedPrefix:stat_ab_"`
	}

	type UserStat1 struct {
		GroundDestroyCount string
	}

	type GroundRate struct {
		GroundDestroyCount int
	}

	type GameUser1 struct {
		gorm.Model
		StatAb       UserStat1  `gorm:"embedded;embeddedPrefix:stat_ab_"`
		GroundRateRb GroundRate `gorm:"embedded;embeddedPrefix:rate_ground_rb_"`
	}

	DB.Migrator().DropTable(&GameUser{})
	err := DB.AutoMigrate(&GameUser{})
	AssertEqual(t, nil, err)

	err = DB.Table("game_users").AutoMigrate(&GameUser1{})
	AssertEqual(t, nil, err)

	_, err = findColumnType(&GameUser{}, "stat_ab_ground_destory_count")
	AssertEqual(t, nil, err)

	_, err = findColumnType(&GameUser{}, "rate_ground_rb_ground_destory_count")
	AssertEqual(t, nil, err)
}
