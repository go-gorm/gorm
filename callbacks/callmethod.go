package callbacks

import (
	"reflect"

	"gorm.io/gorm"
)

// callMethod dispatches fc to the destination value first, then, if fc reports no
// matching hook, to addressable model pointers in a struct or slice/array. It maintains
// CurDestIndex for slice hooks and records ErrInvalidValue for an unaddressable model.
// The hook session uses NewDB to omit the caller's query conditions while retaining its
// connection or transaction. fc is responsible for recording hook errors on the
// original DB.
func callMethod(db *gorm.DB, fc func(value interface{}, tx *gorm.DB) bool) {
	tx := db.Session(&gorm.Session{NewDB: true})
	if called := fc(db.Statement.ReflectValue.Interface(), tx); !called {
		switch db.Statement.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			db.Statement.CurDestIndex = 0
			for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
				if value := reflect.Indirect(db.Statement.ReflectValue.Index(i)); value.CanAddr() {
					fc(value.Addr().Interface(), tx)
				} else {
					db.AddError(gorm.ErrInvalidValue)
					return
				}
				db.Statement.CurDestIndex++
			}
		case reflect.Struct:
			if db.Statement.ReflectValue.CanAddr() {
				fc(db.Statement.ReflectValue.Addr().Interface(), tx)
			} else {
				db.AddError(gorm.ErrInvalidValue)
			}
		}
	}
}
