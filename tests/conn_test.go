package tests_test

import (
	"context"
	"database/sql"
	. "gorm.io/gorm/utils/tests"
	"reflect"
	"testing"
)

func TestConnLeak(t *testing.T) {

	DB.Table("non_existent").WithContext(context.Background()).FirstOrCreate(&User{Name: "foo"})
	DB.Table("non_existent").WithContext(context.Background()).FirstOrCreate(&User{Name: "foo"})
	DB.Table("non_existent").WithContext(context.Background()).FirstOrCreate(&User{Name: "foo"})

	connPool := DB.ConnPool.(*sql.DB)
	v := reflect.ValueOf(connPool).Elem()
	f := v.FieldByName("numOpen")

	if f.Int() > 1 {
		t.Errorf("Expected open one connections but found %d", f.Int())
	}
}
