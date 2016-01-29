package gorm

import (
	"fmt"
	"reflect"
	"time"
)

type cockroach struct {
	commonDialect
}

func (cockroach) BinVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (cockroach) SupportLastInsertId() bool {
	return false
}

func (cockroach) SupportUniquePrimaryKey() bool {
	return false
}

func (cockroach) SupportUpdatePrimaryKey() bool {
	return false
}

func (cockroach) NewUniqueKey(scope *Scope) uint64 {
	rows, err := scope.NewDB().Raw(`SELECT experimental_unique_int()`).Rows()
	if err != nil {
		scope.Err(err)
		return 0
	}
	defer rows.Close()
	var id int64
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			scope.Err(err)
			return 0
		}
	}
	return uint64(id)
}

func (cockroach) SqlTag(value reflect.Value, size int, autoIncrease bool) string {
	switch value.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		if autoIncrease {
			return "INTEGER PRIMARY KEY"
		}
		return "INTEGER"
	case reflect.Int64, reflect.Uint64:
		if autoIncrease {
			return "BIGINT PRIMARY KEY"
		}
		return "BIGINT"
	case reflect.Float32, reflect.Float64:
		return "FLOAT"
	case reflect.String:
		if size > 0 && size < 65532 {
			return "VARCHAR"
		}
		return "TEXT"
	case reflect.Struct:
		if _, ok := value.Interface().(time.Time); ok {
			return "TIMESTAMP"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			return "BYTES"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for cockroach", value.Type().Name(), value.Kind().String()))
}

func (s cockroach) HasTable(scope *Scope, tableName string) bool {
	rows, err := scope.NewDB().Raw("show tables").Rows()
	if err != nil {
		scope.Err(err)
		return false
	}
	defer rows.Close()
	var name string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			scope.Err(err)
			return false
		}
		if name == tableName {
			return true
		}
	}
	return false
}

func (s cockroach) HasColumn(scope *Scope, tableName string, columnName string) bool {
	rows, err := scope.NewDB().Raw(fmt.Sprintf("show columns from %s", tableName)).Rows()
	if err != nil {
		scope.Err(err)
		return false
	}
	defer rows.Close()
	var column string
	var typ, null, defaultVal interface{}
	for rows.Next() {
		if err := rows.Scan(&column, &typ, &null, &defaultVal); err != nil {
			scope.Err(err)
			return false
		}
		if column == columnName {
			return true
		}
	}
	return false
}

func (s cockroach) HasIndex(scope *Scope, tableName string, indexName string) bool {
	rows, err := scope.NewDB().Raw(fmt.Sprintf("show index from %s", tableName)).Rows()
	if err != nil {
		scope.Err(err)
		return false
	}
	defer rows.Close()

	var table, name, column, direction string
	var unique, storing bool
	var seq int
	for rows.Next() {
		if err := rows.Scan(&table, &name, &unique, &seq, &column, &direction, &storing); err != nil {
			scope.Err(err)
			return false
		}
		if name == indexName {
			return true
		}
	}
	return false
}

func (cockroach) RemoveIndex(scope *Scope, indexName string) {
	scope.Err(scope.NewDB().Exec(fmt.Sprintf("DROP INDEX %v@%v", scope.TableName(), indexName)).Error)
}

func (s cockroach) CurrentDatabase(scope *Scope) string {
	var name string
	s.RawScanString(scope, &name, "SHOW DATABASE")
	return name
}
