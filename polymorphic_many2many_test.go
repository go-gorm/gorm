package gorm_test

import (
	"testing"
)

// DB Tables structure:

// simple_posts
//     id - integer
//     name - string

// simple_videos
//     id - integer
//     name - string

// simple_tags
//     id - integer
//     name - string

// taggables
//     tag_id - integer
//     taggable_id - integer
//     taggable_type - string

type SimplePost struct {
	ID   int
	Name string
	Tags []*SimpleTag `gorm:"many2many:taggables;polymorphic:taggable;"`
}

type SimpleVideo struct {
	ID   int
	Name string
	Tags []*SimpleTag `gorm:"many2many:taggables;polymorphic:taggable;polymorphic_value:video"`
}

type SimpleTag struct {
	ID   int
	Name string
}

func TestPolymorphicMany2many(t *testing.T) {
	DB.DropTableIfExists(&SimpleTag{}, &SimplePost{}, &SimpleVideo{}, "taggables")
	DB.AutoMigrate(&SimpleTag{}, &SimplePost{}, &SimpleVideo{})

	DB.LogMode(true)

	tag1 := SimpleTag{Name: "hero"}
	tag2 := SimpleTag{Name: "bloods"}
	tag3 := SimpleTag{Name: "frendship"}
	tag4 := SimpleTag{Name: "romantic"}
	tag5 := SimpleTag{Name: "gruesome"}

	// Test Save associations together
	post1 := SimplePost{Name: "First Post", Tags: []*SimpleTag{&tag1, &tag2, &tag3}}
	err := DB.Save(&post1).Error
	if err != nil {
		t.Errorf("Data init fail : %v \n", err)
	}
	count := DB.Model(&post1).Association("Tags").Count()
	if count != 3 {
		t.Errorf("Post1 should have 3 associations to tags, but got %d", count)
	}

	post2 := SimplePost{Name: "Second Post"}
	video1 := SimpleVideo{Name: "First Video"}
	video2 := SimpleVideo{Name: "Second Video"}
	DB.Save(&post2).Save(&video1).Save(&video2)

	// Test Append
	DB.Model(&post2).Association("Tags").Append(&tag2, &tag4)
	DB.Model(&video1).Association("Tags").Append(&tag1, &tag2, &tag5)
	DB.Model(&video2).Association("Tags").Append(&tag2, &tag3, &tag4)

	count = DB.Model(&post2).Association("Tags").Count()
	if count != 2 {
		t.Errorf("Post2 should have 2 associations to tags, but got %d", count)
	}

	exists := false
	for _, t := range post2.Tags {
		if exists = t.Name == "bloods"; exists {
			break
		}
	}

	if !exists {
		t.Errorf("Post2 should have a tag named 'bloods'")
	}

	count = DB.Model(&video1).Association("Tags").Count()
	if count != 3 {
		t.Errorf("Video1 should have 3 associations to tags, but got %d", count)
	}

	// Test Replace
	tag6 := SimpleTag{Name: "tag6"}
	DB.Model(&post2).Association("Tags").Replace(&tag5, &tag4, &tag6)
	tag2Exists := false
	tag4Exists := false
	tag5Exists := false
	tag6Exists := false
	for _, t := range post2.Tags {
		if !tag2Exists {
			tag2Exists = t.Name == "bloods"
		}
		if !tag4Exists {
			tag4Exists = t.Name == "romantic"
		}
		if !tag5Exists {
			tag5Exists = t.Name == "gruesome"
		}
		if !tag6Exists {
			tag6Exists = t.Name == "tag6"
		}
	}
	if tag2Exists {
		t.Errorf("Post2 should NOT HAVE a tag named 'bloods'")
	}
	if !tag4Exists {
		t.Errorf("Post2 should HAVE a tag named 'romantic'")
	}
	if !tag5Exists {
		t.Errorf("Post2 should HAVE a tag named 'gruesome'")
	}
	if !tag6Exists {
		t.Errorf("Post2 should HAVE a tag named 'tag6'")
	}

	// Test Delete
	DB.Model(&post1).Association("Tags").Delete(&tag1)
	count = DB.Model(&post2).Association("Tags").Count()
	if count != 3 {
		t.Errorf("Post1 should be removed 1 association, should remain 3, but %d", count)
	}

	// Test Clear
	count = DB.Model(&video2).Association("Tags").Count()
	if count != 3 {
		t.Errorf("Video2 should have 3 association, but got %d", count)
	}
	DB.Model(&video2).Association("Tags").Clear()
	count = DB.Model(&video2).Association("Tags").Count()
	if count != 0 {
		t.Errorf("Video2 should be removed all association, but got %d", count)
	}

	DB.LogMode(false)
}

