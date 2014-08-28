package gorm_test

import "testing"

type BasePost struct {
	Id    int64
	Title string
	Url   string
}

type HNPost struct {
	BasePost `gorm:"embedded"`
	Upvotes  int32
}

type EngadgetPost struct {
	BasePost
	ImageUrl string
}

func TestSaveAndQueryEmbeddedStruct(t *testing.T) {
	DB.Save(HNPost{BasePost: BasePost{Title: "hn_news"}})

	var news HNPost
	if err := DB.First(&news, "title = ?", "hn_news").Error; err != nil {
		t.Errorf("no error should happen when query with embedded struct, but got %v", err)
	} else {
		if news.BasePost.Title == "hn_news" {
			t.Errorf("embedded struct's value should be scanned correctly")
		}
	}
}
