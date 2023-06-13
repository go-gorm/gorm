package schema_test

import (
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

type UserCheck struct {
	Name  string `gorm:"check:name_checker,name <> 'jinzhu'"`
	Name2 string `gorm:"check:name <> 'jinzhu'"`
	Name3 string `gorm:"check:,name <> 'jinzhu'"`
}

func TestParseCheck(t *testing.T) {
	user, err := schema.Parse(&UserCheck{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user check, got error %v", err)
	}

	results := map[string]schema.CheckConstraint{
		"name_checker": {
			Name:       "name_checker",
			Constraint: "name <> 'jinzhu'",
		},
		"chk_user_checks_name2": {
			Name:       "chk_user_checks_name2",
			Constraint: "name <> 'jinzhu'",
		},
		"chk_user_checks_name3": {
			Name:       "chk_user_checks_name3",
			Constraint: "name <> 'jinzhu'",
		},
	}

	checks := user.ParseCheckConstraints()

	for k, result := range results {
		v, ok := checks[k]
		if !ok {
			t.Errorf("Failed to found check %v from parsed checks %+v", k, checks)
		}

		for _, name := range []string{"Name", "Constraint"} {
			if reflect.ValueOf(result).FieldByName(name).Interface() != reflect.ValueOf(v).FieldByName(name).Interface() {
				t.Errorf(
					"check %v %v should equal, expects %v, got %v",
					k, name, reflect.ValueOf(result).FieldByName(name).Interface(), reflect.ValueOf(v).FieldByName(name).Interface(),
				)
			}
		}
	}
}

func TestParseUniqueConstraints(t *testing.T) {
	type UserUnique struct {
		Name1 string `gorm:"unique"`
		Name2 string `gorm:"uniqueIndex"`
	}

	user, err := schema.Parse(&UserUnique{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user unique, got error %v", err)
	}
	constraints := user.ParseUniqueConstraints()

	results := map[string]schema.UniqueConstraint{
		"uni_user_uniques_name1": {
			Name:  "uni_user_uniques_name1",
			Field: &schema.Field{Name: "Name1", Unique: true},
		},
	}
	for k, result := range results {
		v, ok := constraints[k]
		if !ok {
			t.Errorf("Failed to found unique constraint %v from parsed constraints %+v", k, constraints)
		}
		tests.AssertObjEqual(t, result, v, "Name")
		tests.AssertObjEqual(t, result.Field, v.Field, "Name", "Unique", "UniqueIndex")
	}
}
