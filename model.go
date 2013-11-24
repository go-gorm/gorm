package gorm

import (
	"errors"
	"go/ast"
	"reflect"
	"regexp"
	"time"
)

var modelFieldMap = map[string][]reflect.StructField{}

type Model struct {
	data          interface{}
	do            *Do
	_cache_fields map[string][]*Field
}

func (m *Model) reflectData() reflect.Value {
	return reflect.Indirect(reflect.ValueOf(m.data))
}

func (m *Model) primaryKeyZero() bool {
	return isBlank(reflect.ValueOf(m.primaryKeyValue()))
}

func (m *Model) primaryKeyValue() interface{} {
	if data := m.reflectData(); data.Kind() == reflect.Struct {
		if field := data.FieldByName(m.primaryKey()); field.IsValid() {
			return field.Interface()
		}
	}
	return 0
}

func (m *Model) primaryKey() string {
	return "Id"
}

func (m *Model) primaryKeyDb() string {
	return toSnake(m.primaryKey())
}

func getStructs(typ reflect.Type) (fs []reflect.StructField) {
	name := typ.Name()
	if fs = modelFieldMap[name]; fs != nil {
		return
	}

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous && ast.IsExported(p.Name) {
			fs = append(fs, p)
		}
	}

	modelFieldMap[name] = fs
	return
}

func (m *Model) fields(operation string) (fields []*Field) {
	if len(m._cache_fields[operation]) > 0 {
		return m._cache_fields[operation]
	}

	indirect_value := m.reflectData()
	if !indirect_value.IsValid() {
		return
	}

	structs := getStructs(indirect_value.Type())
	c := make(chan *Field, len(structs))
	defer close(c)

	for _, field_struct := range structs {
		go func(field_struct reflect.StructField, c chan *Field) {
			var field Field
			field.Name = field_struct.Name
			field.dbName = toSnake(field_struct.Name)
			field.isPrimaryKey = m.primaryKeyDb() == field.dbName
			value := indirect_value.FieldByName(field_struct.Name)
			field.model = m

			if time_value, is_time := value.Interface().(time.Time); is_time {
				field.autoCreateTime = "created_at" == field.dbName
				field.autoUpdateTime = "updated_at" == field.dbName

				switch operation {
				case "create":
					if (field.autoCreateTime || field.autoUpdateTime) && time_value.IsZero() {
						value.Set(reflect.ValueOf(time.Now()))
					}
				case "update":
					if field.autoUpdateTime {
						value.Set(reflect.ValueOf(time.Now()))
					}
				}
			}
			field.structField = field_struct
			field.reflectValue = value
			field.Value = value.Interface()
			field.parseAssociation()
			field.parseBlank()
			c <- &field
		}(field_struct, c)
	}

	for i := 0; i < len(structs); i++ {
		fields = append(fields, <-c)
	}

	if len(m._cache_fields) == 0 {
		m._cache_fields = map[string][]*Field{}
	}
	m._cache_fields[operation] = fields
	return
}

func (m *Model) columnsHasValue(operation string) (fields []*Field) {
	for _, field := range m.fields(operation) {
		if !field.isBlank {
			fields = append(fields, field)
		}
	}
	return
}

func (m *Model) updatedColumnsAndValues(values map[string]interface{}, ignore_protected_attrs bool) (results map[string]interface{}, any_updated bool) {
	if m.data == nil {
		return values, true
	}

	data := m.reflectData()
	if !data.CanAddr() {
		return
	}

	for key, value := range values {
		if field := data.FieldByName(snakeToUpperCamel(key)); field.IsValid() {
			if field.Interface() != value {
				switch field.Kind() {
				case reflect.Int, reflect.Int32, reflect.Int64:
					if field.Int() != reflect.ValueOf(value).Int() {
						any_updated = true
						field.SetInt(reflect.ValueOf(value).Int())
					}
				default:
					any_updated = true
					field.Set(reflect.ValueOf(value))
				}
			}
		}
	}

	if values["updated_at"] != nil && any_updated {
		setFieldValue(data.FieldByName("UpdatedAt"), time.Now())
	}
	return
}

func (m *Model) columnsAndValues(operation string) map[string]interface{} {
	results := map[string]interface{}{}

	for _, field := range m.fields(operation) {
		if !field.isPrimaryKey && len(field.sqlTag()) > 0 {
			results[field.dbName] = field.Value
		}
	}
	return results
}

func (m *Model) hasColumn(name string) bool {
	if data := m.reflectData(); data.Kind() == reflect.Struct {
		return data.FieldByName(name).IsValid()
	} else if data.Kind() == reflect.Slice {
		return reflect.New(data.Type().Elem()).Elem().FieldByName(name).IsValid()
	}
	return false
}

func (m *Model) columnAndValue(name string) (has_column bool, is_slice bool, value interface{}) {
	if data := m.reflectData(); data.Kind() == reflect.Struct {
		if has_column = data.FieldByName(name).IsValid(); has_column {
			value = data.FieldByName(name).Interface()
		}
	} else if data.Kind() == reflect.Slice {
		has_column = reflect.New(data.Type().Elem()).Elem().FieldByName(name).IsValid()
		is_slice = true
	}
	return
}

func (m *Model) typ() reflect.Type {
	typ := m.reflectData().Type()
	if typ.Kind() == reflect.Slice {
		return typ.Elem()
	} else {
		return typ
	}
}

func (m *Model) typeName() string {
	return m.typ().Name()
}

func (m *Model) tableName() (str string) {
	if m.data == nil {
		m.do.err(errors.New("Model haven't been set"))
		return
	}

	data := m.reflectData()
	if fm := data.MethodByName("TableName"); fm.IsValid() {
		if v := fm.Call([]reflect.Value{}); len(v) > 0 {
			if result, ok := v[0].Interface().(string); ok {
				return result
			}
		}
	}

	str = toSnake(m.typeName())

	if !m.do.db.parent.singularTable {
		pluralMap := map[string]string{"ch": "ches", "ss": "sses", "sh": "shes", "day": "days", "y": "ies", "x": "xes", "s?": "s"}
		for key, value := range pluralMap {
			reg := regexp.MustCompile(key + "$")
			if reg.MatchString(str) {
				return reg.ReplaceAllString(str, value)
			}
		}
	}

	return
}

func (m *Model) callMethod(method string) {
	if m.data == nil || m.do.db.hasError() {
		return
	}

	if fm := reflect.ValueOf(m.data).MethodByName(method); fm.IsValid() {
		numin := fm.Type().NumIn()
		var results []reflect.Value
		if numin == 0 {
			results = fm.Call([]reflect.Value{})
		} else if numin == 1 {
			results = fm.Call([]reflect.Value{reflect.ValueOf(m.do.db.new())})
		}
		if len(results) > 0 {
			if verr, ok := results[0].Interface().(error); ok {
				m.do.err(verr)
			}
		}
	}
	return
}

func (m *Model) setValueByColumn(name string, value interface{}, out interface{}) {
	data := reflect.Indirect(reflect.ValueOf(out))
	setFieldValue(data.FieldByName(snakeToUpperCamel(name)), value)
}

func (m *Model) beforeAssociations() (fields []*Field) {
	for _, field := range m.fields("null") {
		if field.beforeAssociation && !field.isBlank {
			fields = append(fields, field)
		}
	}
	return
}

func (m *Model) afterAssociations() (fields []*Field) {
	for _, field := range m.fields("null") {
		if field.afterAssociation && !field.isBlank {
			fields = append(fields, field)
		}
	}
	return
}
