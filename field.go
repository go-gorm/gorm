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
	Tag               string
	AddationalTag     string
	Size              int
	SqlTag            string
	ForeignKey        string
	BeforeAssociation bool
	AfterAssociation  bool

	foreignKey        string
	beforeAssociation bool
	afterAssociation  bool

	dbName         string
	model          *Model
	isBlank        bool
	ignoreField    bool
	isPrimaryKey   bool
	autoCreateTime bool
	autoUpdateTime bool
	reflectValue   reflect.Value
	structField    reflect.StructField
}

func (f *Field) IsScanner() bool {
	_, is_scanner := reflect.New(reflect.ValueOf(f.Value).Type()).Interface().(sql.Scanner)
	return is_scanner
}

func (f *Field) IsTime() bool {
	_, is_time := f.Value.(time.Time)
	return is_time
}

func (f *Field) parseAssociation() {
	elem := reflect.Indirect(reflect.ValueOf(f.Value))
	typ := elem.Type()

	switch elem.Kind() {
	case reflect.Slice:
		typ = typ.Elem()

		if _, ok := f.Value.([]byte); !ok {
			foreignKey := typ.Name() + "Id"
			if reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
				f.foreignKey = foreignKey
			}
			f.afterAssociation = true
		}
	case reflect.Struct:
		if !f.IsTime() && !f.IsScanner() {
			if elem.FieldByName(f.Name + "Id").IsValid() {
				f.foreignKey = f.Name + "Id"
				f.beforeAssociation = true
			} else {
				foreignKey := typ.Name() + "Id"
				if reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
					f.foreignKey = foreignKey
				}
				f.afterAssociation = true
			}
		}
	}
}

func (f *Field) parseBlank() {
	f.isBlank = isBlank(f.reflectValue)
}

func (f *Field) parseIgnore() {
	typ, _, _ := parseSqlTag(f.structField.Tag.Get(f.model.do.db.parent.tagIdentifier))

	if typ == "-" {
		f.ignoreField = true
	}
}

func (f *Field) isScanner() bool {
	_, is_scanner := reflect.New(reflect.ValueOf(f.Value).Type()).Interface().(sql.Scanner)
	return is_scanner
}

func (f *Field) isTime() bool {
	_, is_time := f.Value.(time.Time)
	return is_time
}

func (f *Field) sqlTag() (str string) {
	value := f.Value
	if f.isScanner() {
		value = f.reflectValue.Field(0).Interface()
	}
	reflect_value := f.reflectValue

	switch reflect_value.Kind() {
	case reflect.Slice:
		if _, ok := f.Value.([]byte); !ok {
			return
		}
	case reflect.Struct:
		if !f.isTime() && !f.isScanner() {
			return
		}
	}

	typ, addational_typ, size := parseSqlTag(f.structField.Tag.Get(f.model.do.db.parent.tagIdentifier))

	if typ == "-" {
		return
	}

	if len(typ) == 0 {
		if f.isPrimaryKey {
			typ = f.model.do.dialect().PrimaryKeyTag(value, size)
		} else {
			typ = f.model.do.dialect().SqlTag(value, size)
		}
	}

	if len(addational_typ) > 0 {
		typ = typ + " " + addational_typ
	}
	return typ
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
