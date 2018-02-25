package schema

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

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
		{DBName: "int_pointer", Name: "IntPointer", BindNames: []string{"IntPointer"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "10", TagSettings: map[string]string{"DEFAULT": "10"}},
		{DBName: "string", Name: "String", BindNames: []string{"String"}, IsNormal: true},
		{DBName: "strp", Name: "StringPointer", BindNames: []string{"StringPointer"}, IsNormal: true, TagSettings: map[string]string{"COLUMN": "strp"}},
		{DBName: "time", Name: "Time", BindNames: []string{"Time"}, IsNormal: true},
		{DBName: "time_pointer", Name: "TimePointer", BindNames: []string{"TimePointer"}, IsNormal: true},
		{DBName: "null_int64", Name: "NullInt64", BindNames: []string{"NullInt64"}, IsNormal: true},
	}, t)
}

func TestCustomizePrimaryKey(t *testing.T) {
	// on_embedded_conflict replace, ignore mode
	type MyStruct struct {
		ID    string
		Name  string
		Email string
	}

	schema := Parse(&MyStruct{})
	expectedFields := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"PRIMARY_KEY": "PRIMARY_KEY"}},
	}
	compareFields(schema.PrimaryFields, expectedFields, t)

	type MyStruct2 struct {
		ID    string
		Name  string
		Email string `gorm:"primary_key;"`
	}
	schema2 := Parse(&MyStruct2{})
	expectedFields2 := []*Field{
		{DBName: "email", Name: "Email", BindNames: []string{"Email"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"PRIMARY_KEY": "PRIMARY_KEY"}},
	}
	compareFields(schema2.PrimaryFields, expectedFields2, t)
}

func TestEmbeddedStruct(t *testing.T) {
	// Anonymous Embedded
	type EmbedStruct struct {
		Name string
		Age  string `gorm:"column:my_age"`
		Role string `gorm:"default:guest"`
	}

	type MyStruct struct {
		ID string
		EmbedStruct
	}

	schema := Parse(&MyStruct{})
	expectedFields := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"EmbedStruct", "Name"}, IsNormal: true},
		{DBName: "my_age", Name: "Age", BindNames: []string{"EmbedStruct", "Age"}, IsNormal: true, TagSettings: map[string]string{"COLUMN": "Age"}},
		{DBName: "role", Name: "Role", BindNames: []string{"EmbedStruct", "Role"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "guest", TagSettings: map[string]string{"COLUMN": "Role"}},
	}
	compareFields(schema.Fields, expectedFields, t)

	// Embedded with Tag
	type MyStruct2 struct {
		ID          string
		EmbedStruct EmbedStruct `gorm:"embedded"`
	}

	schema2 := Parse(&MyStruct2{})
	expectedFields2 := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"EMBEDDED": "EMBEDDED"}},
		{DBName: "name", Name: "Name", BindNames: []string{"EmbedStruct", "Name"}, IsNormal: true, TagSettings: map[string]string{"EMBEDDED": "EMBEDDED"}},
		{DBName: "my_age", Name: "Age", BindNames: []string{"EmbedStruct", "Age"}, IsNormal: true, TagSettings: map[string]string{"EMBEDDED": "EMBEDDED", "COLUMN": "Age"}},
		{DBName: "role", Name: "Role", BindNames: []string{"EmbedStruct", "Role"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "guest", TagSettings: map[string]string{"EMBEDDED": "EMBEDDED", "COLUMN": "Role"}},
	}
	compareFields(schema2.Fields, expectedFields2, t)

	// Embedded with prefix
	type MyStruct3 struct {
		ID          string
		EmbedStruct `gorm:"EMBEDDED_PREFIX:my_"`
	}

	schema3 := Parse(&MyStruct3{})
	expectedFields3 := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"EMBEDDED_PREFIX": "my_"}},
		{DBName: "my_name", Name: "Name", BindNames: []string{"EmbedStruct", "Name"}, IsNormal: true, TagSettings: map[string]string{"EMBEDDED_PREFIX": "my_"}},
		{DBName: "my_my_age", Name: "Age", BindNames: []string{"EmbedStruct", "Age"}, IsNormal: true, TagSettings: map[string]string{"EMBEDDED_PREFIX": "my_", "COLUMN": "Age"}},
		{DBName: "my_role", Name: "Role", BindNames: []string{"EmbedStruct", "Role"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "guest", TagSettings: map[string]string{"EMBEDDED_PREFIX": "my_", "COLUMN": "Role"}},
	}
	compareFields(schema3.Fields, expectedFields3, t)
}

func TestEmbeddedStructWithPrimaryKey(t *testing.T) {
	type EmbedStruct struct {
		ID   string
		Age  string `gorm:"column:my_age"`
		Role string `gorm:"default:guest"`
	}

	type MyStruct struct {
		Name string
		EmbedStruct
	}

	schema := Parse(&MyStruct{})
	expectedFields := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"EmbedStruct", "ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "my_age", Name: "Age", BindNames: []string{"EmbedStruct", "Age"}, IsNormal: true, TagSettings: map[string]string{"COLUMN": "Age"}},
		{DBName: "role", Name: "Role", BindNames: []string{"EmbedStruct", "Role"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "guest", TagSettings: map[string]string{"COLUMN": "Role"}},
	}
	compareFields(schema.Fields, expectedFields, t)
}

