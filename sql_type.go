package gorm

import (
	"database/sql"
	"fmt"
	"time"
)

func getPrimaryKeySqlType(adaptor string, column interface{}, size int) string {
	switch adaptor {
	case "sqlite3":
		return "INTEGER PRIMARY KEY"
	case "mysql":
		suffix_str := " NOT NULL AUTO_INCREMENT PRIMARY KEY"
		switch column.(type) {
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			return "int" + suffix_str
		case int64, uint64:
			return "bigint" + suffix_str
		}
	case "postgres":
		switch column.(type) {
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			return "serial"
		case int64, uint64:
			return "bigserial"
		}
	}
	panic("unsupported sql adaptor, please submit an issue in github")
}

func getSqlType(adaptor string, column interface{}, size int) string {
	switch adaptor {
	case "sqlite3":
		switch column.(type) {
		case time.Time:
			return "datetime"
		case bool, sql.NullBool:
			return "bool"
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			return "integer"
		case int64, uint64, sql.NullInt64:
			return "bigint"
		case float32, float64, sql.NullFloat64:
			return "real"
		case string, sql.NullString:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			}
			return "text"
		default:
			panic("invalid sql type")
		}
	case "mysql":
		switch column.(type) {
		case time.Time:
			return "timestamp"
		case bool, sql.NullBool:
			return "boolean"
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			return "int"
		case int64, uint64, sql.NullInt64:
			return "bigint"
		case float32, float64, sql.NullFloat64:
			return "double"
		case []byte:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			}
			return "longblob"
		case string, sql.NullString:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			}
			return "longtext"
		default:
			panic("invalid sql type")
		}

	case "postgres":
		switch column.(type) {
		case time.Time:
			return "timestamp with time zone"
		case bool, sql.NullBool:
			return "boolean"
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			return "integer"
		case int64, uint64, sql.NullInt64:
			return "bigint"
		case float32, float64, sql.NullFloat64:
			return "double precision"
		case []byte:
			return "bytea"
		case string, sql.NullString:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			}
			return "text"
		default:
			panic("invalid sql type")
		}
	default:
		panic("unsupported sql adaptor, please submit an issue in github")
	}
}
