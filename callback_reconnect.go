package gorm

import (
	"time"
)

//define callbacks to reconnect in case of failure
func init() {
	DefaultCallback.Create().After("gorm:create").Register("gorm:begin_reconnect", performReconnect)
	DefaultCallback.Update().After("gorm:update").Register("gorm:begin_reconnect", performReconnect)
	DefaultCallback.Delete().After("gorm:delete").Register("gorm:begin_reconnect", performReconnect)
	DefaultCallback.Query().After("gorm:query").Register("gorm:begin_reconnect", performReconnect)
}

//May be do some kind of settings?
const reconnectAttempts = 5
const reconnectInterval = 5 * time.Second

//performReconnect the callback used to peform some reconnect attempts in case of disconnect
func performReconnect(scope *Scope) {
	if scope.HasError() {

		scope.db.reconnectGuard.Add(1)
		defer scope.db.reconnectGuard.Done()

		err := scope.db.Error

		if scope.db.dialect.IsDisconnectError(err) {
			for i := 0; i < reconnectAttempts; i++ {
				newDb, openErr := Open(scope.db.dialectName, scope.db.dialectArgs...)
				if openErr == nil {
					oldDb := scope.db
					if oldDb.parent != oldDb {
						//In case of cloned db try to fix parents
						//It is thread safe as we share mutex between instances
						fixParentDbs(oldDb, newDb)
					}
					*scope.db = *newDb
					break
				} else {
					//wait for interval and try to reconnect again
					<-time.After(reconnectInterval)
				}
			}
		}
	}
}

func fixParentDbs(current, newDb *DB) {
	iterator := current
	parent := current.parent

	for {
		oldParent := parent
		*parent = *newDb
		parent = oldParent.parent
		iterator = oldParent
		if iterator == parent {
			break
		}
	}
}
