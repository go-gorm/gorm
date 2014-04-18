package dialect

import (
	"fmt"
	"reflect"
)

type mysql struct{}

func (s *mysql) BinVar(i int) string {
	return "$$" // ?
}

func (s *mysql) SupportLastInsertId() bool {
	return true
}

func (d *mysql) SqlTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		} else {
			return "longtext"
		}
	case reflect.Struct:
		if value.Type() == timeType {
			return "datetime"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			} else {
				return "longblob"
			}
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mysql", value.Type().Name(), value.Kind().String()))
}

func (s *mysql) PrimaryKeyTag(value reflect.Value, size int) string {
	suffix_str := " NOT NULL AUTO_INCREMENT PRIMARY KEY"
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int" + suffix_str
	case reflect.Int64, reflect.Uint64:
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
