package gorm_test

import (
	"fmt"
	"os"
	"testing"
)

type Blog struct {
	ID      uint   `gorm:"primary_key"`
	Locale  string `gorm:"primary_key"`
	Subject string
	Body    string
	Tags    []Tag `gorm:"many2many:blog_tags;"`
}

type Tag struct {
	ID     uint   `gorm:"primary_key"`
	Locale string `gorm:"primary_key"`
	Value  string
}

func TestManyToManyWithMultiPrimaryKeys(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect != "" && dialect != "sqlite" {
		DB.Exec(fmt.Sprintf("drop table blog_tags;"))
		DB.AutoMigrate(&Blog{}, &Tag{})
		blog := Blog{
			Locale:  "ZH",
			Subject: "subject",
			Body:    "body",
			Tags: []Tag{
				{Locale: "ZH", Value: "tag1"},
				{Locale: "ZH", Value: "tag2"},
			},
		}

		res := DB.Save(&blog)
		if nil != res.Error {
			t.Errorf("Error while saving blog:%v", res.Error)
		}
		res2 := DB.Model(&blog).Association("Tags").Append([]Tag{{Locale: "ZH", Value: "tag3"}})
		if nil != res2.Error {
			t.Errorf("Error while appending tag to blog:%v", res2.Error)
		}
		
		var tags []Tag
		res = DB.Model(&blog).Related(&tags, "Tags")
		if nil != res.Error {
			t.Errorf("Error while reading tags related to blog:%v", res.Error)
		}
		
		tagsCount := len(tags)
		if tagsCount != 3 {
			t.Errorf("should found 3 tags with blog, found:%v", tagsCount)
		}
	}
}
