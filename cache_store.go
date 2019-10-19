package gorm

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

type cacheItem struct {
	dataMutex   sync.RWMutex
	data        interface{}
	created     int64
	accessMutex sync.RWMutex
	accessCount int64
}

type cache struct {
	size          int
	highWaterMark int
	enabled       bool
	database      map[string]*cacheItem
}

func (c *cache) Enable() {
	// Kick off the maintenance loop
	size := os.Getenv("QUERY_CACHE_SIZE")
	if size == "" {
		size = "8192"
	}

	highWaterMark := os.Getenv("QUERY_CACHE_HIGH_WATER")
	if highWaterMark == "" {
		highWaterMark = "6192"
	}

	c.size, _ = strconv.Atoi(size)
	c.highWaterMark, _ = strconv.Atoi(highWaterMark)

	c.database = make(map[string]*cacheItem, c.size)

	c.enabled = true
}

func (c cache) GetItem(key string, offset int64) interface{} {
	fmt.Println("Getting item " + key)

	if item, ok := c.database[key]; ok {
		item.dataMutex.RLock()
		item.accessMutex.Lock()

		defer item.dataMutex.RUnlock()
		defer item.accessMutex.Unlock()

		item.accessCount++

		if (item.created+offset < time.Now().Unix()) || offset == -1 {
			return item.data
		}
	}

	return nil
}

func (c *cache) StoreItem(key string, data interface{}) {
	fmt.Println("Storing item " + key)

	if _, ok := c.database[key]; !ok {
		c.database[key] = &cacheItem{
			data:    data,
			created: time.Now().Unix(),
		}
	} else {
		c.database[key].dataMutex.Lock()
		c.database[key].data = data
		c.database[key].created = time.Now().Unix()
		c.database[key].dataMutex.Unlock()
	}
}
