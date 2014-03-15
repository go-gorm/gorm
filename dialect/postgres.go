package dialect

import (
	"fmt"
	"time"
)

type postgres struct {
}

func (s *postgres) BinVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (s *postgres) SupportLastInsertId() bool {
	return false
}

func (d *postgres) SqlTag(column interface{}, size int) string {
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
		return "numeric"
	case []byte:
		return "bytea"
	case string:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		} else {
			return "text"
		}
	default:
		panic("Invalid sql type for postgres")
	}
}

func (s *postgres) PrimaryKeyTag(column interface{}, size int) string {
	switch column.(type) {
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return "serial PRIMARY KEY"
	case int64, uint64:
		return "bigserial PRIMARY KEY"
	default:
		panic("Invalid primary key type")
	}
}

func (s *postgres) ReturningStr(key string) (str string) {
	return fmt.Sprintf("RETURNING \"%v\"", key)
}

func (s *postgres) Quote(key string) (str string) {
	return fmt.Sprintf("\"%s\"", key)
}
