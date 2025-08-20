package migrations

import (
	"gorm.io/gorm/configs"
	"gorm.io/gorm/internal/models"
)

// Up migrates table User
func UpUser() {
	configs.DB.AutoMigrate(&models.User{})
}

// Down rolls back table User
func DownUser() {
	configs.DB.Migrator().DropTable(&models.User{})
}