func TestNamedPolymorphicMany2many(t *testing.T) {
	DB.DropTableIfExists(&SimpleTag{}, &SimplePost{}, &SimpleVideo{}, "taggables")
	DB.AutoMigrate(&SimpleTag{}, &SimplePost{}, &SimpleVideo{})

	DB.LogMode(true)

	tag1 := SimpleTag{Name: "hero"}
	tag2 := SimpleTag{Name: "bloods"}
	tag3 := SimpleTag{Name: "frendship"}
	tag4 := SimpleTag{Name: "romantic"}
	tag5 := SimpleTag{Name: "gruesome"}

	// Test Save associations together
	post1 := SimplePost{Name: "First Post", Tags: []*SimpleTag{&tag1, &tag2, &tag3}}
	err := DB.Save(&post1).Error
	if err != nil {
		t.Errorf("Data init fail : %v \n", err)
	}
	count := DB.Model(&post1).Association("Tags").Count()
	if count != 3 {
		t.Errorf("Post1 should have 3 associations to tags, but got %d", count)
	}

	post2 := SimplePost{Name: "Second Post"}
	video1 := SimpleVideo{Name: "First Video"}
	video2 := SimpleVideo{Name: "Second Video"}
	DB.Save(&post2).Save(&video1).Save(&video2)

	// Test Append
	DB.Model(&post2).Association("Tags").Append(&tag2, &tag4)
	DB.Model(&video1).Association("Tags").Append(&tag1, &tag2, &tag5)
	DB.Model(&video2).Association("Tags").Append(&tag2, &tag3, &tag4)

	count = DB.Model(&video1).Association("Tags").Count()
	if count != 3 {
		t.Errorf("Video1 should have 3 associations to tags, but got %d", count)
	}

	exists := false
	for _, t := range video1.Tags {
		if exists = t.Name == "bloods"; exists {
			break
		}
	}

	if !exists {
		t.Errorf("Video1 should have a tag named 'bloods'")
	}

	// Test Replace
	tag6 := SimpleTag{Name: "tag6"}
	DB.Model(&video1).Association("Tags").Replace(&tag2, &tag4, &tag6)
	tag2Exists := false
	tag4Exists := false
	tag5Exists := false
	tag6Exists := false
	for _, t := range video1.Tags {
		if !tag2Exists {
			tag2Exists = t.Name == "bloods"
		}
		if !tag4Exists {
			tag4Exists = t.Name == "romantic"
		}
		if !tag5Exists {
			tag5Exists = t.Name == "gruesome"
		}
		if !tag6Exists {
			tag6Exists = t.Name == "tag6"
		}
	}
	if !tag2Exists {
		t.Errorf("Video1 should HAVE a tag named 'bloods'")
	}
	if !tag4Exists {
		t.Errorf("Video1 should HAVE a tag named 'romantic'")
	}
	if tag5Exists {
		t.Errorf("Video1 should NOT HAVE a tag named 'gruesome'")
	}
	if !tag6Exists {
		t.Errorf("Video1 should HAVE a tag named 'tag6'")
	}

	// Test Delete
	DB.Model(&video1).Association("Tags").Delete(&tag2)
	count = DB.Model(&video1).Association("Tags").Count()
	if count != 2 {
		t.Errorf("video1 should be removed 1 association, should remain 2, but %d", count)
	}

	DB.LogMode(false)
}
