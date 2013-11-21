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
	Value             interface{}
	model             *Model
	dbName            string
	isBlank           bool
	isPrimaryKey      bool
	autoCreateTime    bool
	autoUpdateTime    bool
	foreignKey        string
	beforeAssociation bool
	afterAssociation  bool
	reflectValue      reflect.Value
	structField       reflect.StructField
}

func (f *Field) parseBlank() {
	f.isBlank = isBlank(f.reflectValue)
}

func (f *Field) isScanner() bool {
	_, is_scanner := reflect.New(f.reflectValue.Type()).Interface().(sql.Scanner)
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

func (f *Field) parseAssociation() {
	reflect_value := f.reflectValue

	switch reflect_value.Kind() {
	case reflect.Slice:
		if _, ok := f.Value.([]byte); !ok {
			foreign_key := f.model.typeName() + "Id"
			if reflect.New(reflect_value.Type().Elem()).Elem().FieldByName(foreign_key).IsValid() {
				f.foreignKey = foreign_key
			}
			f.afterAssociation = true
		}
	case reflect.Struct:
		if !f.isTime() && !f.isScanner() {
			if f.model.reflectData().FieldByName(f.Name + "Id").IsValid() {
				f.foreignKey = f.Name + "Id"
				f.beforeAssociation = true
			} else {
				foreign_key := f.model.typeName() + "Id"
				if reflect.New(reflect_value.Type()).Elem().FieldByName(foreign_key).IsValid() {
					f.foreignKey = foreign_key
				}
				f.afterAssociation = true
			}
		}
	}
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
