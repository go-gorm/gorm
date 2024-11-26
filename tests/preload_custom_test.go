package tests_test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Structs for preload tests
type PreloadItem struct {
	ID        uint
	Name      string
	Tags      []PreloadTag `gorm:"many2many:preload_items_preload_tags"`
	CreatedAt time.Time
}

type PreloadTag struct {
	ID        uint
	Name      string
	Status    string
	SubTags   []PreloadSubTag `gorm:"many2many:tag_sub_tags"`
}

type PreloadSubTag struct {
	ID     uint
	Name   string
	Status string
}

// Setup database for preload tests
func setupPreloadTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&PreloadItem{}, &PreloadTag{}, &PreloadSubTag{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

// Test default preload functionality
func TestDefaultPreload(t *testing.T) {
	db := setupPreloadTestDB(t)

	tag1 := PreloadTag{Name: "Tag1", Status: "active"}
	item := PreloadItem{Name: "Item1", Tags: []PreloadTag{tag1}}
	db.Create(&item)

	var items []PreloadItem
	err := db.Preload("Tags").Find(&items).Error
	if err != nil {
		t.Fatalf("default preload failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 1 || items[0].Tags[0].Name != "Tag1" {
		t.Errorf("unexpected default preload results: %v", items)
	}
}

// Test preloading with custom joins and conditions
func TestCustomJoinsWithConditions(t *testing.T) {
	db := setupPreloadTestDB(t)

	tag1 := PreloadTag{Name: "Tag1", Status: "active"}
	tag2 := PreloadTag{Name: "Tag2", Status: "inactive"}
	item := PreloadItem{Name: "Item1", Tags: []PreloadTag{tag1, tag2}}
	db.Create(&item)

	var items []PreloadItem
	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN preload_items_preload_tags ON preload_items_preload_tags.preload_tag_id = preload_tags.id").
			Where("preload_tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("custom join with conditions failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 1 || items[0].Tags[0].Status != "active" {
		t.Errorf("unexpected results in TestCustomJoinsWithConditions: %v", items)
	}
}

// Test nested preload functionality with custom joins
func TestNestedPreloadWithCustomJoins(t *testing.T) {
	db := setupPreloadTestDB(t)

	subTag := PreloadSubTag{Name: "SubTag1", Status: "active"}
	tag := PreloadTag{Name: "Tag1", Status: "active", SubTags: []PreloadSubTag{subTag}}
	item := PreloadItem{Name: "Item1", Tags: []PreloadTag{tag}}
	db.Create(&item)

	var items []PreloadItem
	err := db.Preload("Tags.SubTags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN tag_sub_tags ON tag_sub_tags.preload_sub_tag_id = preload_sub_tags.id").
			Where("preload_sub_tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("nested preload with custom joins failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 1 || len(items[0].Tags[0].SubTags) != 1 || items[0].Tags[0].SubTags[0].Name != "SubTag1" {
		t.Errorf("unexpected nested preload results: %v", items)
	}
}

// Test behavior when no matching records exist
func TestNoMatchingRecords(t *testing.T) {
	db := setupPreloadTestDB(t)

	tag := PreloadTag{Name: "Tag1", Status: "inactive"}
	item := PreloadItem{Name: "Item1", Tags: []PreloadTag{tag}}
	db.Create(&item)

	var items []PreloadItem
	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN preload_items_preload_tags ON preload_items_preload_tags.preload_tag_id = preload_tags.id").
			Where("preload_tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("preload with no matching records failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 0 {
		t.Errorf("unexpected results in TestNoMatchingRecords: %v", items)
	}
}

// Test behavior with an empty database
func TestEmptyDatabase(t *testing.T) {
	db := setupPreloadTestDB(t)

	var items []PreloadItem
	err := db.Preload("Tags").Find(&items).Error
	if err != nil {
		t.Fatalf("preload with empty database failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("unexpected results in TestEmptyDatabase: %v", items)
	}
}

// Test multiple items with different tag statuses
func TestMultipleItemsWithDifferentTagStatuses(t *testing.T) {
	db := setupPreloadTestDB(t)

	tag1 := PreloadTag{Name: "Tag1", Status: "active"}
	tag2 := PreloadTag{Name: "Tag2", Status: "inactive"}
	item1 := PreloadItem{Name: "Item1", Tags: []PreloadTag{tag1}}
	item2 := PreloadItem{Name: "Item2", Tags: []PreloadTag{tag2}}
	db.Create(&item1)
	db.Create(&item2)

	var items []PreloadItem
	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN preload_items_preload_tags ON preload_items_preload_tags.preload_tag_id = preload_tags.id").
			Where("preload_tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("preload with multiple items failed: %v", err)
	}

	if len(items) != 2 || len(items[0].Tags) != 1 || len(items[1].Tags) != 0 {
		t.Errorf("unexpected results in TestMultipleItemsWithDifferentTagStatuses: %v", items)
	}
}

// Test duplicate preload conditions
func TestDuplicatePreloadConditions(t *testing.T) {
	db := setupPreloadTestDB(t)

	tag1 := PreloadTag{Name: "Tag1", Status: "active"}
	tag2 := PreloadTag{Name: "Tag2", Status: "inactive"}
	item := PreloadItem{Name: "Item1", Tags: []PreloadTag{tag1, tag2}}
	db.Create(&item)

	var activeTagsItems []PreloadItem
	var inactiveTagsItems []PreloadItem

	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", "active")
	}).Find(&activeTagsItems).Error
	if err != nil {
		t.Fatalf("preload for active tags failed: %v", err)
	}

	err = db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", "inactive")
	}).Find(&inactiveTagsItems).Error
	if err != nil {
		t.Fatalf("preload for inactive tags failed: %v", err)
	}

	if len(activeTagsItems) != 1 || len(activeTagsItems[0].Tags) != 1 || activeTagsItems[0].Tags[0].Status != "active" {
		t.Errorf("unexpected active tag results in TestDuplicatePreloadConditions: %v", activeTagsItems)
	}

	if len(inactiveTagsItems) != 1 || len(inactiveTagsItems[0].Tags) != 1 || inactiveTagsItems[0].Tags[0].Status != "inactive" {
		t.Errorf("unexpected inactive tag results in TestDuplicatePreloadConditions: %v", inactiveTagsItems)
	}
}
