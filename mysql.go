package gorm

import (
	"fmt"
	"reflect"
	"time"
)

type mysql struct {
	commonDialect
}

func (mysql) SqlTag(value reflect.Value, size int, autoIncrease bool) string {
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		if autoIncrease {
			return "int AUTO_INCREMENT"
		}
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		if autoIncrease {
			return "int unsigned AUTO_INCREMENT"
		}
		return "int unsigned"
	case reflect.Int64:
		if autoIncrease {
			return "bigint AUTO_INCREMENT"
		}
		return "bigint"
	case reflect.Uint64:
		if autoIncrease {
			return "bigint unsigned AUTO_INCREMENT"
		}
		return "bigint unsigned"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "longtext"
	case reflect.Struct:
		if _, ok := value.Interface().(time.Time); ok {
			return "timestamp NULL"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			}
			return "longblob"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mysql", value.Type().Name(), value.Kind().String()))
}

func (mysql) Quote(key string) string {
	return fmt.Sprintf("`%s`", key)
}

func (mysql) SelectFromDummyTable() string {
	return "FROM DUAL"
}