func TestOverwriteEmbeddedStructFields(t *testing.T) {
	type EmbedStruct struct {
		Name string
		Age  string `gorm:"column:my_age"`
		Role string `gorm:"default:guest"`
	}

	// on_embedded_conflict replace, ignore mode
	type MyStruct struct {
		ID string
		EmbedStruct
		Age  string `gorm:"on_embedded_conflict:replace;column:my_age2"`
		Name string `gorm:"on_embedded_conflict:ignore;column:my_name"`
	}

	schema := Parse(&MyStruct{})
	expectedFields := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"EmbedStruct", "Name"}, IsNormal: true},
		{DBName: "my_age2", Name: "Age", BindNames: []string{"Age"}, IsNormal: true, TagSettings: map[string]string{"ON_EMBEDDED_CONFLICT": "replace", "COLUMN": "my_age2"}},
		{DBName: "role", Name: "Role", BindNames: []string{"EmbedStruct", "Role"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "guest", TagSettings: map[string]string{"COLUMN": "Role"}},
	}
	compareFields(schema.Fields, expectedFields, t)

	// on_embedded_conflict update mode, ignore mode w/o corresponding field
	type MyStruct2 struct {
		ID string
		EmbedStruct
		Age   string `gorm:"on_embedded_conflict:update;column:my_age2"`
		Name2 string `gorm:"on_embedded_conflict:ignore;column:my_name2"`
	}

	schema2 := Parse(&MyStruct2{})
	expectedFields2 := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"EmbedStruct", "Name"}, IsNormal: true},
		{DBName: "my_name2", Name: "Name2", BindNames: []string{"Name2"}, IsNormal: true, TagSettings: map[string]string{"ON_EMBEDDED_CONFLICT": "ignore", "COLUMN": "my_name2"}},
		{DBName: "my_age2", Name: "Age", BindNames: []string{"EmbedStruct", "Age"}, IsNormal: true, TagSettings: map[string]string{"ON_EMBEDDED_CONFLICT": "update", "COLUMN": "my_age2"}},
		{DBName: "role", Name: "Role", BindNames: []string{"EmbedStruct", "Role"}, IsNormal: true, HasDefaultValue: true, DefaultValue: "guest", TagSettings: map[string]string{"COLUMN": "Role"}},
	}
	compareFields(schema2.Fields, expectedFields2, t)
}

func TestOverwriteEmbeddedStructPrimaryFields(t *testing.T) {
	type EmbedStruct struct {
		Name  string `gorm:"primary_key"`
		Email string
	}

	// on_embedded_conflict replace, ignore mode
	type MyStruct struct {
		ID string
		EmbedStruct
		Name  string `gorm:"on_embedded_conflict:update;column:my_name"`
		Email string `gorm:"primary_key;on_embedded_conflict:ignore;column:my_email"`
	}

	schema := Parse(&MyStruct{})
	expectedFields := []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"PRIMARY_KEY": "PRIMARY_KEY"}},
	}
	compareFields(schema.PrimaryFields, expectedFields, t)

	// on_embedded_conflict update mode, ignore mode w/o corresponding field
	type MyStruct2 struct {
		ID string
		EmbedStruct
		Name  string `gorm:"on_embedded_conflict:ignore;column:my_name2"`
		Email string `gorm:"primary_key;on_embedded_conflict:update"`
	}

	schema2 := Parse(&MyStruct2{})
	expectedFields2 := []*Field{
		{DBName: "name", Name: "Name", BindNames: []string{"EmbedStruct", "Name"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"PRIMARY_KEY": "PRIMARY_KEY"}},
		{DBName: "email", Name: "Email", BindNames: []string{"EmbedStruct", "Email"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"PRIMARY_KEY": "PRIMARY_KEY", "ON_EMBEDDED_CONFLICT": "update"}},
	}
	compareFields(schema2.PrimaryFields, expectedFields2, t)

	// on_embedded_conflict update mode, ignore mode w/o corresponding field
	type MyStruct3 struct {
		ID string
		EmbedStruct
		Name  string `gorm:"on_embedded_conflict:replace;column:my_name2"`
		Email string `gorm:"primary_key;on_embedded_conflict:replace"`
	}

	schema3 := Parse(&MyStruct3{})
	expectedFields3 := []*Field{
		{DBName: "email", Name: "Email", BindNames: []string{"Email"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"PRIMARY_KEY": "PRIMARY_KEY", "ON_EMBEDDED_CONFLICT": "update"}},
	}
	compareFields(schema3.PrimaryFields, expectedFields3, t)
}

////////////////////////////////////////////////////////////////////////////////
// Test Helpers
////////////////////////////////////////////////////////////////////////////////
func compareFields(fields []*Field, expectedFields []*Field, t *testing.T) {
	if len(fields) != len(expectedFields) {
		var exptectedNames, gotNames []string
		for _, field := range fields {
			gotNames = append(gotNames, field.Name)
		}
		for _, field := range expectedFields {
			exptectedNames = append(exptectedNames, field.Name)
		}
		t.Errorf("expected has %v (%#v) fields, but got %v (%#v)", len(expectedFields), exptectedNames, len(fields), gotNames)
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

	if !reflect.DeepEqual(expected.Relationship, got.Relationship) {
		return fmt.Errorf("field Relationship should be %#v, got %#v", expected.Relationship, got.Relationship)
	}
	return nil
}
