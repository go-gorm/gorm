package dialect

import (
	"fmt"
	"reflect"
)

type postgres struct {
}

func (s *postgres) BinVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (s *postgres) SupportLastInsertId() bool {
	return false
}

func (d *postgres) SqlTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "numeric"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "text"
	case reflect.Struct:
		if value.Type() == timeType {
			return "timestamp with time zone"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			return "bytea"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for postgres", value.Type().Name(), value.Kind().String()))
}

func (s *postgres) PrimaryKeyTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "serial PRIMARY KEY"
	case reflect.Int64, reflect.Uint64:
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
