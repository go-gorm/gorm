package schema_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
)

type UserWithCallback struct {
}

func (UserWithCallback) BeforeSave(*gorm.DB) {
}

func (UserWithCallback) AfterCreate(*gorm.DB) {
}

func TestCallback(t *testing.T) {
	user, _, err := schema.Parse(&UserWithCallback{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user with callback, got error %v", err)
	}

	for _, str := range []string{"BeforeSave", "AfterCreate"} {
		if !reflect.Indirect(reflect.ValueOf(user)).FieldByName(str).Interface().(bool) {
			t.Errorf("%v should be true", str)
		}
	}

	for _, str := range []string{"BeforeCreate", "BeforeUpdate", "AfterUpdate", "AfterSave", "BeforeDelete", "AfterDelete", "AfterFind"} {
		if reflect.Indirect(reflect.ValueOf(user)).FieldByName(str).Interface().(bool) {
			t.Errorf("%v should be false", str)
		}
	}
}
