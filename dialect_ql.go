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

func (ql) Quote(key string) string {
  return fmt.Sprintf(`%s`, key)
}

func (ql) PrimaryKeys(keys []string) string {
  return ""
}

// Get Data Type for Sqlite Dialect
func (ql) DataTypeOf(field *StructField) string {
  var dataValue, sqlType, _, additionalType = ParseFieldStructForDialect(field)

  if sqlType == "" {
    switch dataValue.Kind() {
    case reflect.Bool:
      sqlType = "bool"
    case reflect.Int:
      sqlType = "int"
    case reflect.Int8:
      sqlType = "int8"
    case reflect.Int16:
      sqlType = "int16"
    case reflect.Int32:
      sqlType = "int32"
    case reflect.Int64:
      sqlType = "int64"
    case reflect.Uint:
      sqlType = "uint"
    case reflect.Uint8:
      sqlType = "uint8"
    case reflect.Uint16:
      sqlType = "uint16"
    case reflect.Uint32:
      sqlType = "uint32"
    case reflect.Uint64:
      sqlType = "uint64"
    case reflect.Uintptr:
      sqlType = "uint"
    case reflect.Float32:
      sqlType = "float32"
    case reflect.Float64:
      sqlType = "float64"
    case reflect.String:
      sqlType = "string"
    case reflect.Struct:
      if _, ok := dataValue.Interface().(time.Time); ok {
        sqlType = "time"
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

func (s ql) RemoveIndex(tableName string, indexName string) error {
  _, err := s.db.Exec(fmt.Sprintf("BEGIN TRANSACTION; DROP INDEX %v; COMMIT;", indexName))
  return err
}
