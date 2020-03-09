package callbacks

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func RowQuery(db *gorm.DB) {
	if db.Statement.SQL.String() == "" {
		db.Statement.AddClauseIfNotExists(clause.Select{})
		db.Statement.AddClauseIfNotExists(clause.From{})

		db.Statement.Build("SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR")
	}

	if _, ok := db.Get("rows"); ok {
		db.Statement.Dest, db.Error = db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	} else {
		db.Statement.Dest = db.Statement.ConnPool.QueryRowContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	}
}
