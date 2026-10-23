package lru

// golang -lru
// https://github.com/hashicorp/golang-lru
import (
	"sync"
	"time"
)

// EvictCallback runs synchronously under the cache lock when an entry is removed,
// including purge/resize/expiry. It must not re-enter this cache or block on work
// requiring its lock. Updating an existing key does not invoke it for the old value.
type EvictCallback[K comparable, V any] func(key K, value V)

// LRU is a synchronized least-recently-used cache with insertion/update-based TTL. Get
// refreshes recency but not expiration. Contains, Len, and GetOldest can include
// expired entries pending cleanup; Cap is not synchronized with Resize. Initialize
// through NewLRU, not the zero value.
type LRU[K comparable, V any] struct {
	// size is the maximum entry count; zero means unlimited. It is changed by Resize.
	size int
	// evictList orders entries from most recently used at the front to least recently used
	// at the back.
	evictList *LruList[K, V]
	// items indexes list entries by key; an expired entry can remain until removal or
	// background cleanup.
	items map[K]*Entry[K, V]
	// onEvict runs under mu when an entry is removed. It must not re-enter the cache.
	onEvict EvictCallback[K, V]

	// mu guards maps, recency links, and expiry buckets during cache operations.
	mu sync.RWMutex
	// ttl is the duration applied on insertion or replacement, not extended by Get.
	ttl time.Duration
	// done is observed by the cleanup goroutine but is never closed by an exposed API in
	// this implementation.
	done chan struct{}

	// buckets groups entries for background expiration cleanup rather than scheduling one
	// timer per entry.
	buckets []bucket[K, V]
	// nextCleanupBucket identifies the next expiry bucket to process, wrapping modulo
	// numBuckets.
	nextCleanupBucket uint8
}

// bucket holds entries awaiting expiry cleanup and the latest expiry time added to that
// bucket. Cleanup waits for that time before removing the bucket contents.
type bucket[K comparable, V any] struct {
	// entries indexes entries assigned to this expiry bucket.
	entries map[K]*Entry[K, V]
	// newestEntry is the latest expiry assigned to the bucket; removal does not recompute
	// it.
	newestEntry time.Time
}

// noEvictionTTL is the ten-year TTL sentinel used for non-positive TTL input. It
// disables the background expiry worker, but read-time expiration checks still use the
// stored expiry.
const noEvictionTTL = time.Hour * 24 * 365 * 10

// numBuckets partitions expiration work; the uint8 bucket index requires no more than
// 256 buckets. The cleanup ticker interval is ttl divided by this count.
const numBuckets = 100

// NewLRU creates a cache with a size limit and optional eviction callback. Non-positive
// size is unlimited. Non-positive ttl becomes the ten-year sentinel and disables
// background cleanup; other TTLs must be large enough that ttl/numBuckets is positive
// for time.NewTicker. The cleanup goroutine has no exposed stop method and runs for the
// cache lifetime. Eviction callbacks run under the cache lock and must not re-enter it.
func NewLRU[K comparable, V any](size int, onEvict EvictCallback[K, V], ttl time.Duration) *LRU[K, V] {
	if size < 0 {
		size = 0
	}
	if ttl <= 0 {
		ttl = noEvictionTTL
	}

	res := LRU[K, V]{
		ttl:       ttl,
		size:      size,
		evictList: NewList[K, V](),
		items:     make(map[K]*Entry[K, V]),
		onEvict:   onEvict,
		done:      make(chan struct{}),
	}

	// initialize the buckets
	res.buckets = make([]bucket[K, V], numBuckets)
	for i := 0; i < numBuckets; i++ {
		res.buckets[i] = bucket[K, V]{entries: make(map[K]*Entry[K, V])}
	}

	// enable deleteExpired() running in separate goroutine for cache with non-zero TTL
	//
	// Important: done channel is never closed, so deleteExpired() goroutine will never exit,
	// it's decided to add functionality to close it in the version later than v2.
	if res.ttl != noEvictionTTL {
		go func(done <-chan struct{}) {
			ticker := time.NewTicker(res.ttl / numBuckets)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					res.deleteExpired()
				}
			}
		}(res.done)
	}
	return &res
}

