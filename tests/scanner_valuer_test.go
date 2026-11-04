package tests_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		Allergen: NullString{sql.NullString{String: "Allergen", Valid: true}},
		Password: EncryptedData("pass1"),
		Bytes:    []byte("byte"),
		Num:      18,
		Strings:  StringsSlice{"a", "b", "c"},
		Structs: StructsSlice{
			{"name1", "value1"},
			{"name2", "value2"},
		},
		Role:             Role{Name: "admin"},
		ExampleStruct:    ExampleStruct{"name", "value1"},
		ExampleStructPtr: &ExampleStruct{"name", "value2"},
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("No error should happened when create scanner valuer struct, but got %v", err)
	}

	var result ScannerValuerStruct

	if err := DB.Find(&result, "id = ?", data.ID).Error; err != nil {
		t.Fatalf("no error should happen when query scanner, valuer struct, but got %v", err)
	}

	if result.ExampleStructPtr.Val != "value2" {
		t.Errorf(`ExampleStructPtr.Val should equal to "value2", but got %v`, result.ExampleStructPtr.Val)
	}

	if result.ExampleStruct.Val != "value1" {
		t.Errorf(`ExampleStruct.Val should equal to "value1", but got %#v`, result.ExampleStruct)
	}
	AssertObjEqual(t, data, result, "Name", "Gender", "Age", "Male", "Height", "Birthday", "Password", "Bytes", "Num", "Strings", "Structs")
}

