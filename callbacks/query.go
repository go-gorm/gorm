package callbacks

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func Query(db *gorm.DB) {
	if db.Statement.SQL.String() == "" {
		db.Statement.AddClauseIfNotExists(clause.Select{})
		db.Statement.AddClauseIfNotExists(clause.From{})

		db.Statement.Build("SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR")
	}

	result, err := db.DB.ExecContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	fmt.Println(err)
	fmt.Println(result)
	fmt.Println(db.Statement.SQL.String(), db.Statement.Vars)
}

func Preload(db *gorm.DB) {
}

func AfterQuery(db *gorm.DB) {
	// after find
}
