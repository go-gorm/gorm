package gorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/lib/pq/hstore"
)

type postgres struct {
	commonDialect
}

func (postgres) BindVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (postgres) DataTypeOf(field *StructField) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "boolean"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				sqlType = "serial"
			} else {
				sqlType = "integer"
			}
		case reflect.Int64, reflect.Uint64:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				sqlType = "bigserial"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "numeric"
		case reflect.String:
			if size > 0 && size < 65532 {
				sqlType = fmt.Sprintf("varchar(%d)", size)
			} else {
				sqlType = "text"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "timestamp with time zone"
			}
		case reflect.Map:
			if dataValue.Type() == hstoreType {
				sqlType = "hstore"
			}
		default:
			if isByteArrayOrSlice(dataValue) {
				sqlType = "bytea"
			} else if isUUID(dataValue) {
				sqlType = "uuid"
			}
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for postgres", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s postgres) HasIndex(scope *Scope, tableName string, indexName string) bool {
	var count int
	s.RawScanInt(scope, &count, "SELECT count(*) FROM pg_indexes WHERE tablename = ? AND indexname = ?", tableName, indexName)
	return count > 0
}

func (postgres) RemoveIndex(scope *Scope, indexName string) {
	scope.Err(scope.NewDB().Exec(fmt.Sprintf("DROP INDEX %v", indexName)).Error)
}

func (s postgres) HasTable(scope *Scope, tableName string) bool {
	var count int
	s.RawScanInt(scope, &count, "SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = ? AND table_type = 'BASE TABLE'", tableName)
	return count > 0
}

func (s postgres) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	s.RawScanInt(scope, &count, "SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_name = ? AND column_name = ?", tableName, columnName)
	return count > 0
}

func (s postgres) currentDatabase(scope *Scope) (name string) {
	s.RawScanString(scope, &name, "SELECT CURRENT_DATABASE()")
	return
}

func (s postgres) LastInsertIdReturningSuffix(tableName, key string) string {
	return fmt.Sprintf("RETURNING %v.%v", tableName, key)
}

func (postgres) SupportLastInsertId() bool {
	return false
}

var hstoreType = reflect.TypeOf(Hstore{})

type Hstore map[string]*string

func (h Hstore) Value() (driver.Value, error) {
	hstore := hstore.Hstore{Map: map[string]sql.NullString{}}
	if len(h) == 0 {
		return nil, nil
	}

	for key, value := range h {
		var s sql.NullString
		if value != nil {
			s.String = *value
			s.Valid = true
		}
		hstore.Map[key] = s
	}
	return hstore.Value()
}

func (h *Hstore) Scan(value interface{}) error {
	hstore := hstore.Hstore{}

	if err := hstore.Scan(value); err != nil {
		return err
	}

	if len(hstore.Map) == 0 {
		return nil
	}

	*h = Hstore{}
	for k := range hstore.Map {
		if hstore.Map[k].Valid {
			s := hstore.Map[k].String
			(*h)[k] = &s
		} else {
			(*h)[k] = nil
		}
	}

	return nil
}

func isByteArrayOrSlice(value reflect.Value) bool {
	return (value.Kind() == reflect.Array || value.Kind() == reflect.Slice) && value.Type().Elem() == reflect.TypeOf(uint8(0))
}

func isUUID(value reflect.Value) bool {
	if value.Kind() != reflect.Array || value.Type().Len() != 16 {
		return false
	}
	typename := value.Type().Name()
	lower := strings.ToLower(typename)
	return "uuid" == lower || "guid" == lower
}
