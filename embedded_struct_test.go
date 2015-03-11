package gorm_test

import "testing"

type BasePost struct {
	Id    int64
	Title string
	URL   string
}

type HNPost struct {
	BasePost
	Upvotes int32
}

type EngadgetPost struct {
	BasePost BasePost `gorm:"embedded"`
	ImageUrl string
}

func TestSaveAndQueryEmbeddedStruct(t *testing.T) {
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

	if DB.NewScope(&HNPost{}).PrimaryField() == nil {
		t.Errorf("primary key with embedded struct should works")
	}

	for _, field := range DB.NewScope(&HNPost{}).Fields() {
		if field.Name == "BasePost" {
			t.Errorf("scope Fields should not contain embedded struct")
		}
	}
}
