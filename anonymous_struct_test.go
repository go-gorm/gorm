package gorm_test

import "testing"

type BasePost struct {
	Id    int64
	Title string
	Url   string
}

type HNPost struct {
	BasePost
	Upvotes int32
}

type EngadgetPost struct {
	BasePost
	ImageUrl string
}

func TestAnonymousStruct(t *testing.T) {
	hn := HNPost{}
	hn.Title = "hn_news"
	DB.Debug().Save(hn)

	var news HNPost
	DB.Debug().First(&news)
}
