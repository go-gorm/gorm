package gorm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func setIdentityInsert(scope *Scope) {
	if scope.Dialect().GetName() == "mssql" {
		for _, field := range scope.PrimaryFields() {
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok && !field.IsBlank {
				scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v ON", scope.TableName()))
				scope.InstanceSet("mssql:identity_insert_on", true)
			}
		}
	}
}

func turnOffIdentityInsert(scope *Scope) {
	if scope.Dialect().GetName() == "mssql" {
		if _, ok := scope.InstanceGet("mssql:identity_insert_on"); ok {
			scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v OFF", scope.TableName()))
		}
	}
}

func init() {
	DefaultCallback.Create().Before("gorm:create").Register("mssql:set_identity_insert", setIdentityInsert)
	DefaultCallback.Create().After("gorm:create").Register("mssql:turn_off_identity_insert", turnOffIdentityInsert)
	RegisterDialect("mssql", &mssql{})
}

type mssql struct {
	db SQLCommon
	DefaultForeignKeyNamer
}

func (mssql) GetName() string {
	return "mssql"
}

func (s *mssql) SetDB(db SQLCommon) {
	s.db = db
}

func (mssql) BindVar(i int) string {
	return "$$$" // ?
}

func (mssql) Quote(key string) string {
	return fmt.Sprintf(`[%s]`, key)
}

func (s *mssql) DataTypeOf(field *StructField) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "bit"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if s.fieldCanAutoIncrement(field) {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "int IDENTITY(1,1)"
			} else {
				sqlType = "int"
			}
		case reflect.Int64, reflect.Uint64:
			if s.fieldCanAutoIncrement(field) {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "bigint IDENTITY(1,1)"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "float"
		case reflect.String:
			if size > 0 && size < 4000 {
				sqlType = fmt.Sprintf("nvarchar(%d)", size)
			} else {
				sqlType = "nvarchar(max)"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "datetimeoffset"
			}
		default:
			if IsByteArrayOrSlice(dataValue) {
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

func (s mssql) fieldCanAutoIncrement(field *StructField) bool {
	if value, ok := field.TagSettings["AUTO_INCREMENT"]; ok {
		return value != "FALSE"
	}
	return field.IsPrimaryKey
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
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = ? AND table_catalog = ?", tableName, currentDatabase).Scan(&count)
	return count > 0
}

func (s mssql) HasColumn(tableName string, columnName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow("SELECT count(*) FROM information_schema.columns WHERE table_catalog = ? AND table_name = ? AND column_name = ?", currentDatabase, tableName, columnName).Scan(&count)
	return count > 0
}

func (s mssql) ModifyColumn(tableName string, columnName string, typ string) error {
	_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %v ALTER COLUMN %v %v", tableName, columnName, typ))
	return err
}

func (s mssql) CurrentDatabase() (name string) {
	s.db.QueryRow("SELECT DB_NAME() AS [Current Database]").Scan(&name)
	return
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

func (mssql) DefaultValueStr() string {
	return "DEFAULT VALUES"
}

func (mssql) FormatDate(e *expr, format string) *expr {
	mapping := map[rune]string{
		'y': "yyyy",
		'm': "MM",
		'd': "dd",
		'h': "HH",
		'M': "mm",
		's': "ss",
	}
	parsedFormat := parseDateFormat(format, mapping)

	e.expr = "(format(" + e.expr + ", '" + parsedFormat + "'))"
	return e
}
