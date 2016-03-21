package gorm

import (
  "fmt"
  "reflect"
  "strings"
  "time"
)

type ql struct {
  commonDialect
}

func init() {
  RegisterDialect("ql", &ql{})
  RegisterDialect("ql-mem", &ql{})
}

func (ql) GetName() string {
  return "ql"
}

// Get Data Type for Sqlite Dialect
func (ql) DataTypeOf(field *StructField) string {
  var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field)

  if sqlType == "" {
    switch dataValue.Kind() {
    case reflect.Bool:
      sqlType = "bool"
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
      if field.IsPrimaryKey {
        sqlType = "integer primary key autoincrement"
      } else {
        sqlType = "integer"
      }
    case reflect.Int64, reflect.Uint64:
      if field.IsPrimaryKey {
        sqlType = "integer primary key autoincrement"
      } else {
        sqlType = "bigint"
      }
    case reflect.Float32, reflect.Float64:
      sqlType = "real"
    case reflect.String:
      if size > 0 && size < 65532 {
        sqlType = fmt.Sprintf("varchar(%d)", size)
      } else {
        sqlType = "text"
      }
    case reflect.Struct:
      if _, ok := dataValue.Interface().(time.Time); ok {
        sqlType = "datetime"
      }
    default:
      if _, ok := dataValue.Interface().([]byte); ok {
        sqlType = "blob"
      }
    }
  }

  if sqlType == "" {
    panic(fmt.Sprintf("invalid sql type %s (%s) for ql", dataValue.Type().Name(), dataValue.Kind().String()))
  }

  if strings.TrimSpace(additionalType) == "" {
    return sqlType
  }

  return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s ql) HasTable(tableName string) bool {
  var count int
  s.db.QueryRow("SELECT COUNT(*) FROM __Table WHERE Name = ?", tableName).Scan(&count)
  return count > 0
}

func (s ql) HasColumn(tableName string, columnName string) bool {
  var count int
  s.db.QueryRow("SELECT COUNT(*) FROM __Column WHERE TableName = ? AND Name = ?", tableName, columnName).Scan(&count)
  return count > 0
}

func (s ql) HasIndex(tableName string, indexName string) bool {
  var count int
  s.db.QueryRow("SELECT COUNT(*) FROM __Index WHERE TableName = ? AND Name = ?", tableName, indexName).Scan(&count)
  return count > 0
}
