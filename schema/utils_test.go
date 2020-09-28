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

func TestGenPatternName(t *testing.T) {
	gives := []struct {
		Prefix         string
		IntervalBefore string
		IntervalAfter  string
		Vars           []string
		Want           string
	}{
		{
			Prefix:         "fk",
			IntervalBefore: charUnderscore,
			IntervalAfter:  charPoint,
			Vars:           []string{"a", "b"},
			Want:           "fk_a_b",
		},
		{
			Prefix:         "idx",
			IntervalBefore: charUnderscore,
			IntervalAfter:  charPoint,
			Vars:           []string{"a", "b"},
			Want:           "idx_a_b",
		},
		{
			Prefix:         "chk",
			IntervalBefore: charUnderscore,
			IntervalAfter:  charPoint,
			Vars:           []string{"a", "b"},
			Want:           "chk_a_b",
		},
		{
			Prefix:         "idx",
			IntervalBefore: charBlank,
			IntervalAfter:  charBlank,
			Vars:           []string{"a", "b"},
			Want:           "idxab",
		},
	}

	for i := range gives {
		if genPatternName(gives[i].Prefix, gives[i].IntervalBefore, gives[i].IntervalAfter, gives[i].Vars...) != gives[i].Want {
			t.Errorf("want %s, but got %s", gives[i].Want, genPatternName(gives[i].Prefix, gives[i].IntervalBefore, gives[i].IntervalAfter, gives[i].Vars...))
		}
	}
}
