package callbacks

import (
	"gorm.io/gorm"
)

// BeginTransaction starts the default write transaction when enabled and db has no
// error. On success it swaps the statement's connection pool and marks this operation
// as owning the transaction. ErrInvalidTransaction is ignored, allowing operations
// already on a transaction to proceed; other begin errors are recorded. It does not
// create nested savepoints.
func BeginTransaction(db *gorm.DB) {
	if !db.Config.SkipDefaultTransaction && db.Error == nil {
		if tx := db.Begin(); tx.Error == nil {
			db.Statement.ConnPool = tx.Statement.ConnPool
			db.InstanceSet("gorm:started_transaction", true)
		} else if tx.Error == gorm.ErrInvalidTransaction {
			tx.Error = nil
		} else {
			db.Error = tx.Error
		}
	}
}

// CommitOrRollbackTransaction finishes only a transaction marked as started by
// BeginTransaction, when default transactions are enabled. An existing db.Error causes
// rollback; otherwise it commits and records any commit error. It then restores the
// configured connection pool. It does not commit a caller-owned explicit transaction.
func CommitOrRollbackTransaction(db *gorm.DB) {
	if !db.Config.SkipDefaultTransaction {
		if _, ok := db.InstanceGet("gorm:started_transaction"); ok {
			if db.Error != nil {
				db.Rollback()
			} else {
				db.Commit()
			}

			db.Statement.ConnPool = db.ConnPool
		}
	}
}
