package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
)

func BeforeDelete(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.BeforeDelete {
		callMethod := func(value interface{}) bool {
			if db.Statement.Schema.BeforeDelete {
				if i, ok := value.(gorm.BeforeDeleteInterface); ok {
					i.BeforeDelete(db)
					return true
				}
			}
			return false
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

func Delete(db *gorm.DB) {
}

func AfterDelete(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.AfterDelete {
		callMethod := func(value interface{}) bool {
			if db.Statement.Schema.AfterDelete {
				if i, ok := value.(gorm.AfterDeleteInterface); ok {
					i.AfterDelete(db)
					return true
				}
			}
			return false
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
