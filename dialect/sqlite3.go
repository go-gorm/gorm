package dialect

import (
	"database/sql"
	"fmt"
	"time"
)

type sqlite3 struct{}

func (s *sqlite3) BinVar(i int) string {
	return "$$" // ?
}

func (s *sqlite3) SupportLastInsertId() bool {
	return true
}

func (s *sqlite3) SqlTag(column interface{}, size int) string {
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
	case []byte:
		return "blob"
	case string, sql.NullString:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		} else {
			return "text"
		}
	default:
		panic("Invalid sql type for sqlite3")
	}
}

func (s *sqlite3) PrimaryKeyTag(column interface{}, size int) string {
	return "INTEGER PRIMARY KEY"
}

func (s *sqlite3) ReturningStr(key string) (str string) {
	return
}

func (s *sqlite3) Quote(key string) (str string) {
	return fmt.Sprintf("\"%s\"", key)
}
