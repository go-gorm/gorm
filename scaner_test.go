package gorm_test

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

func TestScannableSlices(t *testing.T) {
	if err := DB.AutoMigrate(&RecordWithSlice{}).Error; err != nil {
		t.Errorf("Should create table with slice values correctly: %s", err)
	}

	r1 := RecordWithSlice{
		Strings: ExampleStringSlice{"a", "b", "c"},
		Structs: ExampleStructSlice{
			{"name1", "value1"},
			{"name2", "value2"},
		},
	}

	if err := DB.Save(&r1).Error; err != nil {
		t.Errorf("Should save record with slice values")
	}

	var r2 RecordWithSlice

	if err := DB.Find(&r2).Error; err != nil {
		t.Errorf("Should fetch record with slice values")
	}

	if len(r2.Strings) != 3 || r2.Strings[0] != "a" || r2.Strings[1] != "b" || r2.Strings[2] != "c" {
		t.Errorf("Should have serialised and deserialised a string array")
	}

	if len(r2.Structs) != 2 || r2.Structs[0].Name != "name1" || r2.Structs[0].Value != "value1" || r2.Structs[1].Name != "name2" || r2.Structs[1].Value != "value2" {
		t.Errorf("Should have serialised and deserialised a struct array")
	}
}

type RecordWithSlice struct {
	ID      uint64
	Strings ExampleStringSlice `sql:"type:text"`
	Structs ExampleStructSlice `sql:"type:text"`
}

type ExampleStringSlice []string

func (l ExampleStringSlice) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func (l *ExampleStringSlice) Scan(input interface{}) error {
	return json.Unmarshal(input.([]byte), l)
}

type ExampleStruct struct {
	Name  string
	Value string
}

type ExampleStructSlice []ExampleStruct

func (l ExampleStructSlice) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func (l *ExampleStructSlice) Scan(input interface{}) error {
	return json.Unmarshal(input.([]byte), l)
}
