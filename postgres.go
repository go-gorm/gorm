package gorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"github.com/lib/pq/hstore"
)

type postgres struct {
}

func (s *postgres) BinVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (s *postgres) SupportLastInsertId() bool {
	return false
}

func (s *postgres) HasTop() bool {
	return false
}

func (d *postgres) SqlTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "numeric"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "text"
	case reflect.Struct:
		if value.Type() == timeType {
			return "timestamp with time zone"
		}
	case reflect.Map:
		if value.Type() == hstoreType {
			return "hstore"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			return "bytea"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for postgres", value.Type().Name(), value.Kind().String()))
}

func (s *postgres) PrimaryKeyTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "serial PRIMARY KEY"
	case reflect.Int64, reflect.Uint64:
		return "bigserial PRIMARY KEY"
	default:
		panic("Invalid primary key type")
	}
}

func (s *postgres) ReturningStr(key string) string {
	return fmt.Sprintf("RETURNING \"%v\"", key)
}

func (s *postgres) SelectFromDummyTable() string {
	return ""
}

func (s *postgres) Quote(key string) string {
	return fmt.Sprintf("\"%s\"", key)
}

func (s *postgres) HasTable(scope *Scope, tableName string) bool {
	var count int
	newScope := scope.New(nil)
	newScope.Raw(fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.tables where table_name = %v and table_type = 'BASE TABLE'", newScope.AddToVars(tableName)))
	newScope.DB().QueryRow(newScope.Sql, newScope.SqlVars...).Scan(&count)
	return count > 0
}

func (s *postgres) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	newScope := scope.New(nil)
	newScope.Raw(fmt.Sprintf("SELECT count(*) FROM information_schema.columns WHERE table_name = %v AND column_name = %v",
		newScope.AddToVars(tableName),
		newScope.AddToVars(columnName),
	))
	newScope.DB().QueryRow(newScope.Sql, newScope.SqlVars...).Scan(&count)
	return count > 0
}

func (s *postgres) RemoveIndex(scope *Scope, indexName string) {
	scope.Raw(fmt.Sprintf("DROP INDEX %v", indexName)).Exec()
}

var hstoreType = reflect.TypeOf(Hstore{})

type Hstore map[string]*string

func (h Hstore) Value() (driver.Value, error) {
	hstore := hstore.Hstore{Map: map[string]sql.NullString{}}
	if len(h) == 0 {
		return nil, nil
	}

	for key, value := range h {
		hstore.Map[key] = sql.NullString{String: *value, Valid: true}
	}
	return hstore.Value()
}

func (h *Hstore) Scan(value interface{}) error {
	hstore := hstore.Hstore{}

	if err := hstore.Scan(value); err != nil {
		return err
	}

	if len(hstore.Map) == 0 {
		return nil
	}

	*h = Hstore{}
	for k := range hstore.Map {
		if hstore.Map[k].Valid {
			s := hstore.Map[k].String
			(*h)[k] = &s
		} else {
			(*h)[k] = nil
		}
	}

	return nil
}
