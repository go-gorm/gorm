package gorm

import "time"

// Model base model definition, including `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`, which could be embeded in your models
type Model struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}
