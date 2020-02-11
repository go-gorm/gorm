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
)

type oracle struct {
	db gorm.SQLCommon
	gorm.DefaultForeignKeyNamer
}

func init() {
	gorm.RegisterDialect("godror", &oracle{})
}

func (oracle) GetName() string {
	return "goracle"
}

func (oracle) BindVar(i int) string {
	return fmt.Sprintf(":%v", i)
}

func (oracle) Quote(key string) string {
	return fmt.Sprintf("%s", key)
}

func (s oracle) CurrentDatabase() (name string) {
	s.db.QueryRow("SELECT ORA_DATABASE_NAME as \"Current Database\" FROM DUAL").Scan(&name)
	return
}

func (oracle) DefaultValueStr() string {
	return "DEFAULT VALUES"
}

func (s oracle) HasColumn(tableName string, columnName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	fmt.Printf(currentDatabase)
	s.db.QueryRow("SELECT count(*) FROM ALL_TAB_COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?", tableName, columnName).Scan(&count)
	return count > 0
}

func (s oracle) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	fmt.Printf(currentDatabase)
	s.db.QueryRow(`SELECT count(*)
  FROM all_cons_columns a
  JOIN all_constraints c ON a.owner = c.owner
                        AND a.constraint_name = c.constraint_name
                        AND a.constraint_name = ?
  JOIN all_constraints c_pk ON c.r_owner = c_pk.owner
                           AND c.r_constraint_name = c_pk.constraint_name
 WHERE c.constraint_type = 'R'
   AND a.table_name = ?;`, foreignKeyName, tableName).Scan(&count)
	return count > 0
}

func (s oracle) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM ALL_INDEXES WHERE INDEX_NAME = ? AND TABLE_NAME = ?", indexName, tableName).Scan(&count)
	return count > 0
}

func (s oracle) HasTable(tableName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	fmt.Printf(currentDatabase)
	s.db.QueryRow("SELECT DISTINCT OBJECT_NAME FROM USER_OBJECTS WHERE OBJECT_TYPE = 'TABLE' AND OBJECT_NAME = ?", tableName).Scan(&count)
	return count > 0
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
	return "SELECT * FROM DUAL"
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
	var dataValue, sqlType, size, additionalType = gorm.ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8,
			reflect.Uint16, reflect.Uintptr, reflect.Int64, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			sqlType = "NUMBER"
		case reflect.String:
			if _, ok := field.TagSettingsGet("SIZE"); !ok {
				size = 0 // if SIZE haven't been set, use `text` as the default type, as there are no performance different
			}

			if size > 0 && size < 4000 {
				sqlType = fmt.Sprintf("VARCHAR2(%d)", size)
			} else {
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
				sqlType = "VARCHAR2"

				/*if isUUID(dataValue) {
					sqlType = "VARCHAR2"
				}

				if isJSON(dataValue) {
					sqlType = "CLOB"
				}*/
			}
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for oracle", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}
func (s oracle) LimitAndOffsetSQL(limit, offset interface{}) (sql string, err error) {
	fmt.Println("Entering LimitAndOffsetSQL function...")
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
	fmt.Println("LimitAndOffsetSQL: " + sql)
	return
}

// NormalizeIndexAndColumn returns argument's index name and column name without doing anything
func (oracle) NormalizeIndexAndColumn(indexName, columnName string) (string, string) {
	return indexName, columnName
}