// Purge removes every stored entry and resets the recency list, invoking onEvict for
// each entry under the lock. It does not stop the background cleanup worker.
func (c *LRU[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v.Value)
		}
		delete(c.items, k)
	}
	for _, b := range c.buckets {
		for _, ent := range b.entries {
			delete(b.entries, ent.Key)
		}
	}
	c.evictList.Init()
}

// Add inserts or replaces a value, marks it most recent, and resets expiry. It returns
// true only when capacity eviction occurs. Replacing an existing key does not call
// onEvict for its prior value, so resource-owning callers must handle replacement
// cleanup.
func (c *LRU[K, V]) Add(key K, value V) (evicted bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		c.removeFromBucket(ent) // remove the entry from its current bucket as expiresAt is renewed
		ent.Value = value
		ent.ExpiresAt = now.Add(c.ttl)
		c.addToBucket(ent)
		return false
	}

	// Add new item
	ent := c.evictList.PushFrontExpirable(key, value, now.Add(c.ttl))
	c.items[key] = ent
	c.addToBucket(ent) // adds the entry to the appropriate bucket and sets entry.expireBucket

	evict := c.size > 0 && c.evictList.Length() > c.size
	// Verify size not exceeded
	if evict {
		c.removeOldest()
	}
	return evict
}

// Get returns an unexpired value and marks it most recent without extending TTL. An
// expired entry is reported missing but is not physically removed by this method.
func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ent *Entry[K, V]
	if ent, ok = c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.ExpiresAt) {
			return value, false
		}
		c.evictList.MoveToFront(ent)
		return ent.Value, true
	}
	return
}

// Contains checks physical key presence without updating recency or checking
// expiration. It can return true while Get reports the same key expired.
func (c *LRU[K, V]) Contains(key K) (ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok = c.items[key]
	return ok
}

// Peek returns an unexpired value without changing recency or TTL. Expired entries are
// reported missing but remain stored until cleanup.
func (c *LRU[K, V]) Peek(key K) (value V, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var ent *Entry[K, V]
	if ent, ok = c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.ExpiresAt) {
			return value, false
		}
		return ent.Value, true
	}
	return
}

// Remove deletes key if present and invokes onEvict under the lock. It reports physical
// presence, including an expired entry waiting for cleanup.
func (c *LRU[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest removes the least recently used stored entry and invokes onEvict. It can
// remove an already expired entry; ok=false means the list is empty.
func (c *LRU[K, V]) RemoveOldest() (key K, value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent)
		return ent.Key, ent.Value, true
	}
	return
}

// GetOldest returns the least recently used stored entry without checking expiration or
// changing recency. Use Get/Peek when live-value semantics are required.
func (c *LRU[K, V]) GetOldest() (key K, value V, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if ent := c.evictList.Back(); ent != nil {
		return ent.Key, ent.Value, true
	}
	return
}

// KeyValues returns a new map of entries unexpired at the snapshot time. Values are
// copied shallowly; referenced objects remain shared.
func (c *LRU[K, V]) KeyValues() map[K]V {
	c.mu.RLock()
	defer c.mu.RUnlock()
	maps := make(map[K]V)
	now := time.Now()
	for ent := c.evictList.Back(); ent != nil; ent = ent.PrevEntry() {
		if now.After(ent.ExpiresAt) {
			continue
		}
		maps[ent.Key] = ent.Value
		// keys = append(keys, ent.Key)
	}
	return maps
}

// Keys returns unexpired keys in least-to-most-recent order without changing recency.
// Expired entries are filtered but not deleted.
func (c *LRU[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]K, 0, len(c.items))
	now := time.Now()
	for ent := c.evictList.Back(); ent != nil; ent = ent.PrevEntry() {
		if now.After(ent.ExpiresAt) {
			continue
		}
		keys = append(keys, ent.Key)
	}
	return keys
}

