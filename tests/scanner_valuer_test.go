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

	"github.com/jinzhu/gorm"
	. "github.com/jinzhu/gorm/tests"
)

func TestScannerValuer(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Errorf("no error should happen when migrate scanner, valuer struct")
	}

	data := ScannerValuerStruct{
		Name:     sql.NullString{String: "name", Valid: true},
		Gender:   &sql.NullString{String: "M", Valid: true},
		Age:      sql.NullInt64{Int64: 18, Valid: true},
		Male:     sql.NullBool{Bool: true, Valid: true},
		Height:   sql.NullFloat64{Float64: 1.8888, Valid: true},
		Birthday: sql.NullTime{Time: time.Now(), Valid: true},
		Password: EncryptedData("pass1"),
		Num:      18,
		Strings:  StringsSlice{"a", "b", "c"},
		Structs: StructsSlice{
			{"name1", "value1"},
			{"name2", "value2"},
		},
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Errorf("No error should happend when create scanner valuer struct, but got %v", err)
	}

	var result ScannerValuerStruct

	if err := DB.Find(&result).Error; err != nil {
		t.Errorf("no error should happen when query scanner, valuer struct, but got %v", err)
	}

	AssertObjEqual(t, data, result, "Name", "Gender", "Age", "Male", "Height", "Birthday", "Password", "Num", "Strings", "Structs")
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
	Name     sql.NullString
	Gender   *sql.NullString
	Age      sql.NullInt64
	Male     sql.NullBool
	Height   sql.NullFloat64
	Birthday sql.NullTime
	Password EncryptedData
	Num      Num
	Strings  StringsSlice
	Structs  StructsSlice
}

type EncryptedData []byte

func (data *EncryptedData) Scan(value interface{}) error {
	if b, ok := value.([]byte); ok {
		if len(b) < 3 || b[0] != '*' || b[1] != '*' || b[2] != '*' {
			return errors.New("Too short")
		}

		*data = b[3:]
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
