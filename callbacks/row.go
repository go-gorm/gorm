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

		// Prefer the direct Statement.RowsMode bool over the legacy
		// Settings["rows"] sync.Map dance. Rows() sets the field to
		// avoid the sync.Map.Store alloc; we reset it here so a reused
		// statement in a transaction can go back to QueryRowContext.
		// Fall back to the sync.Map lookup for third-party callers that
		// still push the flag via Set (kept for backward compatibility).
		if db.Statement.RowsMode {
			db.Statement.RowsMode = false
			db.Statement.Dest, db.Error = db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		} else if isRows, ok := db.Get("rows"); ok && isRows.(bool) {
			db.Statement.Settings.Delete("rows")
			db.Statement.Dest, db.Error = db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		} else {
			db.Statement.Dest = db.Statement.ConnPool.QueryRowContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		}

		db.RowsAffected = -1
	}
}
