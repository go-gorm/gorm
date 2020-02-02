package mysql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
)

type Dialector struct {
}

func Open(dsn string) gorm.Dialector {
	return &Dialector{}
}

func (Dialector) Initialize(db *gorm.DB) error {
	// register callbacks
	callbacks.RegisterDefaultCallbacks(db)

	return nil
}

func (Dialector) Migrator() gorm.Migrator {
	return nil
}

func (Dialector) BindVar(stmt gorm.Statement, v interface{}) string {
	return "?"
}
