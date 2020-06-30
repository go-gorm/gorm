package callbacks

import (
	"reflect"

	"gorm.io/gorm"
)

func callMethod(db *gorm.DB, fc func(value interface{}, tx *gorm.DB) bool) {
	tx := db.Session(&gorm.Session{})
	if called := fc(db.Statement.Dest, tx); !called {
		switch db.Statement.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			db.Statement.CurDestIndex = 0
			for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
				fc(reflect.Indirect(db.Statement.ReflectValue.Index(i)).Addr().Interface(), tx)
				db.Statement.CurDestIndex++
			}
		case reflect.Struct:
			fc(db.Statement.ReflectValue.Addr().Interface(), tx)
		}
	}
}
