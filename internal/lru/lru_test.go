package lru

import (
	"sync"
	"testing"
	"time"
)

// evictRecorder is mutex-guarded because the cleanup goroutine may call onEvict.
type evictRecorder struct {
	mu     sync.Mutex
	values []string
}

func (r *evictRecorder) record(_ string, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values = append(r.values, value)
}

func (r *evictRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.values...)
}

// expire backdates an entry so it is expired without waiting for the TTL.
func expire[K comparable, V any](c *LRU[K, V], key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key].ExpiresAt = time.Now().Add(-time.Minute)
}

func TestGetExpiredEvictsValue(t *testing.T) {
	var rec evictRecorder
	c := NewLRU[string, string](10, rec.record, time.Hour)

	c.Add("key", "old")
	expire(c, "key")

	if _, ok := c.Get("key"); ok {
		t.Fatal("Get() on an expired entry = hit, want miss")
	}
	if got := rec.snapshot(); len(got) != 1 || got[0] != "old" {
		t.Fatalf("onEvict got %q after reading an expired entry, want [old]", got)
	}
	if c.Len() != 0 {
		t.Fatalf("Len() = %d after reading an expired entry, want 0", c.Len())
	}

	// The re-add must be a fresh insert, not a silent overwrite.
	c.Add("key", "new")
	if value, ok := c.Get("key"); !ok || value != "new" {
		t.Fatalf("Get() = %q, %v; want %q, true", value, ok, "new")
	}
	if got := rec.snapshot(); len(got) != 1 {
		t.Fatalf("onEvict got %q overall, want just [old]", got)
	}
}

func TestGetLiveEntryDoesNotEvict(t *testing.T) {
	var rec evictRecorder
	c := NewLRU[string, string](10, rec.record, time.Hour)

	c.Add("key", "value")
	if value, ok := c.Get("key"); !ok || value != "value" {
		t.Fatalf("Get() = %q, %v; want %q, true", value, ok, "value")
	}
	if got := rec.snapshot(); len(got) != 0 {
		t.Fatalf("onEvict got %q for a live entry, want none", got)
	}
	if c.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", c.Len())
	}
}
