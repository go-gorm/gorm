package tests_test

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestEmbeddedStruct(t *testing.T) {
	type ReadOnly struct {
		ReadOnly *bool
	}

	type BasePost struct {
		Id    int64
		Title string
		URL   string
		ReadOnly
	}

	type Author struct {
		ID    string
		Name  string
		Email string
	}

	type HNPost struct {
		BasePost
		Author  `gorm:"EmbeddedPrefix:user_"` // Embedded struct
		Upvotes int32
	}

	type EngadgetPost struct {
		BasePost BasePost `gorm:"Embedded"`
		Author   Author   `gorm:"Embedded;EmbeddedPrefix:author_"` // Embedded struct
		ImageUrl string
	}

	DB.Migrator().DropTable(&HNPost{}, &EngadgetPost{})
	if err := DB.Migrator().AutoMigrate(&HNPost{}, &EngadgetPost{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	for _, name := range []string{"author_id", "author_name", "author_email"} {
		if !DB.Migrator().HasColumn(&EngadgetPost{}, name) {
			t.Errorf("should has prefixed column %v", name)
		}
	}

	stmt := gorm.Statement{DB: DB}
	if err := stmt.Parse(&EngadgetPost{}); err != nil {
		t.Fatalf("failed to parse embedded struct")
	} else if len(stmt.Schema.PrimaryFields) != 1 {
		t.Errorf("should have only one primary field with embedded struct, but got %v", len(stmt.Schema.PrimaryFields))
	}

	for _, name := range []string{"user_id", "user_name", "user_email"} {
		if !DB.Migrator().HasColumn(&HNPost{}, name) {
			t.Errorf("should has prefixed column %v", name)
		}
	}

	// save embedded struct
	DB.Save(&HNPost{BasePost: BasePost{Title: "news"}})
	DB.Save(&HNPost{BasePost: BasePost{Title: "hn_news"}})
	var news HNPost
	if err := DB.First(&news, "title = ?", "hn_news").Error; err != nil {
		t.Errorf("no error should happen when query with embedded struct, but got %v", err)
	} else if news.Title != "hn_news" {
		t.Errorf("embedded struct's value should be scanned correctly")
	}

	DB.Save(&EngadgetPost{BasePost: BasePost{Title: "engadget_news"}})
	var egNews EngadgetPost
	if err := DB.First(&egNews, "title = ?", "engadget_news").Error; err != nil {
		t.Errorf("no error should happen when query with embedded struct, but got %v", err)
	} else if egNews.BasePost.Title != "engadget_news" {
		t.Errorf("embedded struct's value should be scanned correctly")
	}
}

func TestEmbeddedPointerTypeStruct(t *testing.T) {
	type BasePost struct {
		Id    int64
		Title string
		URL   string
	}

	type HNPost struct {
		*BasePost
		Upvotes int32
	}

	DB.Migrator().DropTable(&HNPost{})
	if err := DB.Migrator().AutoMigrate(&HNPost{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	DB.Create(&HNPost{BasePost: &BasePost{Title: "embedded_pointer_type"}})

	var hnPost HNPost
	if err := DB.First(&hnPost, "title = ?", "embedded_pointer_type").Error; err != nil {
		t.Errorf("No error should happen when find embedded pointer type, but got %v", err)
	}

	if hnPost.Title != "embedded_pointer_type" {
		t.Errorf("Should find correct value for embedded pointer type")
	}
}

type Content struct {
	Content interface{} `gorm:"type:String"`
}

func (c Content) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *Content) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return errors.New("Embedded.Scan byte assertion failed")
	}

	var value Content
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}

	*c = value

	return nil
}

func TestEmbeddedScanValuer(t *testing.T) {
	type HNPost struct {
		gorm.Model
		Content
	}

	DB.Migrator().DropTable(&HNPost{})
	if err := DB.Migrator().AutoMigrate(&HNPost{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	hnPost := HNPost{Content: Content{Content: "hello world"}}

	if err := DB.Create(&hnPost).Error; err != nil {
		t.Errorf("Failed to create got error %v", err)
	}
}

func TestEmbeddedRelations(t *testing.T) {
	type AdvancedUser struct {
		User     `gorm:"embedded"`
		Advanced bool
	}

	DB.Migrator().DropTable(&AdvancedUser{})

	if err := DB.AutoMigrate(&AdvancedUser{}); err != nil {
		if DB.Dialector.Name() != "sqlite" {
			t.Errorf("Failed to auto migrate advanced user, got error %v", err)
		}
	}
}
