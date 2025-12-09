package lru_test

import (
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm/internal/lru"
)

func TestLRU_KeysValuesKeyValues_OrderAndEviction(t *testing.T) {
	lc := lru.NewLRU[string, string](3, nil, 0)
	lc.Add("k1", "v1")
	lc.Add("k2", "v2")
	lc.Add("k3", "v3")

	// initial order: k1, k2, k3 (oldest -> newest)
	if got := lc.Keys(); !reflect.DeepEqual(got, []string{"k1", "k2", "k3"}) {
		t.Fatalf("Keys() = %v, want [k1 k2 k3]", got)
	}
	if got := lc.Values(); !reflect.DeepEqual(got, []string{"v1", "v2", "v3"}) {
		t.Fatalf("Values() = %v, want [v1 v2 v3]", got)
	}
	kv := lc.KeyValues()
	if len(kv) != 3 || kv["k1"] != "v1" || kv["k3"] != "v3" {
		t.Fatalf("KeyValues() = %v, want map with all three items", kv)
	}

	// Adding a 4th element should evict the oldest (k1)
	lc.Add("k4", "v4")
	if _, ok := lc.Get("k1"); ok {
		t.Fatalf("expected k1 to be evicted after adding k4")
	}
	if got := lc.Keys(); !reflect.DeepEqual(got, []string{"k2", "k3", "k4"}) {
		t.Fatalf("Keys() after eviction = %v, want [k2 k3 k4]", got)
	}
}

func TestLRU_ExpirationFiltering(t *testing.T) {
	// TTL is short so that expiration happens quickly
	lc := lru.NewLRU[string, string](0, nil, 50*time.Millisecond)
	lc.Add("a", "1")
	lc.Add("b", "2")

	// both present initially
	if lc.Len() != 2 {
		t.Fatalf("expected length 2, got %d", lc.Len())
	}

	// wait for entries to expire and for background reaper to run
	time.Sleep(200 * time.Millisecond)

	// expired entries should be filtered out by Keys/Values/KeyValues
	if got := lc.Keys(); len(got) != 0 {
		t.Fatalf("expected no keys after expiration, got %v", got)
	}
	if got := lc.Values(); len(got) != 0 {
		t.Fatalf("expected no values after expiration, got %v", got)
	}
	if got := lc.KeyValues(); len(got) != 0 {
		t.Fatalf("expected no key-values after expiration, got %v", got)
	}
}
