package mssql

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
	"github.com/jinzhu/gorm/logger"
	"github.com/jinzhu/gorm/migrator"
	"github.com/jinzhu/gorm/schema"
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

	db.DB, err = sql.Open("sqlserver", dialector.DSN)
	return
}

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   dialector,
		CreateIndexAfterCreateTable: true,
	}}}
}

func (dialector Dialector) BindVar(stmt *gorm.Statement, v interface{}) string {
	return "@p" + strconv.Itoa(len(stmt.Vars))
}

func (dialector Dialector) QuoteChars() [2]byte {
	return [2]byte{'"', '"'} // `name`
}

var numericPlaceholder = regexp.MustCompile("@p(\\d+)")

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, numericPlaceholder, `'`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "bit"
	case schema.Int, schema.Uint:
		var sqlType string
		switch {
		case field.Size < 16:
			sqlType = "smallint"
		case field.Size < 31:
			sqlType = "int"
		default:
			sqlType = "bigint"
		}

		if field.AutoIncrement {
			return sqlType + " IDENTITY(1,1)"
		}
		return sqlType
	case schema.Float:
		return "decimal"
	case schema.String:
		size := field.Size
		if field.PrimaryKey && size == 0 {
			size = 256
		}
		if size > 0 && size <= 4000 {
			return fmt.Sprintf("nvarchar(%d)", size)
		}
		return "ntext"
	case schema.Time:
		return "datetimeoffset"
	case schema.Bytes:
		return "binary"
	}

	return ""
}
