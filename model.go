package gorm

import "time"

// Model base model definition, including fields `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`, which could be embedded in your models
//    type User struct {
//      gorm.Model
//    }
type Model struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type ModelInt64 struct {
	ID        uint `gorm:"primary_key"`
	UpdatedAt int64
	CreatedAt int64
	DeletedAt int64 `sql:"index"`
}

type ModelString struct {
	ID        uint `gorm:"primary_key"`
	UpdatedAt string
	CreatedAt string
	DeletedAt string `sql:"index"`
}
