package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
)

func BeforeUpdate(db *gorm.DB) {
	if db.Statement.Schema != nil && (db.Statement.Schema.BeforeSave || db.Statement.Schema.BeforeUpdate) {
		callMethod := func(value interface{}) bool {
			var ok bool
			if db.Statement.Schema.BeforeSave {
				if i, ok := value.(gorm.BeforeSaveInterface); ok {
					ok = true
					i.BeforeSave(db)
				}
			}

			if db.Statement.Schema.BeforeUpdate {
				if i, ok := value.(gorm.BeforeUpdateInterface); ok {
					ok = true
					i.BeforeUpdate(db)
				}
			}
			return ok
		}

		if ok := callMethod(db.Statement.Dest); !ok {
			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i <= db.Statement.ReflectValue.Len(); i++ {
					callMethod(db.Statement.ReflectValue.Index(i).Interface())
				}
			case reflect.Struct:
				callMethod(db.Statement.ReflectValue.Interface())
			}
		}
	}
}

func Update(db *gorm.DB) {
}

func AfterUpdate(db *gorm.DB) {
	if db.Statement.Schema != nil && (db.Statement.Schema.AfterSave || db.Statement.Schema.AfterUpdate) {
		callMethod := func(value interface{}) bool {
			var ok bool
			if db.Statement.Schema.AfterSave {
				if i, ok := value.(gorm.AfterSaveInterface); ok {
					ok = true
					i.AfterSave(db)
				}
			}

			if db.Statement.Schema.AfterUpdate {
				if i, ok := value.(gorm.AfterUpdateInterface); ok {
					ok = true
					i.AfterUpdate(db)
				}
			}
			return ok
		}

		if ok := callMethod(db.Statement.Dest); !ok {
			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i <= db.Statement.ReflectValue.Len(); i++ {
					callMethod(db.Statement.ReflectValue.Index(i).Interface())
				}
			case reflect.Struct:
				callMethod(db.Statement.ReflectValue.Interface())
			}
		}
	}
}
