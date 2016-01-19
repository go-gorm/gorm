package gorm

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type mssql struct {
	commonDialect
}

func (mssql) DataTypeOf(dataValue reflect.Value, tagSettings map[string]string) string {
	var size int
	if num, ok := tagSettings["SIZE"]; ok {
		size, _ = strconv.Atoi(num)
	}

	switch dataValue.Kind() {
	case reflect.Bool:
		return "bit"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		if _, ok := tagSettings["AUTO_INCREMENT"]; ok {
			return "int IDENTITY(1,1)"
		}
		return "int"
	case reflect.Int64, reflect.Uint64:
		if _, ok := tagSettings["AUTO_INCREMENT"]; ok {
			return "bigint IDENTITY(1,1)"
		}
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("nvarchar(%d)", size)
		}
		return "text"
	case reflect.Struct:
		if _, ok := dataValue.Interface().(time.Time); ok {
			return "datetime2"
		}
	default:
		if _, ok := dataValue.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			}
			return "text"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mssql", dataValue.Type().Name(), dataValue.Kind().String()))
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
