package callbacks

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

func BeforeCreate(db *gorm.DB) {
	// before save
	// before create

	// assign timestamp
}

func SaveBeforeAssociations(db *gorm.DB) {
}

func Create(db *gorm.DB) {
	db.Statement.Build("WITH", "INSERT", "VALUES", "ON_CONFLICT", "RETURNING")
	db.DB.ExecContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	fmt.Println(db.Statement.SQL.String(), db.Statement.Vars)
}

func SaveAfterAssociations(db *gorm.DB) {
}

func AfterCreate(db *gorm.DB) {
	// after save
	// after create
}
