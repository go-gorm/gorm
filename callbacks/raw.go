package callbacks

import (
	"gorm.io/gorm"
)

// RawExec executes the statement's existing SQL and Vars with its context when db has
// no error and DryRun is false. It records execution errors, RowsAffected, and an
// optional native sql.Result collector. It does not build SQL, load a model, run model
// hooks, or start a transaction.
func RawExec(db *gorm.DB) {
	if db.Error == nil && !db.DryRun {
		result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		if err != nil {
			db.AddError(err)
			return
		}

		db.RowsAffected, _ = result.RowsAffected()

		if db.Statement.Result != nil {
			db.Statement.Result.Result = result
			db.Statement.Result.RowsAffected = db.RowsAffected
		}
	}
}
