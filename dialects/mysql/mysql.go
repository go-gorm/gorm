package mysql

import (
	"database/sql"
	"fmt"
	"math"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
	"github.com/jinzhu/gorm/clause"
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
	db.ConnPool, err = sql.Open("mysql", dialector.DSN)
	return
}

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{
		DB:        db,
		Dialector: dialector,
	}}}
}

func (dialector Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (dialector Dialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	writer.WriteString(str)
	writer.WriteByte('`')
}

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		sqlType := "int"
		switch {
		case field.Size <= 8:
			sqlType = "tinyint"
		case field.Size <= 16:
			sqlType = "smallint"
		case field.Size <= 32:
			sqlType = "int"
		default:
			sqlType = "bigint"
		}

		if field.DataType == schema.Uint {
			sqlType += " unsigned"
		}

		if field.AutoIncrement || field == field.Schema.PrioritizedPrimaryField {
			sqlType += " AUTO_INCREMENT"
		}
		return sqlType
	case schema.Float:
		if field.Size <= 32 {
			return "float"
		}
		return "double"
	case schema.String:
		size := field.Size
		if field.PrimaryKey && size == 0 {
			size = 256
		}

		if size >= 65536 && size <= int(math.Pow(2, 24)) {
			return "mediumtext"
		} else if size > int(math.Pow(2, 24)) || size <= 0 {
			return "longtext"
		}
		return fmt.Sprintf("varchar(%d)", size)
	case schema.Time:
		precision := ""
		if field.Precision == 0 {
			field.Precision = 3
		}

		if field.Precision > 0 {
			precision = fmt.Sprintf("(%d)", field.Precision)
		}

		if field.NotNull || field.PrimaryKey {
			return "datetime" + precision
		}
		return "datetime" + precision + " NULL"
	case schema.Bytes:
		if field.Size > 0 && field.Size < 65536 {
			return fmt.Sprintf("varbinary(%d)", field.Size)
		}

		if field.Size >= 65536 && field.Size <= int(math.Pow(2, 24)) {
			return "mediumblob"
		}

		return "longblob"
	}

	return ""
}
