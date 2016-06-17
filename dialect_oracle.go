package gorm

import (
	"crypto/sha1"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type oracle struct {
	commonDialect
}

func init() {
	RegisterDialect("ora", &oracle{})
}

func (oracle) GetName() string {
	return "ora"
}

func (oracle) Quote(key string) string {
	return fmt.Sprintf("\"%s\"", strings.ToUpper(key))
}

func (oracle) SelectFromDummyTable() string {
	return "FROM dual"
}

func (oracle) BindVar(i int) string {
	return fmt.Sprintf(":%d", i)
}

func (oracle) DataTypeOf(field *StructField) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "CHAR(1)"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			sqlType = "INTEGER"
		case reflect.Int64, reflect.Uint64:
			sqlType = "NUMBER"
		case reflect.Float32, reflect.Float64:
			sqlType = "FLOAT"
		case reflect.String:
			if size > 0 && size < 255 {
				sqlType = fmt.Sprintf("VARCHAR(%d)", size)
			} else {
				sqlType = "VARCHAR(255)"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "TIMESTAMP"
			}
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for ora", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s oracle) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM USER_INDEXES WHERE TABLE_NAME = :1 AND INDEX_NAME = :2", strings.ToUpper(tableName), strings.ToUpper(indexName)).Scan(&count)
	return count > 0
}

func (s oracle) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM USER_CONSTRAINTS WHERE CONSTRAINT_TYPE = 'R' AND TABLE_NAME = :1 AND CONSTRAINT_NAME = :2", strings.ToUpper(tableName), strings.ToUpper(foreignKeyName)).Scan(&count)
	return count > 0
}

func (s oracle) HasTable(tableName string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM USER_TABLES WHERE TABLE_NAME = :1", strings.ToUpper(tableName)).Scan(&count)
	return count > 0
}

func (s oracle) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM USER_TAB_COLUMNS WHERE TABLE_NAME = :1 AND COLUMN_NAME = :2", strings.ToUpper(tableName), strings.ToUpper(columnName)).Scan(&count)
	return count > 0
}

func (oracle) LimitAndOffsetSQL(limit, offset int) (whereSQL, suffixSQL string) {
	if limit > 0 {
		whereSQL += fmt.Sprintf("ROWNUM <= %d", limit)
	}
	return
}

func (s oracle) BuildForeignKeyName(tableName, field, dest string) string {
	keyName := s.commonDialect.BuildForeignKeyName(tableName, field, dest)
	if utf8.RuneCountInString(keyName) <= 30 {
		return keyName
	}
	h := sha1.New()
	h.Write([]byte(keyName))
	bs := h.Sum(nil)

	// sha1 is 40 digits, keep first 24 characters of destination
	destRunes := []rune(regexp.MustCompile("(_*[^a-zA-Z]+_*|_+)").ReplaceAllString(dest, "_"))
	result := fmt.Sprintf("%s%x", string(destRunes), bs)
	if len(result) <= 30 {
		return result
	}
	return result[:29]
}
