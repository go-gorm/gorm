package gorm

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cacheItem struct {
	dataMutex   sync.RWMutex
	data        interface{}
	err         error
	created     int64
	accessMutex sync.RWMutex
	accessCount int64
}

type cache struct {
	size          int
	highWaterMark int
	enabled       bool
	idMapMutex    sync.RWMutex
	idMapping     map[modelId][]string
	database      map[string]*cacheItem
	mutex         sync.RWMutex
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

	fmt.Println("Cache High Water Mark: ", c.highWaterMark)
	fmt.Println("Cache Size: ", c.size)

	c.database = make(map[string]*cacheItem, c.size*2) // Size is larger to allow for temporary bursting
	c.idMapping = make(map[modelId][]string, 100)

	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				c.Empty()
			}
		}
	}()

	c.enabled = true
}

type KeyValue struct {
	Key   string
	Value *cacheItem
}

func (c cache) Empty() {
	if len(c.database) > c.size {
		fmt.Println("Over the limit. Running cleanup")

		var s []KeyValue
		c.mutex.RLock()
		for k, v := range c.database {
			s = append(s, KeyValue{
				Key:   k,
				Value: v,
			})
		}
		c.mutex.RUnlock()

		// Sort the results
		sort.Slice(s, func(i, j int) bool {
			return s[i].Value.accessCount < s[j].Value.accessCount
		})

		// Go through the end of the results list and knock those keys off
		c.mutex.Lock()
		for _, res := range s[c.highWaterMark : len(s)-1] {
			fmt.Println("Cleaned up query " + res.Key + " having only " + strconv.Itoa(int(res.Value.accessCount)) + " accesses.")
			delete(c.database, res.Key)
		}
		c.mutex.Unlock()
	}
}

func (c cache) GetItem(key string, offset int64) (interface{}, error) {
	fmt.Print("Getting item " + key + " ... ")

	c.mutex.RLock()
	if item, ok := c.database[key]; ok {
		item.accessMutex.Lock()
		item.accessCount++
		item.accessMutex.Unlock()

		item.dataMutex.RLock()
		defer item.dataMutex.RUnlock()

		if (item.created+(offset*1000000000) > time.Now().UnixNano()) || offset == -1 {
			fmt.Print("Found \n")
			c.mutex.RUnlock()
			return item.data, item.err
		}

		fmt.Print("Expired \n")
	} else {
		fmt.Print("Not found \n")
	}

	c.mutex.RUnlock()
	return nil, nil
}

type modelId struct {
	model string
	id    string
}

func (c *cache) StoreItem(key string, data interface{}, errors error) {
	fmt.Println("Storing item " + key)

	// Affected IDs
	affectedIDs := make([]string, 0, 100)
	var model string

	// Go through the IDs in the interface and add them and the model to the
	switch reflect.TypeOf(data).Kind() {
	case reflect.Slice:
		// Loop through each of the items and get the primary key or "ID" value
		s := reflect.ValueOf(data)
		model = reflect.TypeOf(data).Elem().String()

		for i := 0; i < s.Len(); i++ {
			affectedIDs = append(affectedIDs, getID(s.Index(i).Interface()))
		}

	case reflect.Struct:
		model = reflect.TypeOf(data).String()
		affectedIDs = []string{getID(data)}
	}

	if _, ok := c.database[key]; !ok {
		c.mutex.Lock()
		c.database[key] = &cacheItem{
			created:     time.Now().UnixNano(),
			accessCount: 1,
			data:        data,
			err:         errors,
		}
		c.mutex.Unlock()
	} else {
		c.mutex.RLock()
		c.database[key].dataMutex.Lock()
		c.database[key].data = data
		c.database[key].err = errors
		c.database[key].created = time.Now().UnixNano()
		c.database[key].dataMutex.Unlock()
		c.mutex.RUnlock()
	}

	// Store the query selector agains the relevent IDs
	c.idMapMutex.Lock()
	for _, id := range affectedIDs {
		sel := modelId{model: model, id: id}

		if _, ok := c.idMapping[sel]; !ok {
			// We need to create the array
			c.idMapping[sel] = []string{key}
		} else {
			c.idMapping[sel] = append(c.idMapping[sel], key)
		}
	}
	c.idMapMutex.Unlock()
}

func (c *cache) Expireitem(model, id string) {
	// Get the relevent cache items
	sel := modelId{model: model, id: id}
	c.idMapMutex.Lock()
	items := c.idMapping[sel]
	delete(c.idMapping, sel)
	c.idMapMutex.Unlock()

	// Delete the items from the cache
	c.mutex.Lock()
	for _, key := range items {
		fmt.Println("Expiring item " + key + "(based on " + model + "/" + id)
		delete(c.database, key)
	}
	c.mutex.Unlock()
}

func getID(data interface{}) string {
	d := reflect.ValueOf(data)
	idField := d.FieldByName("ID")

	if idField.IsValid() {
		return fmt.Sprint(idField.Interface())
	}

	// We haven't found an id the easy way so instead go through all of the primary key fields
	// From those fields, get the value and concat using / as a seperator
	idParts := []string{}
	intType := reflect.TypeOf(data)
	for i := 0; i < intType.NumField(); i++ {
		tag := intType.Field(i).Tag
		if strings.Contains(tag.Get("gorm"), "primary_key") {
			idParts = append(idParts, d.Field(i).String())
		}
	}

	if len(idParts) > 0 {
		return strings.Join(idParts, "/")
	}

	return ""
}
