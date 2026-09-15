package callbacks

import (
	"database/sql"

	"gorm.io/gorm"
)

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

func ScanRows(db *gorm.DB) {
	if db.Error != nil {
		return
	}

	dest, ok := db.Get("scan-dest")
	if !ok {
		// The callback was initiated from `db.Row()` or `db.Rows()`
		// In that case we don't want to scan the rows.
		return
	}
	db.Statement.Settings.Delete("scan-dest")

	rows, ok := db.Statement.Dest.(*sql.Rows)
	if !ok && db.DryRun && db.Error == nil {
		_ = db.AddError(gorm.ErrDryRunModeUnsupported)
		return
	}

	defer func() {
		if err := rows.Close(); err != nil {
			_ = db.AddError(err)
		}
	}()

	if rows.Next() {
		_ = db.ScanRows(rows, dest)
		return
	} else {
		db.RowsAffected = 0
		_ = db.AddError(rows.Err())
	}
}
