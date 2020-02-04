package callbacks

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func BeforeCreate(db *gorm.DB) {
	// before save
	// before create

	// assign timestamp
}

func SaveBeforeAssociations(db *gorm.DB) {
}

func Create(db *gorm.DB) {
	db.Statement.AddClauseIfNotExists(clause.Insert{
		Table: clause.Table{Table: db.Statement.Table},
	})

	db.Statement.Build("INSERT", "VALUES", "ON_CONFLICT")
	result, err := db.DB.ExecContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	fmt.Println(err)
	fmt.Println(result)
	fmt.Println(db.Statement.SQL.String(), db.Statement.Vars)
}

func SaveAfterAssociations(db *gorm.DB) {
}

func AfterCreate(db *gorm.DB) {
	// after save
	// after create
}