// Values returns unexpired values in least-to-most-recent order. The slice is new but
// reference-valued entries are not deep-copied.
func (c *LRU[K, V]) Values() []V {
	c.mu.RLock()
	defer c.mu.RUnlock()
	values := make([]V, 0, len(c.items))
	now := time.Now()
	for ent := c.evictList.Back(); ent != nil; ent = ent.PrevEntry() {
		if now.After(ent.ExpiresAt) {
			continue
		}
		values = append(values, ent.Value)
	}
	return values
}

// Len reports stored entries, including expired entries awaiting cleanup. It is not the
// same as len(Keys()) when expiration has occurred.
func (c *LRU[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.evictList.Length()
}

// Resize changes the capacity and removes least-recent entries until the new positive
// limit is met, returning the number removed. Non-positive size makes capacity
// unlimited. Eviction callbacks run under the lock.
func (c *LRU[K, V]) Resize(size int) (evicted int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if size <= 0 {
		c.size = 0
		return 0
	}
	diff := c.evictList.Length() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	c.size = size
	return diff
}

// Close destroys cleanup goroutine. To clean up the cache, run Purge() before Close().
// func (c *LRU[K, V]) Close() {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	select {
//	case <-c.done:
//		return
//	default:
//	}
//	close(c.done)
// }

// removeOldest removes the back entry if present. The caller must hold the write lock;
// removal invokes the eviction callback.
func (c *LRU[K, V]) removeOldest() {
	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent)
	}
}

// removeElement unlinks e from the recency list, key map, and expiry bucket, then
// invokes onEvict. The caller must hold the write lock and supply an entry owned by
// this cache.
func (c *LRU[K, V]) removeElement(e *Entry[K, V]) {
	c.evictList.Remove(e)
	delete(c.items, e.Key)
	c.removeFromBucket(e)
	if c.onEvict != nil {
		c.onEvict(e.Key, e.Value)
	}
}

// deleteExpired processes the next expiry bucket, temporarily releasing the lock while
// waiting for its newest recorded expiry, then removes its entries and advances the
// bucket index. Cleanup can lag read-time expiration.
func (c *LRU[K, V]) deleteExpired() {
	c.mu.Lock()
	bucketIdx := c.nextCleanupBucket
	timeToExpire := time.Until(c.buckets[bucketIdx].newestEntry)
	// wait for newest entry to expire before cleanup without holding lock
	if timeToExpire > 0 {
		c.mu.Unlock()
		time.Sleep(timeToExpire)
		c.mu.Lock()
	}
	for _, ent := range c.buckets[bucketIdx].entries {
		c.removeElement(ent)
	}
	c.nextCleanupBucket = (c.nextCleanupBucket + 1) % numBuckets
	c.mu.Unlock()
}

// addToBucket assigns an entry to the bucket before the next cleanup position and
// extends that bucket's newest expiry if necessary. The caller must hold the write
// lock.
func (c *LRU[K, V]) addToBucket(e *Entry[K, V]) {
	bucketID := (numBuckets + c.nextCleanupBucket - 1) % numBuckets
	e.ExpireBucket = bucketID
	c.buckets[bucketID].entries[e.Key] = e
	if c.buckets[bucketID].newestEntry.Before(e.ExpiresAt) {
		c.buckets[bucketID].newestEntry = e.ExpiresAt
	}
}

// removeFromBucket removes an entry from its recorded expiry bucket without changing
// the bucket's newest-expiry timestamp. The caller must hold the write lock.
func (c *LRU[K, V]) removeFromBucket(e *Entry[K, V]) {
	delete(c.buckets[e.ExpireBucket].entries, e.Key)
}

// Cap returns the configured capacity, with zero meaning unlimited. It does not lock;
// do not call it concurrently with Resize without external synchronization.
func (c *LRU[K, V]) Cap() int {
	return c.size
}

