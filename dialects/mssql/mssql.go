package mssql

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jinzhu/gorm"
)

const (
	queryMSSQLHasIndex        = "SELECT count(*) FROM sys.indexes WHERE name=? AND object_id=OBJECT_ID(?)"
	queryMSSQLRemoveIndex     = "DROP INDEX %v ON %v"
	queryMSSQLHasTable        = "SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = ? AND table_catalog = ?"
	queryMSSQLHasColumn       = "SELECT count(*) FROM information_schema.columns WHERE table_catalog = ? AND table_name = ? AND column_name = ?"
	queryMSSQLCurrentDatabase = "SELECT DB_NAME() AS [Current Database]"
)

func setIdentityInsert(scope *gorm.Scope) {
	if scope.Dialect().GetName() == "mssql" {
		for _, field := range scope.PrimaryFields() {
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok && !field.IsBlank {
				scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v ON", scope.TableName()))
				scope.InstanceSet("mssql:identity_insert_on", true)
			}
		}
	}
}

func turnOffIdentityInsert(scope *gorm.Scope) {
	if scope.Dialect().GetName() == "mssql" {
		if _, ok := scope.InstanceGet("mssql:identity_insert_on"); ok {
			scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v OFF", scope.TableName()))
		}
	}
}

func init() {
	gorm.DefaultCallback.Create().After("gorm:begin_transaction").Register("mssql:set_identity_insert", setIdentityInsert)
	gorm.DefaultCallback.Create().Before("gorm:commit_or_rollback_transaction").Register("mssql:turn_off_identity_insert", turnOffIdentityInsert)
	gorm.RegisterDialect("mssql", &mssql{})
}

type mssql struct {
	db gorm.SQLCommon
	gorm.DefaultForeignKeyNamer
}

func (mssql) GetName() string {
	return "mssql"
}

func (s *mssql) SetDB(db gorm.SQLCommon) {
	s.db = db
}

func (mssql) BindVar(i int) string {
	return "$$$" // ?
}

func (mssql) Quote(key string) string {
	return fmt.Sprintf(`"%s"`, key)
}

func (s *mssql) DataTypeOf(field *gorm.StructField) string {
	var dataValue, sqlType, size, additionalType = gorm.ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "bit"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "int IDENTITY(1,1)"
			} else {
				sqlType = "int"
			}
		case reflect.Int64, reflect.Uint64:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "bigint IDENTITY(1,1)"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "float"
		case reflect.String:
			if size > 0 && size < 8000 {
				sqlType = fmt.Sprintf("nvarchar(%d)", size)
			} else {
				sqlType = "nvarchar(max)"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "datetimeoffset"
			}
		default:
			if gorm.IsByteArrayOrSlice(dataValue) {
				if size > 0 && size < 8000 {
					sqlType = fmt.Sprintf("varbinary(%d)", size)
				} else {
					sqlType = "varbinary(max)"
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

func (s mssql) HasForeignKey(tableName string, foreignKeyName string) bool {
	return false
}

func (mssql) LimitAndOffsetSQL(limit, offset interface{}) (sql string) {
	if offset != nil {
		if parsedOffset, err := strconv.ParseInt(fmt.Sprint(offset), 0, 0); err == nil && parsedOffset >= 0 {
			sql += fmt.Sprintf(" OFFSET %d ROWS", parsedOffset)
		}
	}
	if limit != nil {
		if parsedLimit, err := strconv.ParseInt(fmt.Sprint(limit), 0, 0); err == nil && parsedLimit >= 0 {
			if sql == "" {
				// add default zero offset
				sql += " OFFSET 0 ROWS"
			}
			sql += fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", parsedLimit)
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
