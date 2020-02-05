package sqlite

import (
	"database/sql"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
	_ "github.com/mattn/go-sqlite3"
)

type Dialector struct {
	DSN string
}

func Open(dsn string) gorm.Dialector {
	return &Dialector{DSN: dsn}
}

func (dialector Dialector) Initialize(db *gorm.DB) (err error) {
	// register callbacks
	callbacks.RegisterDefaultCallbacks(db)

	db.DB, err = sql.Open("sqlite3", dialector.DSN)
	return
}

func (Dialector) Migrator() gorm.Migrator {
	return nil
}

func (Dialector) BindVar(stmt *gorm.Statement, v interface{}) string {
	return "?"
}

func (Dialector) QuoteChars() [2]byte {
	return [2]byte{'`', '`'} // `name`
}
