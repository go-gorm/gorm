package callbacks

import (
	"gorm.io/gorm"
)

func RowQuery(db *gorm.DB) {
	if db.Error == nil {
		BuildQuerySQL(db)
		if db.DryRun || db.Error != nil {
			return
		}

		if types, ok := db.Statement.Settings.Load("rows"); ok && types.(*gorm.QueryTypes).Pop() {
			db.Statement.Dest, db.Error = db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		} else {
			db.Statement.Dest = db.Statement.ConnPool.QueryRowContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		}

		db.RowsAffected = -1
	}
}
