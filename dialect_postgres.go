package gorm

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"time"

	dPostgres "github.com/jinzhu/gorm/dialects/postgres"
)

type postgres struct {
	commonDialect
}

func init() {
	RegisterDialect("postgres", &postgres{})
	RegisterDialect("cloudsqlpostgres", &postgres{})
}

func (postgres) GetName() string {
	return "postgres"
}

func literal(s string) string {
	p := ""

	if strings.Contains(s, `\`) {
		p = "E"
	}

	s = strings.Replace(s, `'`, `''`, -1)
	s = strings.Replace(s, `\`, `\\`, -1)
	return p + `'` + s + `'`
}

func isNil(value interface{}) (ret bool) {
	ret = false
	defer func() {
		if e := recover(); e != nil {
			// DO NOTHING
		}
	}()

	if value == nil {
		ret = true
		return
	}

	if reflect.ValueOf(value).IsNil() {
		ret = true
		return
	}

	return
}

func (p postgres) StringifyVar(value interface{}) (ret string, ok bool) {
	ok = true
	if isNil(value) {
		ret = "NULL"
		return
	}

	if reflect.TypeOf(value).Kind() == reflect.Ptr {
		ret, ok = p.StringifyVar(reflect.ValueOf(value).Elem().Interface())
		return
	}

	switch value.(type) {
	case string:
		s := value.(string)
		ret = literal(s)
		return
	case time.Time:
		s := value.(time.Time)
		ret = literal(s.Format(time.RFC3339Nano))
		return
	case dPostgres.Hstore:
		s := value.(dPostgres.Hstore)
		if v, err := s.Value(); err == nil {
			ret = literal(string(v.([]byte)))
			return
		}
	case []byte:
		s := value.([]byte)
		ret = "decode(" + literal(hex.EncodeToString(s)) + ", 'hex')"
		return
	}

	if s, ok2 := value.(driver.Valuer); ok2 {
		if v, err := s.Value(); err == nil {
			ret, ok = p.StringifyVar(v)
			return
		}
	}

	if s, ok2 := value.(fmt.Stringer); ok2 {
		ret = literal(s.String())
		return
	}

	ret = fmt.Sprintf("%v", value)
	return
}

func (postgres) BindVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (s *postgres) DataTypeOf(field *StructField) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "boolean"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uintptr:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "serial"
			} else {
				sqlType = "integer"
			}
		case reflect.Int64, reflect.Uint32, reflect.Uint64:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "bigserial"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "numeric"
		case reflect.String:
			if _, ok := field.TagSettings["SIZE"]; !ok {
				size = 0 // if SIZE haven't been set, use `text` as the default type, as there are no performance different
			}

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
			if dataValue.Type().Name() == "Hstore" {
				sqlType = "hstore"
			}
		default:
			if IsByteArrayOrSlice(dataValue) {
				sqlType = "bytea"
				if isUUID(dataValue) {
					sqlType = "uuid"
				}
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

func (s postgres) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM pg_indexes WHERE tablename = $1 AND indexname = $2 AND schemaname = CURRENT_SCHEMA()", tableName, indexName).Scan(&count)
	return count > 0
}

func (s postgres) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	s.db.QueryRow("SELECT count(con.conname) FROM pg_constraint con WHERE $1::regclass::oid = con.conrelid AND con.conname = $2 AND con.contype='f'", tableName, foreignKeyName).Scan(&count)
	return count > 0
}

func (s postgres) HasTable(tableName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = $1 AND table_type = 'BASE TABLE' AND table_schema = CURRENT_SCHEMA()", tableName).Scan(&count)
	return count > 0
}

func (s postgres) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_name = $1 AND column_name = $2 AND table_schema = CURRENT_SCHEMA()", tableName, columnName).Scan(&count)
	return count > 0
}

func (s postgres) CurrentDatabase() (name string) {
	s.db.QueryRow("SELECT CURRENT_DATABASE()").Scan(&name)
	return
}

func (s postgres) LastInsertIDReturningSuffix(tableName, key string) string {
	return fmt.Sprintf("RETURNING %v.%v", tableName, key)
}

func (postgres) SupportLastInsertID() bool {
	return false
}

func isUUID(value reflect.Value) bool {
	if value.Kind() != reflect.Array || value.Type().Len() != 16 {
		return false
	}
	typename := value.Type().Name()
	lower := strings.ToLower(typename)
	return "uuid" == lower || "guid" == lower
}
