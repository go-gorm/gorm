package postgres

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
	"github.com/jinzhu/gorm/logger"
	"github.com/jinzhu/gorm/migrator"
	"github.com/jinzhu/gorm/schema"
	_ "github.com/lib/pq"
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
	db.ConnPool, err = sql.Open("postgres", dialector.DSN)
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
	return "$" + strconv.Itoa(len(stmt.Vars))
}

func (dialector Dialector) QuoteTo(builder *strings.Builder, str string) {
	builder.WriteByte('"')
	builder.WriteString(str)
	builder.WriteByte('"')
}

var numericPlaceholder = regexp.MustCompile("\\$(\\d+)")

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, numericPlaceholder, `'`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		if field.AutoIncrement {
			switch {
			case field.Size < 16:
				return "smallserial"
			case field.Size < 31:
				return "serial"
			default:
				return "bigserial"
			}
		} else {
			switch {
			case field.Size < 16:
				return "smallint"
			case field.Size < 31:
				return "integer"
			default:
				return "bigint"
			}
		}
	case schema.Float:
		return "decimal"
	case schema.String:
		if field.Size > 0 {
			return fmt.Sprintf("varchar(%d)", field.Size)
		}
		return "text"
	case schema.Time:
		return "timestamp with time zone"
	case schema.Bytes:
		return "bytea"
	}

	return ""
}
