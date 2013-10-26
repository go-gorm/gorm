package gorm

import (
	"fmt"
	"time"
)

func getSqlType(adaptor string, column interface{}, size int) string {
	switch adaptor {
	case "mysql":
		switch column.(type) {
		case time.Time:
			return "timestamp"
		case bool:
			return "boolean"
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			return "int"
		case int64, uint64:
			return "bigint"
		case float32, float64:
			return "double"
		case []byte:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			}
			return "longblob"
		case string:
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
		case bool:
			return "boolean"
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			return "integer"
		case int64, uint64:
			return "bigint"
		case float32, float64:
			return "double precision"
		case []byte:
			return "bytea"
		case string:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			}
			return "text"
		default:
			panic("invalid sql type")
		}
	}

	panic("unsupported sql adaptor, please submit an issue in github")
}