// Entry is a node in LruList holding a key/value and optional expiration metadata. List
// links are internal and not independently synchronized; use the owning LRU for
// concurrent cache access.
type Entry[K comparable, V any] struct {
	// next and prev form a doubly linked ring through the owning list's sentinel. Removed
	// entries have both links cleared.
	next, prev *Entry[K, V]

	// list identifies the owning list or is nil after removal.
	list *LruList[K, V]

	// Key is the cache lookup key associated with this node.
	Key K

	// Value is the cached value; referenced objects are not copied by the list.
	Value V

	// ExpiresAt is the absolute expiry checked by LRU reads and cleanup scheduling.
	ExpiresAt time.Time

	// ExpireBucket indexes the owning LRU's expiration bucket for removal/rescheduling.
	ExpireBucket uint8
}

// PrevEntry returns the preceding node toward the front, or nil at the sentinel or
// after removal. It requires the caller to provide appropriate synchronization.
func (e *Entry[K, V]) PrevEntry() *Entry[K, V] {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// LruList is an unsynchronized doubly linked ring used by LRU. Its zero value is lazily
// initialized by insertion; do not copy a populated list because node links point to
// its sentinel.
type LruList[K comparable, V any] struct {
	// root is the sentinel ring node, excluded from the element count.
	root Entry[K, V] // sentinel list element, only &root, root.prev, and root.next are used
	// len counts data nodes, excluding root.
	len int // current list Length excluding (this) sentinel element
}

// Init resets the sentinel links and length and returns the list. It does not visit or
// detach previously held nodes; callers must discard old entries when clearing the
// list.
func (l *LruList[K, V]) Init() *LruList[K, V] {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

// NewList allocates an initialized empty list. It provides no locking; LRU supplies
// synchronization around list operations.
func NewList[K comparable, V any]() *LruList[K, V] { return new(LruList[K, V]).Init() }

// Length returns the stored element count in constant time without locking.
func (l *LruList[K, V]) Length() int { return l.len }

// Back returns the least-recent end node or nil when the list is empty. The list does
// not check entry expiration.
func (l *LruList[K, V]) Back() *Entry[K, V] {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// lazyInit links the zero-value sentinel to itself before insertion.
func (l *LruList[K, V]) lazyInit() {
	if l.root.next == nil {
		l.Init()
	}
}

// insert links e immediately after at, sets its owner, and increments length. at must
// belong to this initialized list and e must be suitable for insertion.
func (l *LruList[K, V]) insert(e, at *Entry[K, V]) *Entry[K, V] {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.list = l
	l.len++
	return e
}

// insertValue allocates a key/value node with expiresAt and inserts it after at. It
// assumes the list and insertion point are valid.
func (l *LruList[K, V]) insertValue(k K, v V, expiresAt time.Time, at *Entry[K, V]) *Entry[K, V] {
	return l.insert(&Entry[K, V]{Value: v, Key: k, ExpiresAt: expiresAt}, at)
}

// Remove unlinks e, clears its owner and links, decrements length, and returns its
// value. e must be a non-sentinel member of this list; membership is not checked here.
func (l *LruList[K, V]) Remove(e *Entry[K, V]) V {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	e.list = nil
	l.len--

	return e.Value
}

// move relinks e after at without changing length. Both nodes must belong to this list;
// moving a node after itself is a no-op.
func (l *LruList[K, V]) move(e, at *Entry[K, V]) {
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
}

// PushFront inserts a key/value at the front with a zero expiry. It is a list
// operation; zero expiry is not an unexpired-entry promise to an LRU reader.
func (l *LruList[K, V]) PushFront(k K, v V) *Entry[K, V] {
	l.lazyInit()
	return l.insertValue(k, v, time.Time{}, &l.root)
}

// PushFrontExpirable inserts a node at the front with the supplied absolute expiration
// time. It lazily initializes the list but does not schedule cleanup itself.
func (l *LruList[K, V]) PushFrontExpirable(k K, v V, expiresAt time.Time) *Entry[K, V] {
	l.lazyInit()
	return l.insertValue(k, v, expiresAt, &l.root)
}

// MoveToFront moves a non-nil member to the front. It leaves the list unchanged for an
// entry owned by another list or one already at the front; callers provide
// synchronization.
func (l *LruList[K, V]) MoveToFront(e *Entry[K, V]) {
	if e.list != l || l.root.next == e {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, &l.root)
}
