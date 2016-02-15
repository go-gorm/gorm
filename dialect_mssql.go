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

func init() {
	RegisterDialect("mssql", &mssql{})
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

func (s mssql) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM sys.indexes WHERE name=? AND object_id=OBJECT_ID(?)", indexName, tableName).Scan(&count)
	return count > 0
}

func (s mssql) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP INDEX %v ON %v", indexName, s.Quote(tableName)))
	return err
}

func (s mssql) HasTable(tableName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = ? AND table_catalog = ?", tableName, s.currentDatabase()).Scan(&count)
	return count > 0
}

func (s mssql) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM information_schema.columns WHERE table_catalog = ? AND table_name = ? AND column_name = ?", s.currentDatabase(), tableName, columnName).Scan(&count)
	return count > 0
}

func (s mssql) currentDatabase() (name string) {
	s.db.QueryRow("SELECT DB_NAME() AS [Current Database]").Scan(&name)
	return
}

func (mssql) LimitAndOffsetSQL(limit, offset int) (sql string) {
	if limit > 0 || offset > 0 {
		if offset < 0 {
			offset = 0
		}

		sql += fmt.Sprintf(" OFFSET %d ROWS", offset)

		if limit >= 0 {
			sql += fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", limit)
		}
	}
	return
}
