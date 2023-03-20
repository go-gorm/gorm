package gorm

import (
	"sync"
)

type easedTask struct {
	wg *sync.WaitGroup
	db *DB
}
