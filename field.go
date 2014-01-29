package gorm

import (
	"database/sql"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Field struct {
	Name              string
	DBName            string
	Value             interface{}
	IsBlank           bool
	IsIgnored         bool
	Tag               reflect.StructTag
	SqlTag            string
	ForeignKey        string
	BeforeAssociation bool
	AfterAssociation  bool
	isPrimaryKey      bool
}

func (f *Field) IsScanner() bool {
	_, is_scanner := reflect.New(reflect.ValueOf(f.Value).Type()).Interface().(sql.Scanner)
	return is_scanner
}

func (f *Field) IsTime() bool {
	_, is_time := f.Value.(time.Time)
	return is_time
}

func parseSqlTag(str string) (typ string, addational_typ string, size int) {
	if str == "-" {
		typ = str
	} else if str != "" {
		tags := strings.Split(str, ";")
		m := make(map[string]string)
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
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
