package callbacks

import (
	"gorm.io/gorm"
	"reflect"
)

// AfterError after error callback executes if any error happens during main callbacks
func AfterError(db *gorm.DB) {
	if db.Statement.ReflectValue.Kind() == reflect.Ptr && db.Statement.ReflectValue.IsNil() {
		return
	}
	if db.Error != nil && db.Statement.Schema != nil && !db.Statement.SkipHooks {
		callMethod(db, func(value interface{}, tx *gorm.DB) bool {
			if db.Statement.Schema.AfterError {
				if i, ok := value.(AfterErrorInterface); ok {
					db.AddError(i.AfterError(tx))
					return true
				}
			}
			return false
		})
	}
	return
}
