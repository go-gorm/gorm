package schema

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func fieldEqual(got, expected *Field) error {
	if expected.DBName != got.DBName {
		return fmt.Errorf("field DBName should be %v, got %v", expected.DBName, got.DBName)
	}

	if expected.Name != got.Name {
		return fmt.Errorf("field Name should be %v, got %v", expected.Name, got.Name)
	}

	if !reflect.DeepEqual(expected.BindNames, got.BindNames) {
		return fmt.Errorf("field BindNames should be %#v, got %#v", expected.BindNames, got.BindNames)
	}

	if (expected.TagSettings == nil && len(got.TagSettings) != 0) && !reflect.DeepEqual(expected.TagSettings, got.TagSettings) {
		return fmt.Errorf("field TagSettings should be %#v, got %#v", expected.TagSettings, got.TagSettings)
	}

	if expected.IsNormal != got.IsNormal {
		return fmt.Errorf("field IsNormal should be %v, got %v", expected.IsNormal, got.IsNormal)
	}

	if expected.IsPrimaryKey != got.IsPrimaryKey {
		return fmt.Errorf("field IsPrimaryKey should be %v, got %v", expected.IsPrimaryKey, got.IsPrimaryKey)
	}

	if expected.IsIgnored != got.IsIgnored {
		return fmt.Errorf("field IsIgnored should be %v, got %v", expected.IsIgnored, got.IsIgnored)
	}

	if expected.IsForeignKey != got.IsForeignKey {
		return fmt.Errorf("field IsForeignKey should be %v, got %v", expected.IsForeignKey, got.IsForeignKey)
	}

	if expected.DefaultValue != got.DefaultValue {
		return fmt.Errorf("field DefaultValue should be %v, got %v", expected.DefaultValue, got.DefaultValue)
	}

	if expected.HasDefaultValue != got.HasDefaultValue {
		return fmt.Errorf("field HasDefaultValue should be %v, got %v", expected.HasDefaultValue, got.HasDefaultValue)
	}
	return nil
}

func compareFields(fields []*Field, expectedFields []*Field, t *testing.T) {
	if len(fields) != len(expectedFields) {
		t.Errorf("expected has %v fields, but got %v", len(expectedFields), len(fields))
	}

	for _, expectedField := range expectedFields {
		field := getSchemaField(expectedField.DBName, fields)
		if field == nil {
			t.Errorf("Field %#v is not found", expectedField.Name)
		} else if err := fieldEqual(field, expectedField); err != nil {
			t.Error(err)
		}
	}
}

func TestParse(t *testing.T) {
	type MyStruct struct {
		ID            int
		Int           uint
		IntPointer    *uint `gorm:"default:10"`
		String        string
		StringPointer *string `gorm:"column:strp"`
		Time          time.Time
		TimePointer   *time.Time
		NullInt64     sql.NullInt64
	}

	schema := Parse(&MyStruct{})

	compareFields(schema.Fields, []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "int", Name: "Int", BindNames: []string{"Int"}, IsNormal: true},
		{DBName: "int_pointer", Name: "IntPointer", BindNames: []string{"IntPointer"}, TagSettings: map[string]string{"DEFAULT": "10"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "10"},
		{DBName: "string", Name: "String", BindNames: []string{"String"}, IsNormal: true},
		{DBName: "strp", Name: "StringPointer", BindNames: []string{"StringPointer"}, TagSettings: map[string]string{"COLUMN": "strp"}, IsNormal: true},
		{DBName: "time", Name: "Time", BindNames: []string{"Time"}, IsNormal: true},
		{DBName: "time_pointer", Name: "TimePointer", BindNames: []string{"TimePointer"}, IsNormal: true},
		{DBName: "null_int64", Name: "NullInt64", BindNames: []string{"NullInt64"}, IsNormal: true},
	}, t)
}

func TestEmbeddedStruct(t *testing.T) {
}

func TestOverwriteEmbeddedStructFields(t *testing.T) {
}

func TestCustomizePrimaryKey(t *testing.T) {
}

func TestCompositePrimaryKeys(t *testing.T) {
}
