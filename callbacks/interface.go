package callbacks

import "gorm.io/gorm"

type beforeSaveInterface interface {
	BeforeSave(*gorm.DB) error
}

type beforeCreateInterface interface {
	BeforeCreate(*gorm.DB) error
}
