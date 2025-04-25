package tests_test

import (
	"gorm.io/gorm/internal/lru"
	"testing"
	"time"
)

func TestLRU_Add_ExistingKey_UpdatesValueAndExpiresAt(t *testing.T) {
	lru := lru.NewLRU[string, int](10, nil, time.Hour)
	lru.Add("key1", 1)
	lru.Add("key1", 2)

	if value, ok := lru.Get("key1"); !ok || value != 2 {
		t.Errorf("Expected value to be updated to 2, got %v", value)
	}
}

func TestLRU_Add_NewKey_AddsEntry(t *testing.T) {
	lru := lru.NewLRU[string, int](10, nil, time.Hour)
	lru.Add("key1", 1)

	if value, ok := lru.Get("key1"); !ok || value != 1 {
		t.Errorf("Expected key1 to be added with value 1, got %v", value)
	}
}

func TestLRU_Add_ExceedsSize_RemovesOldest(t *testing.T) {
	lru := lru.NewLRU[string, int](2, nil, time.Hour)
	lru.Add("key1", 1)
	lru.Add("key2", 2)
	lru.Add("key3", 3)

	if _, ok := lru.Get("key1"); ok {
		t.Errorf("Expected key1 to be removed, but it still exists")
	}
}

func TestLRU_Add_UnlimitedSize_NoEviction(t *testing.T) {
	lru := lru.NewLRU[string, int](0, nil, time.Hour)
	lru.Add("key1", 1)
	lru.Add("key2", 2)
	lru.Add("key3", 3)

	if _, ok := lru.Get("key1"); !ok {
		t.Errorf("Expected key1 to exist, but it was evicted")
	}
}

func TestLRU_Add_Eviction(t *testing.T) {
	lru := lru.NewLRU[string, int](0, nil, time.Second*2)
	lru.Add("key1", 1)
	lru.Add("key2", 2)
	lru.Add("key3", 3)
	time.Sleep(time.Second * 3)
	if lru.Cap() != 0 {
		t.Errorf("Expected lru to be empty, but it was not")
	}

}
