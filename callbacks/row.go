package callbacks

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func RowQuery(db *gorm.DB) {
	db.Statement.AddClauseIfNotExists(clause.Select{})
	db.Statement.AddClauseIfNotExists(clause.From{})

	db.Statement.Build("SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR")

	if _, ok := db.Get("rows"); ok {
		db.Statement.Dest, db.Error = db.DB.QueryContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	} else {
		db.Statement.Dest = db.DB.QueryRowContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	}
}
