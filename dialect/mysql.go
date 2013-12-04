package dialect

import (
	"database/sql"
	"fmt"
	"time"
)

type mysql struct{}

func (s *mysql) BinVar(i int) string {
	return "$$" // ?
}

func (s *mysql) SupportLastInsertId() bool {
	return true
}

func (d *mysql) SqlTag(column interface{}, size int) string {
	switch column.(type) {
	case time.Time:
		return "datetime"
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
		} else {
			return "longblob"
		}
	case string, sql.NullString:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		} else {
			return "longtext"
		}
	default:
		panic("Invalid sql type for mysql")
	}
}

func (s *mysql) PrimaryKeyTag(column interface{}, size int) string {
	suffix_str := " NOT NULL AUTO_INCREMENT PRIMARY KEY"
	switch column.(type) {
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return "int" + suffix_str
	case int64, uint64:
		return "bigint" + suffix_str
	default:
		panic("Invalid primary key type")
	}
}

func (s *mysql) ReturningStr(key string) (str string) {
	return
}

func (s *mysql) Quote(key string) (str string) {
	return fmt.Sprintf("`%s`", key)
}
