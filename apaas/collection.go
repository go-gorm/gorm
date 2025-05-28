package apaas

import (
	"sync"
	"sync/atomic"
)

var gDBCol atomic.Value

func init() {
	dbCol := &DBCollection{
		dbs: make(map[string]*DBMeta, 128),
	}
	SetDBCol(dbCol)
}

func SetDBCol(dbCol *DBCollection) {
	if dbCol != nil {
		gDBCol.Store(dbCol)
	}
}

func GetDBCol() *DBCollection {
	dbCol, ok := gDBCol.Load().(*DBCollection)
	if ok {
		return dbCol
	}
	return nil
}

type DBCollection struct {
	dbs  map[string]*DBMeta
	lock sync.RWMutex
}

func (p *DBCollection) GetDB(dbName string) (*DBMeta, bool) {
	p.lock.RLock()
	v, ok := p.dbs[dbName]
	p.lock.RUnlock()
	return v, ok
}

func (p *DBCollection) SetDB(dbName string, dbMeta *DBMeta) {
	p.lock.Lock()
	p.dbs[dbName] = dbMeta
	p.lock.Unlock()
}

func (p *DBCollection) DeleteDB(dbName string) {
	p.lock.Lock()
	delete(p.dbs, dbName)
	p.lock.Unlock()
}
