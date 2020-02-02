package callbacks

import "github.com/jinzhu/gorm"

func BeforeCreate(db *gorm.DB) {
	// before save
	// before create

	// assign timestamp
}

func SaveBeforeAssociations(db *gorm.DB) {
}

func Create(db *gorm.DB) {
}

func SaveAfterAssociations(db *gorm.DB) {
}

func AfterCreate(db *gorm.DB) {
	// after save
	// after create
}
