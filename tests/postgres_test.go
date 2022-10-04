package tests_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestPostgresReturningIDWhichHasStringType(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	type Yasuo struct {
		ID        string `gorm:"default:gen_random_uuid()"`
		Name      string
		CreatedAt time.Time `gorm:"type:TIMESTAMP WITHOUT TIME ZONE"`
		UpdatedAt time.Time `gorm:"type:TIMESTAMP WITHOUT TIME ZONE;default:current_timestamp"`
	}

	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;").Error; err != nil {
		t.Errorf("Failed to create extension pgcrypto, got error %v", err)
	}

	DB.Migrator().DropTable(&Yasuo{})

	if err := DB.AutoMigrate(&Yasuo{}); err != nil {
		t.Fatalf("Failed to migrate for uuid default value, got error: %v", err)
	}

	yasuo := Yasuo{Name: "jinzhu"}
	if err := DB.Create(&yasuo).Error; err != nil {
		t.Fatalf("should be able to create data, but got %v", err)
	}

	if yasuo.ID == "" {
		t.Fatal("should be able to has ID, but got zero value")
	}

	var result Yasuo
	if err := DB.First(&result, "id = ?", yasuo.ID).Error; err != nil || yasuo.Name != "jinzhu" {
		t.Errorf("No error should happen, but got %v", err)
	}

	if err := DB.Where("id = $1", yasuo.ID).First(&Yasuo{}).Error; err != nil || yasuo.Name != "jinzhu" {
		t.Errorf("No error should happen, but got %v", err)
	}

	yasuo.Name = "jinzhu1"
	if err := DB.Save(&yasuo).Error; err != nil {
		t.Errorf("Failed to update date, got error %v", err)
	}

	if err := DB.First(&result, "id = ?", yasuo.ID).Error; err != nil || yasuo.Name != "jinzhu1" {
		t.Errorf("No error should happen, but got %v", err)
	}
}

func TestPostgres(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	type Harumph struct {
		gorm.Model
		Name      string         `gorm:"check:name_checker,name <> ''"`
		Test      uuid.UUID      `gorm:"type:uuid;not null;default:gen_random_uuid()"`
		CreatedAt time.Time      `gorm:"type:TIMESTAMP WITHOUT TIME ZONE"`
		UpdatedAt time.Time      `gorm:"type:TIMESTAMP WITHOUT TIME ZONE;default:current_timestamp"`
		Things    pq.StringArray `gorm:"type:text[]"`
	}

	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;").Error; err != nil {
		t.Errorf("Failed to create extension pgcrypto, got error %v", err)
	}

	DB.Migrator().DropTable(&Harumph{})

	if err := DB.AutoMigrate(&Harumph{}); err != nil {
		t.Fatalf("Failed to migrate for uuid default value, got error: %v", err)
	}

	harumph := Harumph{}
	if err := DB.Create(&harumph).Error; err == nil {
		t.Fatalf("should failed to create data, name can't be blank")
	}

	harumph = Harumph{Name: "jinzhu"}
	if err := DB.Create(&harumph).Error; err != nil {
		t.Fatalf("should be able to create data, but got %v", err)
	}

	var result Harumph
	if err := DB.First(&result, "id = ?", harumph.ID).Error; err != nil || harumph.Name != "jinzhu" {
		t.Errorf("No error should happen, but got %v", err)
	}

	if err := DB.Where("id = $1", harumph.ID).First(&Harumph{}).Error; err != nil || harumph.Name != "jinzhu" {
		t.Errorf("No error should happen, but got %v", err)
	}

	harumph.Name = "jinzhu1"
	if err := DB.Save(&harumph).Error; err != nil {
		t.Errorf("Failed to update date, got error %v", err)
	}

	if err := DB.First(&result, "id = ?", harumph.ID).Error; err != nil || harumph.Name != "jinzhu1" {
		t.Errorf("No error should happen, but got %v", err)
	}
}

type Post struct {
	ID         uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4();"`
	Title      string
	Categories []*Category `gorm:"Many2Many:post_categories"`
}

type Category struct {
	ID    uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4();"`
	Title string
	Posts []*Post `gorm:"Many2Many:post_categories"`
}

func TestMany2ManyWithDefaultValueUUID(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	if err := DB.Exec(`create extension if not exists "uuid-ossp"`).Error; err != nil {
		t.Fatalf("Failed to create 'uuid-ossp' extension, but got error %v", err)
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

func TestPostgresOnConstraint(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip()
	}

	type Thing struct {
		gorm.Model
		SomeID  string
		OtherID string
		Data    string
	}

	DB.Migrator().DropTable(&Thing{})
	DB.Migrator().CreateTable(&Thing{})
	if err := DB.Exec("ALTER TABLE things ADD CONSTRAINT some_id_other_id_unique UNIQUE (some_id, other_id)").Error; err != nil {
		t.Error(err)
	}

	thing := Thing{
		SomeID:  "1234",
		OtherID: "1234",
		Data:    "something",
	}

	DB.Create(&thing)

	thing2 := Thing{
		SomeID:  "1234",
		OtherID: "1234",
		Data:    "something else",
	}

	result := DB.Clauses(clause.OnConflict{
		OnConstraint: "some_id_other_id_unique",
		UpdateAll:    true,
	}).Create(&thing2)
	if result.Error != nil {
		t.Errorf("creating second thing: %v", result.Error)
	}

	var things []Thing
	if err := DB.Find(&things).Error; err != nil {
		t.Errorf("Failed, got error: %v", err)
	}

	if len(things) > 1 {
		t.Errorf("expected 1 thing got more")
	}
}
