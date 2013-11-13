package gorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func parseTag(str string) (typ string, addational_typ string, size int) {
	if str == "-" {
		typ = str
	} else if str != "" {
		tags := strings.Split(str, ";")
		m := make(map[string]string)
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.Trim(strings.ToUpper(v[0]), " ")
			if len(v) == 2 {
				m[k] = v[1]
			} else {
				m[k] = k
			}
		}

		if len(m["SIZE"]) > 0 {
			size, _ = strconv.Atoi(m["SIZE"])
		}

		if len(m["TYPE"]) > 0 {
			typ = m["TYPE"]
		}

		addational_typ = m["NOT NULL"] + " " + m["UNIQUE"]
	}
	return
}

func formatColumnValue(column interface{}) interface{} {
	if v, ok := column.(reflect.Value); ok {
		column = v.Interface()
	}

	if valuer, ok := interface{}(column).(driver.Valuer); ok {
		column = reflect.New(reflect.ValueOf(valuer).Field(0).Type()).Elem().Interface()
	}
	return column
}

func getPrimaryKeySqlType(adaptor string, column interface{}, tag string) string {
	column = formatColumnValue(column)
	typ, addational_typ, _ := parseTag(tag)

	if len(typ) != 0 {
		return typ + addational_typ
	}

	switch adaptor {
	case "sqlite3":
		return "INTEGER PRIMARY KEY"
	case "mysql":
		suffix_str := " NOT NULL AUTO_INCREMENT PRIMARY KEY"
		switch column.(type) {
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			typ = "int" + suffix_str
		case int64, uint64:
			typ = "bigint" + suffix_str
		}
	case "postgres":
		switch column.(type) {
		case int, int8, int16, int32, uint, uint8, uint16, uint32:
			typ = "serial"
		case int64, uint64:
			typ = "bigserial"
		}
	default:
		panic("unsupported sql adaptor, please submit an issue in github")
	}
	return typ
}

func getSqlType(adaptor string, column interface{}, tag string) string {
	column = formatColumnValue(column)
	typ, addational_typ, size := parseTag(tag)

	if typ == "-" {
		return ""
	}

	if len(typ) == 0 {
		switch adaptor {
		case "sqlite3":
			switch column.(type) {
			case time.Time:
				typ = "datetime"
			case bool, sql.NullBool:
				typ = "bool"
			case int, int8, int16, int32, uint, uint8, uint16, uint32:
				typ = "integer"
			case int64, uint64, sql.NullInt64:
				typ = "bigint"
			case float32, float64, sql.NullFloat64:
				typ = "real"
			case string, sql.NullString:
				if size > 0 && size < 65532 {
					typ = fmt.Sprintf("varchar(%d)", size)
				} else {
					typ = "text"
				}
			default:
				panic("invalid sql type")
			}
		case "mysql":
			switch column.(type) {
			case time.Time:
				typ = "timestamp"
			case bool, sql.NullBool:
				typ = "boolean"
			case int, int8, int16, int32, uint, uint8, uint16, uint32:
				typ = "int"
			case int64, uint64, sql.NullInt64:
				typ = "bigint"
			case float32, float64, sql.NullFloat64:
				typ = "double"
			case []byte:
				if size > 0 && size < 65532 {
					typ = fmt.Sprintf("varbinary(%d)", size)
				} else {
					typ = "longblob"
				}
			case string, sql.NullString:
				if size > 0 && size < 65532 {
					typ = fmt.Sprintf("varchar(%d)", size)
				} else {
					typ = "longtext"
				}
			default:
				panic("invalid sql type")
			}

		case "postgres":
			switch column.(type) {
			case time.Time:
				typ = "timestamp with time zone"
			case bool, sql.NullBool:
				typ = "boolean"
			case int, int8, int16, int32, uint, uint8, uint16, uint32:
				typ = "integer"
			case int64, uint64, sql.NullInt64:
				typ = "bigint"
			case float32, float64, sql.NullFloat64:
				typ = "double precision"
			case []byte:
				typ = "bytea"
			case string, sql.NullString:
				if size > 0 && size < 65532 {
					typ = fmt.Sprintf("varchar(%d)", size)
				} else {
					typ = "text"
				}
			default:
				panic("invalid sql type")
			}
		default:
			panic("unsupported sql adaptor, please submit an issue in github")
		}
	}

	if len(addational_typ) > 0 {
		typ = typ + " " + addational_typ
	}
	return typ
}
