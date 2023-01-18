package callbacks

import (
	"reflect"

	"gorm.io/gorm"
)

func callMethod(db *gorm.DB, fc func(value interface{}, tx *gorm.DB) bool) {
	tx := db.Session(&gorm.Session{NewDB: true})
	if called := fc(db.Statement.ReflectValue.Interface(), tx); !called {
		switch db.Statement.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			db.Statement.CurDestIndex = 0
			for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
				value := reflect.Indirect(db.Statement.ReflectValue.Index(i))
				if value.CanAddr() {
					fc(value.Addr().Interface(), tx)
				} else {
					fc(value.Interface(), tx)
				}
				db.Statement.CurDestIndex++
			}
		case reflect.Struct:
			if db.Statement.ReflectValue.CanAddr() {
				fc(db.Statement.ReflectValue.Addr().Interface(), tx)
			}
		}
	}
}
