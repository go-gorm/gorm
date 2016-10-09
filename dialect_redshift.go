package gorm

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type redshift struct {
	commonDialect
}

func init() {
	RegisterDialect("redshift", &redshift{})
}

func (redshift) GetName() string {
	return "redshift"
}

func (redshift) BindVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (redshift) DataTypeOf(field *StructField) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "boolean"
		case reflect.Float32:
			sqlType = "float4"
		case reflect.Float64:
			sqlType = "float8"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			if field.IsPrimaryKey {
				delete(field.TagSettings, "AUTO_INCREMENT")
				delete(field.TagSettings, "IDENTITY(1, 1)")
				sqlType = "integer IDENTITY(1, 1)"
			} else if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok {
				delete(field.TagSettings, "AUTO_INCREMENT")
				delete(field.TagSettings, "IDENTITY(1, 1)")
				sqlType = "integer IDENTITY(1, 1)"
			} else if _, ok := field.TagSettings["IDENTITY(1, 1)"]; ok {
				delete(field.TagSettings, "AUTO_INCREMENT")
				delete(field.TagSettings, "IDENTITY(1, 1)")
				sqlType = "integer IDENTITY(1, 1)"
			} else {
				sqlType = "integer"
			}
		case reflect.Int64, reflect.Uintptr, reflect.Uint64:
			if _, ok := field.TagSettings["IDENTITY(1, 1)"]; ok || field.IsPrimaryKey {
				field.TagSettings["IDENTITY(1, 1)"] = "IDENTITY(1, 1)"
				sqlType = "bigint"
			} else {
				sqlType = "bigint"
			}
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
		default:
			sqlType = ""
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for redshift", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s redshift) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM pg_indexes WHERE tablename = $1 AND indexname = $2", tableName, indexName).Scan(&count)
	return count > 0
}

func (s redshift) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	s.db.QueryRow("SELECT count(con.conname) FROM pg_constraint con WHERE $1::regclass::oid = con.conrelid AND con.conname = $2 AND con.contype='f'", tableName, foreignKeyName).Scan(&count)
	return count > 0
}

func (s redshift) HasTable(tableName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = $1 AND table_type = 'BASE TABLE'", tableName).Scan(&count)
	return count > 0
}

func (s redshift) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_name = $1 AND column_name = $2", tableName, columnName).Scan(&count)
	return count > 0
}

func (s redshift) CurrentDatabase() (name string) {
	s.db.QueryRow("SELECT CURRENT_DATABASE()").Scan(&name)
	return
}

func (s redshift) LastInsertIDReturningSuffix(tableName, key string) string {
	return ""
}

func (redshift) SupportLastInsertID() bool {
	return false
}
