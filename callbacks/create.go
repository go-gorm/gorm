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

	fmt.Println(db.Statement.SQL.String(), db.Statement.Vars)
}

func SaveAfterAssociations(db *gorm.DB) {
}

func AfterCreate(db *gorm.DB) {
	// after save
	// after create
}

func objectToFieldsMap(stmt *gorm.Statement) {
	if stmt.Schema != nil {
		if s, ok := stmt.Clauses["SELECT"]; ok {
			s.Attrs
		}

		if s, ok := stmt.Clauses["OMIT"]; ok {
			s.Attrs
		}

		stmt.Schema.LookUpField(s.S)
	}
}