func TestScannerValuerArray(t *testing.T) {
	// Use custom dialector to enable array handler
	os.Setenv("GORM_DIALECT", "postgres")
	os.Setenv("GORM_ENABLE_ARRAY_HANDLER", "true")
	var err error
	if DB, err = OpenTestConnection(&gorm.Config{}); err != nil {
		log.Printf("failed to connect database, got error %v", err)
		os.Exit(1)
	}

	DB.Migrator().DropTable(&ScannerValuerStructOfArrays{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStructOfArrays{}); err != nil {
		t.Fatalf("no error should happen when migrate scanner, valuer struct, got error %v", err)
	}

	data := ScannerValuerStructOfArrays{
		StringArray: []string{"a", "b", "c"},
		IntArray:    []int{1, 2, 3},
		Int8Array:   []int8{1, 2, 3},
		Int16Array:  []int16{1, 2, 3},
		Int32Array:  []int32{1, 2, 3},
		Int64Array:  []int64{1, 2, 3},
		UintArray:   []uint{1, 2, 3},
		Uint16Array: []uint16{1, 2, 3},
		Uint32Array: []uint32{1, 2, 3},
		Uint64Array: []uint64{1, 2, 3},
		Float32Array: []float32{
			1.1, 2.2, 3.3,
		},
		Float64Array: []float64{
			1.1, 2.2, 3.3,
		},
		BoolArray: []bool{true, false, true},
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("No error should happened when create scanner valuer struct, but got %v", err)
	}

	var result ScannerValuerStructOfArrays

	if err := DB.Find(&result, "id = ?", data.ID).Error; err != nil {
		t.Fatalf("no error should happen when query scanner, valuer struct, but got %v", err)
	}

	AssertObjEqual(t, data, result, "StringArray", "IntArray", "Int8Array", "Int16Array", "Int32Array", "Int64Array", "UintArray", "Uint16Array", "Uint32Array", "Uint64Array", "Float32Array", "Float64Array", "BoolArray")
}

func TestScannerValuerWithFirstOrCreate(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Errorf("no error should happen when migrate scanner, valuer struct")
	}

	data := ScannerValuerStruct{
		Name:             sql.NullString{String: "name", Valid: true},
		Gender:           &sql.NullString{String: "M", Valid: true},
		Age:              sql.NullInt64{Int64: 18, Valid: true},
		ExampleStruct:    ExampleStruct{"name", "value1"},
		ExampleStructPtr: &ExampleStruct{"name", "value2"},
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
		Password:         EncryptedData("xpass1"),
		ExampleStruct:    ExampleStruct{"name", "value1"},
		ExampleStructPtr: &ExampleStruct{"name", "value2"},
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
	Name             sql.NullString
	Gender           *sql.NullString
	Age              sql.NullInt64
	Male             sql.NullBool
	Height           sql.NullFloat64
	Birthday         sql.NullTime
	Allergen         NullString
	Password         EncryptedData
	Bytes            []byte
	Num              Num
	Strings          StringsSlice
	Structs          StructsSlice
	Role             Role
	UserID           *sql.NullInt64
	User             User
	EmptyTime        EmptyTime
	ExampleStruct    ExampleStruct
	ExampleStructPtr *ExampleStruct
}

type ScannerValuerStructOfArrays struct {
	gorm.Model
	StringArray  []string
	IntArray     []int
	Int8Array    []int8
	Int16Array   []int16
	Int32Array   []int32
	Int64Array   []int64
	UintArray    []uint
	Uint16Array  []uint16
	Uint32Array  []uint32
	Uint64Array  []uint64
	Float32Array []float32
	Float64Array []float64
	BoolArray    []bool
}

type EncryptedData []byte

func (data *EncryptedData) Scan(value interface{}) error {
	if b, ok := value.([]byte); ok {
		if len(b) < 3 || b[0] != '*' || b[1] != '*' || b[2] != '*' {
			return errors.New("Too short")
		}

		*data = append((*data)[0:], b[3:]...)
		return nil
	} else if s, ok := value.(string); ok {
		*data = []byte(s[3:])
		return nil
	}

	return errors.New("Bytes expected")
}

func (data EncryptedData) Value() (driver.Value, error) {
	if len(data) > 0 && data[0] == 'x' {
		// needed to test failures
		return nil, errors.New("Should not start with 'x'")
	}

	// prepend asterisks
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
	Name string
	Val  string
}

func (ExampleStruct) GormDataType() string {
	return "bytes"
}

func (s ExampleStruct) Value() (driver.Value, error) {
	if len(s.Name) == 0 {
		return nil, nil
	}
	// for test, has no practical meaning
	s.Name = ""
	return json.Marshal(s)
}

func (s *ExampleStruct) Scan(src interface{}) error {
	switch value := src.(type) {
	case string:
		return json.Unmarshal([]byte(value), s)
	case []byte:
		return json.Unmarshal(value, s)
	default:
		return errors.New("not supported")
	}
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

type NullString struct {
	sql.NullString
}

type Point struct {
	X, Y int
}

func (point Point) GormDataType() string {
	return "geo"
}

func (point Point) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return clause.Expr{
		SQL:  "ST_PointFromText(?)",
		Vars: []interface{}{fmt.Sprintf("POINT(%d %d)", point.X, point.Y)},
	}
}

func TestGORMValuer(t *testing.T) {
	type UserWithPoint struct {
		Name  string
		Point Point
	}

	dryRunDB := DB.Session(&gorm.Session{DryRun: true})

	stmt := dryRunDB.Create(&UserWithPoint{
		Name:  "jinzhu",
		Point: Point{X: 100, Y: 100},
	}).Statement

	if stmt.SQL.String() == "" || len(stmt.Vars) != 2 {
		t.Errorf("Failed to generate sql, got %v", stmt.SQL.String())
	}

	if !regexp.MustCompile(`INSERT INTO .user_with_points. \(.name.,.point.\) VALUES \(.+,ST_PointFromText\(.+\)\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("insert with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}

	stmt = dryRunDB.Model(UserWithPoint{}).Create(map[string]interface{}{
		"Name":  "jinzhu",
		"Point": clause.Expr{SQL: "ST_PointFromText(?)", Vars: []interface{}{"POINT(100 100)"}},
	}).Statement

	if !regexp.MustCompile(`INSERT INTO .user_with_points. \(.name.,.point.\) VALUES \(.+,ST_PointFromText\(.+\)\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("insert with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}

	stmt = dryRunDB.Table("user_with_points").Create(&map[string]interface{}{
		"Name":  "jinzhu",
		"Point": clause.Expr{SQL: "ST_PointFromText(?)", Vars: []interface{}{"POINT(100 100)"}},
	}).Statement

	if !regexp.MustCompile(`INSERT INTO .user_with_points. \(.Name.,.Point.\) VALUES \(.+,ST_PointFromText\(.+\)\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("insert with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}

	stmt = dryRunDB.Session(&gorm.Session{
		AllowGlobalUpdate: true,
	}).Model(&UserWithPoint{}).Updates(UserWithPoint{
		Name:  "jinzhu",
		Point: Point{X: 100, Y: 100},
	}).Statement

	if !regexp.MustCompile(`UPDATE .user_with_points. SET .name.=.+,.point.=ST_PointFromText\(.+\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("update with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}
}
