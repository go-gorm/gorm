package callbacks

import (
	"database/sql"

	"gorm.io/gorm"
)

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

func Begin(tx *gorm.DB) {
	err := tx.Error

	if err != nil {
		return
	}

	var opt *sql.TxOptions

	if v, ok := tx.InstanceGet("gorm:transaction_options"); ok {
		if txOpts, ok := v.(*sql.TxOptions); ok {
			opt = txOpts
		}
	}

	switch beginner := tx.Statement.ConnPool.(type) {
	case gorm.TxBeginner:
		tx.Statement.ConnPool, err = beginner.BeginTx(tx.Statement.Context, opt)
	case gorm.ConnPoolBeginner:
		tx.Statement.ConnPool, err = beginner.BeginTx(tx.Statement.Context, opt)
	default:
		err = gorm.ErrInvalidTransaction
	}

	if err != nil {
		_ = tx.AddError(err)
	}

}
