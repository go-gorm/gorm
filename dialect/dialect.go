package dialect

import (
	"reflect"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

type Dialect interface {
	BinVar(i int) string
	SupportLastInsertId() bool
	SqlTag(value reflect.Value, size int) string
	PrimaryKeyTag(value reflect.Value, size int) string
	ReturningStr(key string) string
	Quote(key string) string
}

func New(driver string) Dialect {
	var d Dialect
	switch driver {
	case "postgres":
		d = &postgres{}
	case "mysql":
		d = &mysql{}
	case "sqlite3":
		d = &sqlite3{}
	}
	return d
}
