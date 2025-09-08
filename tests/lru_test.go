package tests_test

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm/internal/lru"
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

func BenchmarkLRU_Rand_NoExpire(b *testing.B) {
	l := lru.NewLRU[int64, int64](8192, nil, 0)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = getRand(b) % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkLRU_Freq_NoExpire(b *testing.B) {
	l := lru.NewLRU[int64, int64](8192, nil, 0)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = getRand(b) % 16384
		} else {
			trace[i] = getRand(b) % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		if _, ok := l.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkLRU_Rand_WithExpire(b *testing.B) {
	l := lru.NewLRU[int64, int64](8192, nil, time.Millisecond*10)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = getRand(b) % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkLRU_Freq_WithExpire(b *testing.B) {
	l := lru.NewLRU[int64, int64](8192, nil, time.Millisecond*10)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = getRand(b) % 16384
		} else {
			trace[i] = getRand(b) % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		if _, ok := l.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func TestLRUNoPurge(t *testing.T) {
	lc := lru.NewLRU[string, string](10, nil, 0)

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok := lc.Peek("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	if !lc.Contains("key1") {
		t.Fatalf("should contain key1")
	}
	if lc.Contains("key2") {
		t.Fatalf("should not contain key2")
	}

	v, ok = lc.Peek("key2")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	if !reflect.DeepEqual(lc.Keys(), []string{"key1"}) {
		t.Fatalf("value differs from expected")
	}

	if lc.Resize(0) != 0 {
		t.Fatalf("evicted count differs from expected")
	}
	if lc.Resize(2) != 0 {
		t.Fatalf("evicted count differs from expected")
	}
	lc.Add("key2", "val2")
	if lc.Resize(1) != 1 {
		t.Fatalf("evicted count differs from expected")
	}
}

func TestLRUEdgeCases(t *testing.T) {
	lc := lru.NewLRU[string, *string](2, nil, 0)

	// Adding a nil value
	lc.Add("key1", nil)

	value, exists := lc.Get("key1")
	if value != nil || !exists {
		t.Fatalf("unexpected value or existence flag for key1: value=%v, exists=%v", value, exists)
	}

	// Adding an entry with the same key but different value
	newVal := "val1"
	lc.Add("key1", &newVal)

	value, exists = lc.Get("key1")
	if value != &newVal || !exists {
		t.Fatalf("unexpected value or existence flag for key1: value=%v, exists=%v", value, exists)
	}
}

func TestLRU_Values(t *testing.T) {
	lc := lru.NewLRU[string, string](3, nil, 0)

	lc.Add("key1", "val1")
	lc.Add("key2", "val2")
	lc.Add("key3", "val3")

	values := lc.Values()
	if !reflect.DeepEqual(values, []string{"val1", "val2", "val3"}) {
		t.Fatalf("values differs from expected")
	}
}

// func TestExpirableMultipleClose(_ *testing.T) {
//	lc :=lru.NewLRU[string, string](10, nil, 0)
//	lc.Close()
//	// should not panic
//	lc.Close()
// }

func TestLRUWithPurge(t *testing.T) {
	var evicted []string
	lc := lru.NewLRU(10, func(key string, value string) { evicted = append(evicted, key, value) }, 150*time.Millisecond)

	k, v, ok := lc.GetOldest()
	if k != "" {
		t.Fatalf("should be empty")
	}
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	lc.Add("key1", "val1")

	time.Sleep(100 * time.Millisecond) // not enough to expire
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Get("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	time.Sleep(200 * time.Millisecond) // expire
	v, ok = lc.Get("key1")
	if ok {
		t.Fatalf("should be false")
	}
	if v != "" {
		t.Fatalf("should be nil")
	}

	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1"}) {
		t.Fatalf("value differs from expected")
	}

	// add new entry
	lc.Add("key2", "val2")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	k, v, ok = lc.GetOldest()
	if k != "key2" {
		t.Fatalf("value differs from expected")
	}
	if v != "val2" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

}

func TestLRUWithPurgeEnforcedBySize(t *testing.T) {
	lc := lru.NewLRU[string, string](10, nil, time.Hour)

	for i := 0; i < 100; i++ {
		i := i
		lc.Add(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i))
		v, ok := lc.Get(fmt.Sprintf("key%d", i))
		if v != fmt.Sprintf("val%d", i) {
			t.Fatalf("value differs from expected")
		}
		if !ok {
			t.Fatalf("should be true")
		}
		if lc.Len() > 20 {
			t.Fatalf("length should be less than 20")
		}
	}

	if lc.Len() != 10 {
		t.Fatalf("length differs from expected")
	}
}

func TestLRUConcurrency(t *testing.T) {
	lc := lru.NewLRU[string, string](0, nil, 0)
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func(i int) {
			lc.Add(fmt.Sprintf("key-%d", i/10), fmt.Sprintf("val-%d", i/10))
			wg.Done()
		}(i)
	}
	wg.Wait()
	if lc.Len() != 100 {
		t.Fatalf("length differs from expected")
	}
}

func TestLRUInvalidateAndEvict(t *testing.T) {
	var evicted int
	lc := lru.NewLRU(-1, func(_, _ string) { evicted++ }, 0)

	lc.Add("key1", "val1")
	lc.Add("key2", "val2")

	val, ok := lc.Get("key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if val != "val1" {
		t.Fatalf("value differs from expected")
	}
	if evicted != 0 {
		t.Fatalf("value differs from expected")
	}

	lc.Remove("key1")
	if evicted != 1 {
		t.Fatalf("value differs from expected")
	}
	val, ok = lc.Get("key1")
	if val != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestLoadingExpired(t *testing.T) {
	lc := lru.NewLRU[string, string](0, nil, time.Millisecond*5)

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok := lc.Peek("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	v, ok = lc.Get("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	for {
		result, ok := lc.Get("key1")
		if ok && result == "" {
			t.Fatalf("ok should return a result")
		}
		if !ok {
			break
		}
	}

	time.Sleep(time.Millisecond * 100) // wait for expiration reaper
	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Peek("key1")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	v, ok = lc.Get("key1")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestLRURemoveOldest(t *testing.T) {
	lc := lru.NewLRU[string, string](2, nil, 0)

	if lc.Cap() != 2 {
		t.Fatalf("expect cap is 2")
	}

	k, v, ok := lc.RemoveOldest()
	if k != "" {
		t.Fatalf("should be empty")
	}
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	ok = lc.Remove("non_existent")
	if ok {
		t.Fatalf("should be false")
	}

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Get("key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}

	if !reflect.DeepEqual(lc.Keys(), []string{"key1"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	lc.Add("key2", "val2")
	if !reflect.DeepEqual(lc.Keys(), []string{"key1", "key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 2 {
		t.Fatalf("length differs from expected")
	}

	k, v, ok = lc.RemoveOldest()
	if k != "key1" {
		t.Fatalf("value differs from expected")
	}
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	if !reflect.DeepEqual(lc.Keys(), []string{"key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}
}

func getRand(tb testing.TB) int64 {
	out, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		tb.Fatal(err)
	}
	return out.Int64()
}
