package sqlite

import (
	"database/sql"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
	"github.com/jinzhu/gorm/migrator"
	"github.com/jinzhu/gorm/schema"
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

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{DB: db}}}
}

func (dialector Dialector) BindVar(stmt *gorm.Statement, v interface{}) string {
	return "?"
}

func (dialector Dialector) QuoteChars() [2]byte {
	return [2]byte{'`', '`'} // `name`
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "NUMERIC"
	case schema.Int, schema.Uint:
		if field.AutoIncrement {
			// https://www.sqlite.org/autoinc.html
			return "INTEGER PRIMARY KEY AUTOINCREMENT"
		} else {
			return "INTEGER"
		}
	case schema.Float:
		return "REAL"
	case schema.String, schema.Time:
		return "TEXT"
	case schema.Bytes:
		return "BLOB"
	}

	return ""
}
