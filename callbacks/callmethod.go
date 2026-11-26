package callbacks

import (
	"reflect"

	"gorm.io/gorm"
)

func callMethod(db *gorm.DB, fc func(value interface{}, tx *gorm.DB) bool) {
	tx := db.Session(&gorm.Session{NewDB: true})

	// the destination may be a pointer to an interface, e.g. `var rows interface{} = []*User{}; db.Find(&rows)`,
	// unwrap it locally (db.Statement.ReflectValue must stay untouched, scan needs to Set it)
	reflectValue := db.Statement.ReflectValue
	if reflectValue.Kind() == reflect.Interface && !reflectValue.IsNil() {
		reflectValue = reflectValue.Elem()
	}

	if called := fc(reflectValue.Interface(), tx); !called {
		switch reflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			db.Statement.CurDestIndex = 0
			for i := 0; i < reflectValue.Len(); i++ {
				if value := reflect.Indirect(reflectValue.Index(i)); value.CanAddr() {
					fc(value.Addr().Interface(), tx)
				} else {
					db.AddError(gorm.ErrInvalidValue)
					return
				}
				db.Statement.CurDestIndex++
			}
		case reflect.Struct:
			if reflectValue.CanAddr() {
				fc(reflectValue.Addr().Interface(), tx)
			} else {
				db.AddError(gorm.ErrInvalidValue)
			}
		}
	}
}
