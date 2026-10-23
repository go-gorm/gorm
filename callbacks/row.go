package callbacks

import (
	"gorm.io/gorm"
)

// RowQuery builds SELECT SQL and, unless DryRun or an earlier error prevents execution,
// stores either *sql.Rows or *sql.Row in Statement.Dest. The internal rows setting
// selects QueryContext versus QueryRowContext. It sets RowsAffected to -1 because it
// does not scan or count results. The caller owns Rows.Close/Rows.Err or Row.Scan;
// row-query errors may be deferred until Scan. It does not preload associations or
// invoke AfterFind.
func RowQuery(db *gorm.DB) {
	if db.Error == nil {
		BuildQuerySQL(db)
		if db.DryRun || db.Error != nil {
			return
		}

		if isRows, ok := db.Get("rows"); ok && isRows.(bool) {
			db.Statement.Settings.Delete("rows")
			db.Statement.Dest, db.Error = db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		} else {
			db.Statement.Dest = db.Statement.ConnPool.QueryRowContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		}

		db.RowsAffected = -1
	}
}
