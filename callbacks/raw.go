package callbacks

import "github.com/jinzhu/gorm"

func RawExec(db *gorm.DB) {
	result, err := db.DB.ExecContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	db.RowsAffected, _ = result.RowsAffected()
	if err != nil {
		db.AddError(err)
	}
}
