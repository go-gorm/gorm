package gorm

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type mssql struct {
	commonDialect
}

func (mssql) DataTypeOf(field *StructField) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "bit"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				sqlType = "int IDENTITY(1,1)"
			} else {
				sqlType = "int"
			}
		case reflect.Int64, reflect.Uint64:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				sqlType = "bigint IDENTITY(1,1)"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "float"
		case reflect.String:
			if size > 0 && size < 65532 {
				sqlType = fmt.Sprintf("nvarchar(%d)", size)
			} else {
				sqlType = "text"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "datetime2"
			}
		default:
			if _, ok := dataValue.Interface().([]byte); ok {
				if size > 0 && size < 65532 {
					sqlType = fmt.Sprintf("varchar(%d)", size)
				} else {
					sqlType = "text"
				}
			}
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for mssql", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s mssql) HasIndex(scope *Scope, tableName string, indexName string) bool {
	var count int
	s.RawScanInt(scope, &count, "SELECT count(*) FROM sys.indexes WHERE name=? AND object_id=OBJECT_ID(?)", indexName, tableName)
	return count > 0
}

func (s mssql) HasTable(scope *Scope, tableName string) bool {
	var (
		count        int
		databaseName = s.currentDatabase(scope)
	)
	s.RawScanInt(scope, &count, "SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = ? AND table_catalog = ?", tableName, databaseName)
	return count > 0
}

func (s mssql) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var (
		count        int
		databaseName = s.currentDatabase(scope)
	)
	s.RawScanInt(scope, &count, "SELECT count(*) FROM information_schema.columns WHERE table_catalog = ? AND table_name = ? AND column_name = ?", databaseName, tableName, columnName)
	return count > 0
}

func (s mssql) currentDatabase(scope *Scope) (name string) {
	s.RawScanString(scope, &name, "SELECT DB_NAME() AS [Current Database]")
	return
}

func (mssql) LimitAndOffsetSQL(limit, offset int) (sql string) {
	if limit < 0 && offset < 0 {
		return
	}

	if offset < 0 {
		offset = 0
	}

	sql += fmt.Sprintf(" OFFSET %d ROWS", offset)

	if limit >= 0 {
		sql += fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", limit)
	}
	return
}
