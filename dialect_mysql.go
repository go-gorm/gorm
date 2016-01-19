package gorm

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type mysql struct {
	commonDialect
}

func (mysql) Quote(key string) string {
	return fmt.Sprintf("`%s`", key)
}

func (mysql) DataTypeOf(dataValue reflect.Value, tagSettings map[string]string) string {
	var size int
	if num, ok := tagSettings["SIZE"]; ok {
		size, _ = strconv.Atoi(num)
	}

	switch dataValue.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		if _, ok := tagSettings["AUTO_INCREMENT"]; ok {
			return "int AUTO_INCREMENT"
		}
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		if _, ok := tagSettings["AUTO_INCREMENT"]; ok {
			return "int unsigned AUTO_INCREMENT"
		}
		return "int unsigned"
	case reflect.Int64:
		if _, ok := tagSettings["AUTO_INCREMENT"]; ok {
			return "bigint AUTO_INCREMENT"
		}
		return "bigint"
	case reflect.Uint64:
		if _, ok := tagSettings["AUTO_INCREMENT"]; ok {
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
		if _, ok := dataValue.Interface().(time.Time); ok {
			return "timestamp NULL"
		}
	default:
		if _, ok := dataValue.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			}
			return "longblob"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mysql", dataValue.Type().Name(), dataValue.Kind().String()))
}

func (s mysql) currentDatabase(scope *Scope) (name string) {
	s.RawScanString(scope, &name, "SELECT DATABASE()")
	return
}

func (mysql) SelectFromDummyTable() string {
	return "FROM DUAL"
}
