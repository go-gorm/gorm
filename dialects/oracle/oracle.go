//Oracle dialect for GORM
package gorm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "github.com/godror/godror"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-oci8"
	_ "gopkg.in/rana/ora.v4"
)

//const dialectName = "godror"
// const dialectName = "ora"
const dialectName = "oci8"

type oracle struct {
	db gorm.SQLCommon
	gorm.DefaultForeignKeyNamer
}

func init() {
	gorm.RegisterDialect(dialectName, &oracle{})

}

func (s *oracle) fieldCanAutoIncrement(field *gorm.StructField) bool {
	if value, ok := field.TagSettingsGet("AUTO_INCREMENT"); ok {
		return strings.ToLower(value) != "false"
	}
	return field.IsPrimaryKey
}

func (oracle) GetName() string {
	return dialectName
}

func (oracle) BindVar(i int) string {
	return fmt.Sprintf(":%v", i)
}

func (oracle) Quote(key string) string {
	return key
}

func (s oracle) CurrentDatabase() string {
	var name string
	s.db.QueryRow("SELECT ORA_DATABASE_NAME as \"Current Database\" FROM DUAL").Scan(&name)
	return name
}

func (oracle) DefaultValueStr() string {
	return "DEFAULT VALUES"
}

func (s oracle) HasColumn(tableName string, columnName string) bool {
	var count int
	_, tableName = currentDatabaseAndTable(&s, tableName)
	tableName = strings.ToUpper(tableName)
	columnName = strings.ToUpper(columnName)
	if err := s.db.QueryRow("SELECT count(*) FROM ALL_TAB_COLUMNS WHERE TABLE_NAME = :1 AND COLUMN_NAME = :2", tableName, columnName).Scan(&count); err == nil {
		return count > 0
	}
	return false
}

func (s oracle) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	tableName = strings.ToUpper(tableName)
	foreignKeyName = strings.ToUpper(foreignKeyName)

	if err := s.db.QueryRow(`SELECT count(*) FROM USER_CONSTRAINTS WHERE CONSTRAINT_NAME = :1 AND constraint_type = 'R' AND table_name = :2`, foreignKeyName, tableName).Scan(&count); err == nil {
	   return count > 0
   } 
   return false
}

func (s oracle) HasIndex(tableName string, indexName string) bool {
	var count int
	tableName = strings.ToUpper(tableName)
	indexName = strings.ToUpper(indexName)
	if err := s.db.QueryRow("SELECT count(*) FROM ALL_INDEXES WHERE INDEX_NAME = :1 AND TABLE_NAME = :2", indexName, tableName).Scan(&count); err == nil {
		return count > 0
	}
	return false
}

func (s oracle) HasTable(tableName string) bool {
	var count int
	_, tableName = currentDatabaseAndTable(&s, tableName)
	tableName = strings.ToUpper(tableName)
	if err := s.db.QueryRow("select count(*) from user_tables where table_name = :1", tableName).Scan(&count); err == nil {
		return count > 0
	} 
	return false
}

func (oracle) LastInsertIDReturningSuffix(tableName, columnName string) string {
	return ""
}

func (oracle) LastInsertIDOutputInterstitial(tableName, columnName string, columns []string) string {
	return ""
}

func (s oracle) ModifyColumn(tableName string, columnName string, typ string) error {
	_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %v MODIFY %v %v", tableName, columnName, typ))
	return err
}

func (s oracle) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP INDEX %v", indexName))
	return err
}

func (oracle) SelectFromDummyTable() string {
	return "FROM DUAL"
}

func (s *oracle) SetDB(db gorm.SQLCommon) {
	s.db = db
}

func currentDatabaseAndTable(dialect gorm.Dialect, tableName string) (string, string) {
	if strings.Contains(tableName, ".") {
		splitStrings := strings.SplitN(tableName, ".", 2)
		return splitStrings[0], splitStrings[1]
	}
	return dialect.CurrentDatabase(), tableName
}

func (s *oracle) DataTypeOf(field *gorm.StructField) string {
	if _, found := field.TagSettingsGet("RESTRICT"); found {
		field.TagSettingsDelete("RESTRICT")
	}
	var dataValue, sqlType, size, additionalType = gorm.ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8,
			reflect.Uint16, reflect.Uintptr, reflect.Int64, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			if s.fieldCanAutoIncrement(field) {
				sqlType = "NUMBER GENERATED BY DEFAULT AS IDENTITY"
			} else {
				sqlType = "NUMBER"
			}
		case reflect.String:
			if _, ok := field.TagSettingsGet("SIZE"); !ok {
				size = 0 // if SIZE haven't been set, use `text` as the default type, as there are no performance different
			}
			switch {
			case size > 0 && size < 4000:
				sqlType = fmt.Sprintf("VARCHAR2(%d)", size)
			case size == 0:
				sqlType = "VARCHAR2 (4000)"
			default:
				sqlType = "CLOB"
			}

		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "TIMESTAMP WITH TIME ZONE"
			}
		case reflect.Map:
			if dataValue.Type().Name() == "Hstore" {
				sqlType = "hstore"
			}
		default:
			if gorm.IsByteArrayOrSlice(dataValue) {
				sqlType = "VARCHAR2 (4000)"
			}
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for oracle", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	if strings.EqualFold(sqlType, "json") {
		sqlType = "VARCHAR2 (4000)"
	}

	// For oracle, we have to redo the order of the Default type from tag setting
	notNull, _ := field.TagSettingsGet("NOT NULL")
	unique, _ := field.TagSettingsGet("UNIQUE")
	additionalType = notNull + " " + unique
	if value, ok := field.TagSettingsGet("DEFAULT"); ok {
		additionalType = fmt.Sprintf("%s %s %s", "DEFAULT", value, additionalType)
		// additionalType = additionalType + " DEFAULT " + value
	}

	if value, ok := field.TagSettingsGet("COMMENT"); ok {
		additionalType = additionalType + " COMMENT " + value
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}
func (s oracle) LimitAndOffsetSQL(limit, offset interface{}) (sql string, err error) {
	if limit != nil {
		if parsedLimit, err := strconv.ParseInt(fmt.Sprint(limit), 0, 0); err == nil && parsedLimit >= 0 {
			sql += fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", parsedLimit)

			if offset != nil {
				if parsedOffset, err := strconv.ParseInt(fmt.Sprint(offset), 0, 0); err == nil && parsedOffset >= 0 {
					sql += fmt.Sprintf(" OFFSET %d ROWS ", parsedOffset)
				}
			}
		}
	}
	return
}

// NormalizeIndexAndColumn returns argument's index name and column name without doing anything
func (oracle) NormalizeIndexAndColumn(indexName, columnName string) (string, string) {
	return indexName, columnName
}