// Package lru implements the expiring least-recently-used cache used by GORM's
// prepared-statement store. Go's internal-package rules limit imports to GORM's
// module tree. Use NewLRU to initialize a cache; the zero value is not usable.
//
// # Capacity, recency, and expiry
//
// A positive size limits the number of stored entries; a non-positive size means
// unlimited capacity. Add inserts or replaces a value and renews its expiry. Get
// moves a live entry to the most-recent position without extending its TTL; Peek
// leaves recency unchanged. Capacity eviction removes the least-recent entry.
//
// A non-positive TTL selects a ten-year sentinel and disables background cleanup.
// Other TTLs start a cleanup worker whose ticker period is TTL divided by 100;
// that period must be positive. Expiration checks and physical removal are
// separate: Get and Peek reject expired entries, while Contains, Len, and
// GetOldest may still observe them until cleanup. Keys and KeyValues filter
// expired entries without removing them.
//
// # Concurrency and ownership
//
// Cache operations synchronize their maps and recency list, except that Cap must
// not race with Resize. Values are copied shallowly, so callers remain responsible
// for synchronizing the objects stored in the cache. LruList and Entry expose the
// underlying list mechanics and have no independent synchronization.
//
// An eviction callback runs synchronously while the cache lock is held. It must
// not re-enter the same cache or wait for work requiring that lock. Add replacing
// an existing key does not invoke the callback for the old value, so callers that
// store resources must handle replacement ownership themselves. Removal, purge,
// capacity eviction, and expiry invoke the callback when configured.
//
// The cleanup worker has no public stop method. Purge removes entries but does
// not terminate the worker; creating short-lived caches with positive TTLs can
// retain workers. The cache does not close resources on its own beyond invoking
// the supplied callback.
package lru
