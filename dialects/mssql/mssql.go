package mssql

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jinzhu/gorm"
)

func setIdentityInsert(scope *gorm.Scope) {
	if scope.Dialect().GetName() == "mssql" {
		scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v ON", scope.TableName()))
	}
}

func init() {
	gorm.DefaultCallback.Create().After("gorm:begin_transaction").Register("mssql:set_identity_insert", setIdentityInsert)
	gorm.RegisterDialect("mssql", &mssql{})
}

type mssql struct {
	db *sql.DB
}

func (mssql) GetName() string {
	return "mssql"
}

func (s *mssql) SetDB(db *sql.DB) {
	s.db = db
}

func (mssql) BindVar(i int) string {
	return "$$" // ?
}

func (mssql) Quote(key string) string {
	return fmt.Sprintf(`"%s"`, key)
}

func (mssql) DataTypeOf(field *gorm.StructField) string {
	var dataValue, sqlType, size, additionalType = gorm.ParseFieldStructForDialect(field)

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

func (s mssql) HasForeignKey(tableName string, foreignKeyName string) bool {
	return false
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

func (mssql) SelectFromDummyTable() string {
	return ""
}

func (mssql) LastInsertIDReturningSuffix(tableName, columnName string) string {
	return ""
}
