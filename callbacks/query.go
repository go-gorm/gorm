package callbacks

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func Query(db *gorm.DB) {
	if db.Statement.SQL.String() == "" {
		db.Statement.AddClauseIfNotExists(clause.Select{})
		db.Statement.AddClauseIfNotExists(clause.From{})

		db.Statement.Build("SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR")
	}

	rows, err := db.DB.QueryContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	db.AddError(err)
}

func Preload(db *gorm.DB) {
}

func AfterQuery(db *gorm.DB) {
	// after find
}
