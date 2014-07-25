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
	_, isScanner := reflect.New(reflect.ValueOf(f.Value).Type()).Interface().(sql.Scanner)
	return isScanner
}

func (f *Field) IsTime() bool {
	_, isTime := f.Value.(time.Time)
	return isTime
}

func parseSqlTag(str string) (typ string, additionalType string, size int) {
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

		additionalType = m["NOT NULL"] + " " + m["UNIQUE"]

		if len(m["DEFAULT"]) > 0 {
			additionalType = additionalType + "DEFAULT " + m["DEFAULT"]
		}

	}
	return
}
