package callbacks

import (
	"gorm.io/gorm"
)

func RowQuery(db *gorm.DB) {
	if db.Error == nil {
		BuildQuerySQL(db)

		if !db.DryRun {
			if isRows, ok := db.InstanceGet("rows"); ok && isRows.(bool) {
				db.Statement.Dest, db.Error = db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
			} else {
				db.Statement.Dest = db.Statement.ConnPool.QueryRowContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
			}

			db.RowsAffected = -1
		}
	}
}
