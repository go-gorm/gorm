package tests_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func TestPostgres(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	type Harumph struct {
		gorm.Model
		Test   uuid.UUID      `gorm:"type:uuid;not null;default:gen_random_uuid()"`
		Things pq.StringArray `gorm:"type:text[]"`
	}

	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;").Error; err != nil {
		t.Errorf("Failed to create extension pgcrypto, got error %v", err)
	}

	DB.Migrator().DropTable(&Harumph{})

	if err := DB.AutoMigrate(&Harumph{}); err != nil {
		t.Fatalf("Failed to migrate for uuid default value, got error: %v", err)
	}

	harumph := Harumph{}
	DB.Create(&harumph)

	var result Harumph
	if err := DB.First(&result, "id = ?", harumph.ID).Error; err != nil {
		t.Errorf("No error should happen, but got %v", err)
	}
}

type Post struct {
	ID         uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Title      string
	Categories []*Category `gorm:"Many2Many:post_categories"`
}

type Category struct {
	ID    uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Title string
	Posts []*Post `gorm:"Many2Many:post_categories"`
}

func TestMany2ManyWithDefaultValueUUID(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	DB.Migrator().DropTable(&Post{}, &Category{}, "post_categories")
	DB.AutoMigrate(&Post{}, &Category{})

	post := Post{
		Title: "Hello World",
		Categories: []*Category{
			{Title: "Coding"},
			{Title: "Golang"},
		},
	}

	if err := DB.Create(&post).Error; err != nil {
		t.Errorf("Failed, got error: %v", err)
	}
}
