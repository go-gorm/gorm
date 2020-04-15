package tests

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestJoins(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	t.Run("Joins", func(t *testing.T) {
	})
}
