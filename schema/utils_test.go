package schema

import (
	"reflect"
	"testing"
)

func TestRemoveSettingFromTag(t *testing.T) {
	tags := map[string]string{
		`gorm:"before:value;column:db;after:value" other:"before:value;column:db;after:value"`:  `gorm:"before:value;after:value" other:"before:value;column:db;after:value"`,
		`gorm:"before:value;column:db;" other:"before:value;column:db;after:value"`:             `gorm:"before:value;" other:"before:value;column:db;after:value"`,
		`gorm:"before:value;column:db" other:"before:value;column:db;after:value"`:              `gorm:"before:value;" other:"before:value;column:db;after:value"`,
		`gorm:"column:db" other:"before:value;column:db;after:value"`:                           `gorm:"" other:"before:value;column:db;after:value"`,
		`gorm:"before:value;column:db ;after:value" other:"before:value;column:db;after:value"`: `gorm:"before:value;after:value" other:"before:value;column:db;after:value"`,
		`gorm:"before:value;column:db; after:value" other:"before:value;column:db;after:value"`: `gorm:"before:value; after:value" other:"before:value;column:db;after:value"`,
		`gorm:"before:value;column; after:value" other:"before:value;column:db;after:value"`:    `gorm:"before:value; after:value" other:"before:value;column:db;after:value"`,
	}

	for k, v := range tags {
		if string(removeSettingFromTag(reflect.StructTag(k), "column")) != v {
			t.Errorf("%v after removeSettingFromTag should equal %v, but got %v", k, v, removeSettingFromTag(reflect.StructTag(k), "column"))
		}
	}
}

func TestParseTagSettingWithDoubleQuoteEscape(t *testing.T) {
	tag := `gorm:"expression:to_tsvector('english', \"Name\")"`
	settings := ParseTagSetting(reflect.StructTag(tag).Get("gorm"), ";")
	if v, ok := settings["EXPRESSION"]; !ok || v != `to_tsvector('english', "Name")` {
		t.Errorf("ParseTagSetting did not handle escaped double quotes correctly: got %#v", v)
	}
}
