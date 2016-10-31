package gorm

import (
	"crypto/sha1"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

type oci8 struct {
	commonDialect
}

func init() {
	RegisterDialect("oci8", new(oci8))
}

func (*oci8) GetName() string {
	return "oci8"
}

func (o *oci8) Quote(key string) string {
	// oracle only support names with a maximum of 30 characters
	key = o.buildSha(key)
	return fmt.Sprintf(`"%s"`, strings.ToUpper(key))
}

func (*oci8) SelectFromDummyTable() string {
	return "FROM DUAL"
}

func (*oci8) BindVar(i int) string {
	return fmt.Sprintf(":%d", i)
}

func (o *oci8) DataTypeOf(field *StructField) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field)

	if len(sqlType) == 0 {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "CHAR(1)"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				field.TagSettings["SEQUENCE"] = "SEQUENCE"
			}
			sqlType = "INTEGER"
		case reflect.Int64, reflect.Uint64:
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok || field.IsPrimaryKey {
				field.TagSettings["SEQUENCE"] = "SEQUENCE"
			}
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
		case reflect.Array, reflect.Slice:
			if isByteArrayOrSlice(dataValue) {
				sqlType = "BLOB"
			}
		}
	}

	if len(sqlType) == 0 {
		panic(fmt.Sprintf("invalid sql type %s (%s) for oci8", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if len(strings.TrimSpace(additionalType)) == 0 {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (o *oci8) HasIndex(tableName string, indexName string) bool {
	var count int
	o.db.QueryRow("SELECT COUNT(*) FROM USER_INDEXES WHERE TABLE_NAME = :1 AND INDEX_NAME = :2", strings.ToUpper(tableName), strings.ToUpper(indexName)).Scan(&count)
	return count > 0
}

func (o *oci8) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	o.db.QueryRow("SELECT COUNT(*) FROM USER_CONSTRAINTS WHERE CONSTRAINT_TYPE = 'R' AND TABLE_NAME = :1 AND CONSTRAINT_NAME = :2", strings.ToUpper(tableName), strings.ToUpper(foreignKeyName)).Scan(&count)
	return count > 0
}

func (o *oci8) HasTable(tableName string) bool {
	var count int
	o.db.QueryRow("SELECT COUNT(*) FROM USER_TABLES WHERE TABLE_NAME = :1", strings.ToUpper(tableName)).Scan(&count)
	return count > 0
}

func (o *oci8) HasColumn(tableName string, columnName string) bool {
	var count int
	o.db.QueryRow("SELECT COUNT(*) FROM USER_TAB_COLUMNS WHERE TABLE_NAME = :1 AND COLUMN_NAME = :2", strings.ToUpper(tableName), strings.ToUpper(columnName)).Scan(&count)
	return count > 0
}

func (*oci8) LimitAndOffsetSQL(limit, offset interface{}) (whereSQL, suffixSQL string) {
	// switch limit := limit.(type) {
	// case int, uint, uint8, int8, uint16, int16, uint32, int32, uint64, int64:
	// 	whereSQL += fmt.Sprintf("ROWNUM <= %d", limit)
	// }
	return
}

func (o *oci8) BuildForeignKeyName(tableName, field, dest string) string {
	keyName := o.commonDialect.BuildForeignKeyName(tableName, field, dest)
	return o.buildSha(keyName)
}

func (o *oci8) SequenceName(tableName, columnName string) string {
	seqName := fmt.Sprintf("%s_%s", tableName, columnName)
	return o.buildSha(seqName)
}

func (o *oci8) NextSequenceSQL(tableName, columnName string) string {
	return fmt.Sprintf("%s.NEXTVAL", o.SequenceName(tableName, columnName))
}

func (*oci8) buildSha(str string) string {
	if utf8.RuneCountInString(str) <= 30 {
		return str
	}

	h := sha1.New()
	h.Write([]byte(str))
	bs := h.Sum(nil)

	result := fmt.Sprintf("%x", bs)
	if len(result) <= 30 {
		return result
	}
	return result[:29]
}
