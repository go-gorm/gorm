package callbacks

import "github.com/jinzhu/gorm"

func RegisterDefaultCallbacks(db *gorm.DB) {
	callback := db.Callback()
	callback.Create().Register("gorm:before_create", BeforeCreate)
	callback.Create().Register("gorm:save_before_associations", SaveBeforeAssociations)
	callback.Create().Register("gorm:create", Create)
	callback.Create().Register("gorm:save_after_associations", SaveAfterAssociations)
	callback.Create().Register("gorm:after_create", AfterCreate)
}
