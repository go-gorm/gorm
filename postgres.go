package gorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"

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

func (s *postgres) SqlTag(value reflect.Value, size int, autoIncrease bool) string {
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		if autoIncrease {
			return "serial"
		}
		return "integer"
	case reflect.Int64, reflect.Uint64:
		if autoIncrease {
			return "bigserial"
		}
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "numeric"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "text"
	case reflect.Struct:
		if _, ok := value.Interface().(time.Time); ok {
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

func (s *postgres) ReturningStr(tableName, key string) string {
	return fmt.Sprintf("RETURNING %v.%v", s.Quote(tableName), key)
}

func (s *postgres) SelectFromDummyTable() string {
	return ""
}

func (s *postgres) Quote(key string) string {
	return fmt.Sprintf("\"%s\"", key)
}

func (s *postgres) HasTable(scope *Scope, tableName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_schema = current_schema AND table_name = ? AND (table_type = 'BASE TABLE' OR table_type = 'TABLE')", tableName).Row().Scan(&count)
	return count > 0
}

func (s *postgres) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = current_schema AND table_name = ? AND column_name = ?", tableName, columnName).Row().Scan(&count)
	return count > 0
}

func (s *postgres) RemoveIndex(scope *Scope, indexName string) {
	scope.NewDB().Exec(fmt.Sprintf("DROP INDEX %v", indexName))
}

func (s *postgres) HasIndex(scope *Scope, tableName string, indexName string) bool {
	// Determine whehter the pg_indexes talbe is
	detect := []int{} //make([]int, 2) // If detect[0] == 1 then we have postgres, else if detect[1] == 1 then we have detectationDB.
	scope.NewDB().Raw(`SELECT count(*) "detect" FROM INFORMATION_SCHEMA.tables WHERE table_name = 'pg_indexes' UNION SELECT count(*) "detect" FROM INFORMATION_SCHEMA.tables WHERE table_name = 'indexes'`).Pluck("detect", &detect)
	var count int
	var indexQuery string
	if detect[0] == 1 { // Then we have postgres.
		indexQuery = "SELECT count(*) FROM pg_indexes WHERE tablename = ? AND indexname = ?"
	} else if detect[1] == 1 { // Then we have FoundationDB.
		indexQuery = "SELECT count(*) FROM INFORMATION_SCHEMA.indexes WHERE table_schema = current_schema AND table_name = ? AND index_name = ?"
	} else {
		panic("unable to query for indexes, unrecognized database schema layout")
	}
	scope.NewDB().Raw(indexQuery, tableName, indexName).Row().Scan(&count)
	return count > 0
}

var hstoreType = reflect.TypeOf(Hstore{})

type Hstore map[string]*string

func (h Hstore) Value() (driver.Value, error) {
	hstore := hstore.Hstore{Map: map[string]sql.NullString{}}
	if len(h) == 0 {
		return nil, nil
	}

	for key, value := range h {
		var s sql.NullString
		if value != nil {
			s.String = *value
			s.Valid = true
		}
		hstore.Map[key] = s
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
