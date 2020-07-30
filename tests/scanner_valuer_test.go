package tests_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestScannerValuer(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Fatalf("no error should happen when migrate scanner, valuer struct, got error %v", err)
	}

	data := ScannerValuerStruct{
		Name:     sql.NullString{String: "name", Valid: true},
		Gender:   &sql.NullString{String: "M", Valid: true},
		Age:      sql.NullInt64{Int64: 18, Valid: true},
		Male:     sql.NullBool{Bool: true, Valid: true},
		Height:   sql.NullFloat64{Float64: 1.8888, Valid: true},
		Birthday: sql.NullTime{Time: time.Now(), Valid: true},
		Password: EncryptedData("pass1"),
		Bytes:    []byte("byte"),
		Num:      18,
		Strings:  StringsSlice{"a", "b", "c"},
		Structs: StructsSlice{
			{"name1", "value1"},
			{"name2", "value2"},
		},
		Role: Role{Name: "admin"},
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("No error should happened when create scanner valuer struct, but got %v", err)
	}

	var result ScannerValuerStruct

	if err := DB.Find(&result).Error; err != nil {
		t.Fatalf("no error should happen when query scanner, valuer struct, but got %v", err)
	}

	AssertObjEqual(t, data, result, "Name", "Gender", "Age", "Male", "Height", "Birthday", "Password", "Bytes", "Num", "Strings", "Structs")
}

func TestScannerValuerWithFirstOrCreate(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Errorf("no error should happen when migrate scanner, valuer struct")
	}

	data := ScannerValuerStruct{
		Name:   sql.NullString{String: "name", Valid: true},
		Gender: &sql.NullString{String: "M", Valid: true},
		Age:    sql.NullInt64{Int64: 18, Valid: true},
	}

	var result ScannerValuerStruct
	tx := DB.Where(data).FirstOrCreate(&result)

	if tx.RowsAffected != 1 {
		t.Errorf("RowsAffected should be 1 after create some record")
	}

	if tx.Error != nil {
		t.Errorf("Should not raise any error, but got %v", tx.Error)
	}

	AssertObjEqual(t, result, data, "Name", "Gender", "Age")

	if err := DB.Where(data).Assign(ScannerValuerStruct{Age: sql.NullInt64{Int64: 18, Valid: true}}).FirstOrCreate(&result).Error; err != nil {
		t.Errorf("Should not raise any error, but got %v", err)
	}

	if result.Age.Int64 != 18 {
		t.Errorf("should update age to 18")
	}

	var result2 ScannerValuerStruct
	if err := DB.First(&result2, result.ID).Error; err != nil {
		t.Errorf("got error %v when query with %v", err, result.ID)
	}

	AssertObjEqual(t, result2, result, "ID", "CreatedAt", "UpdatedAt", "Name", "Gender", "Age")
}

func TestInvalidValuer(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Errorf("no error should happen when migrate scanner, valuer struct")
	}

	data := ScannerValuerStruct{
		Password: EncryptedData("xpass1"),
	}

	if err := DB.Create(&data).Error; err == nil {
		t.Errorf("Should failed to create data with invalid data")
	}

	data.Password = EncryptedData("pass1")
	if err := DB.Create(&data).Error; err != nil {
		t.Errorf("Should got no error when creating data, but got %v", err)
	}

	if err := DB.Model(&data).Update("password", EncryptedData("xnewpass")).Error; err == nil {
		t.Errorf("Should failed to update data with invalid data")
	}

	if err := DB.Model(&data).Update("password", EncryptedData("newpass")).Error; err != nil {
		t.Errorf("Should got no error update data with valid data, but got %v", err)
	}

	AssertEqual(t, data.Password, EncryptedData("newpass"))
}

type ScannerValuerStruct struct {
	gorm.Model
	Name      sql.NullString
	Gender    *sql.NullString
	Age       sql.NullInt64
	Male      sql.NullBool
	Height    sql.NullFloat64
	Birthday  sql.NullTime
	Password  EncryptedData
	Bytes     []byte
	Num       Num
	Strings   StringsSlice
	Structs   StructsSlice
	Role      Role
	UserID    *sql.NullInt64
	User      User
	EmptyTime EmptyTime
}

type EncryptedData []byte

func (data *EncryptedData) Scan(value interface{}) error {
	if b, ok := value.([]byte); ok {
		if len(b) < 3 || b[0] != '*' || b[1] != '*' || b[2] != '*' {
			return errors.New("Too short")
		}

		*data = b[3:]
		return nil
	} else if s, ok := value.(string); ok {
		*data = []byte(s)[3:]
		return nil
	}

	return errors.New("Bytes expected")
}

func (data EncryptedData) Value() (driver.Value, error) {
	if len(data) > 0 && data[0] == 'x' {
		//needed to test failures
		return nil, errors.New("Should not start with 'x'")
	}

	//prepend asterisks
	return append([]byte("***"), data...), nil
}

type Num int64

func (i *Num) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		n, _ := strconv.Atoi(string(s))
		*i = Num(n)
	case int64:
		*i = Num(s)
	default:
		return errors.New("Cannot scan NamedInt from " + reflect.ValueOf(src).String())
	}
	return nil
}

type StringsSlice []string

func (l StringsSlice) Value() (driver.Value, error) {
	bytes, err := json.Marshal(l)
	return string(bytes), err
}

func (l *StringsSlice) Scan(input interface{}) error {
	switch value := input.(type) {
	case string:
		return json.Unmarshal([]byte(value), l)
	case []byte:
		return json.Unmarshal(value, l)
	default:
		return errors.New("not supported")
	}
}

type ExampleStruct struct {
	Name  string
	Value string
}

type StructsSlice []ExampleStruct

func (l StructsSlice) Value() (driver.Value, error) {
	bytes, err := json.Marshal(l)
	return string(bytes), err
}

func (l *StructsSlice) Scan(input interface{}) error {
	switch value := input.(type) {
	case string:
		return json.Unmarshal([]byte(value), l)
	case []byte:
		return json.Unmarshal(value, l)
	default:
		return errors.New("not supported")
	}
}

type Role struct {
	Name string `gorm:"size:256"`
}

func (role *Role) Scan(value interface{}) error {
	if b, ok := value.([]uint8); ok {
		role.Name = string(b)
	} else {
		role.Name = value.(string)
	}
	return nil
}

func (role Role) Value() (driver.Value, error) {
	return role.Name, nil
}

func (role Role) IsAdmin() bool {
	return role.Name == "admin"
}

type EmptyTime struct {
	time.Time
}

func (t *EmptyTime) Scan(v interface{}) error {
	nullTime := sql.NullTime{}
	err := nullTime.Scan(v)
	t.Time = nullTime.Time
	return err
}

func (t EmptyTime) Value() (driver.Value, error) {
	return time.Now() /* pass tests, mysql 8 doesn't support 0000-00-00 by default */, nil
}
