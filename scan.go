package gorm

import (
	"database/sql"
	"reflect"
	"strings"

	"github.com/jinzhu/gorm/schema"
)

func Scan(rows *sql.Rows, db *DB, initialized bool) {
	columns, _ := rows.Columns()
	values := make([]interface{}, len(columns))

	switch dest := db.Statement.Dest.(type) {
	case map[string]interface{}, *map[string]interface{}:
		for idx, _ := range columns {
			values[idx] = new(interface{})
		}

		if initialized || rows.Next() {
			db.RowsAffected++
			db.AddError(rows.Scan(values...))
		}

		mapValue, ok := dest.(map[string]interface{})
		if ok {
			if v, ok := dest.(*map[string]interface{}); ok {
				mapValue = *v
			}
		}

		for idx, column := range columns {
			mapValue[column] = *(values[idx].(*interface{}))
		}
	case *[]map[string]interface{}:
		for idx, _ := range columns {
			values[idx] = new(interface{})
		}

		for initialized || rows.Next() {
			initialized = false
			db.RowsAffected++
			db.AddError(rows.Scan(values...))

			v := map[string]interface{}{}
			for idx, column := range columns {
				v[column] = *(values[idx].(*interface{}))
			}
			*dest = append(*dest, v)
		}
	case *int, *int64, *uint, *uint64:
		for initialized || rows.Next() {
			initialized = false
			db.RowsAffected++
			db.AddError(rows.Scan(dest))
		}
	default:
		switch db.Statement.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			reflectValueType := db.Statement.ReflectValue.Type().Elem()
			isPtr := reflectValueType.Kind() == reflect.Ptr
			if isPtr {
				reflectValueType = reflectValueType.Elem()
			}

			db.Statement.ReflectValue.Set(reflect.MakeSlice(db.Statement.ReflectValue.Type(), 0, 0))
			fields := make([]*schema.Field, len(columns))
			joinFields := make([][2]*schema.Field, len(columns))

			for idx, column := range columns {
				if field := db.Statement.Schema.LookUpField(column); field != nil && field.Readable {
					fields[idx] = field
				} else if names := strings.Split(column, "__"); len(names) > 1 {
					if rel, ok := db.Statement.Schema.Relationships.Relations[names[0]]; ok {
						if field := rel.FieldSchema.LookUpField(strings.Join(names[1:], "__")); field != nil && field.Readable {
							joinFields[idx] = [2]*schema.Field{rel.Field, field}
							continue
						}
					}
					values[idx] = &sql.RawBytes{}
				} else {
					values[idx] = &sql.RawBytes{}
				}
			}

			for initialized || rows.Next() {
				initialized = false
				elem := reflect.New(reflectValueType).Elem()

				if reflectValueType.Kind() != reflect.Struct && len(fields) == 1 {
					values[0] = elem.Addr().Interface()
				} else {
					for idx, field := range fields {
						if field != nil {
							values[idx] = field.ReflectValueOf(elem).Addr().Interface()
						} else if joinFields[idx][0] != nil {
							relValue := joinFields[idx][0].ReflectValueOf(elem)
							if relValue.Kind() == reflect.Ptr && relValue.IsNil() {
								relValue.Set(reflect.New(relValue.Type().Elem()))
							}

							values[idx] = joinFields[idx][1].ReflectValueOf(relValue).Addr().Interface()
						}
					}
				}

				db.RowsAffected++
				db.AddError(rows.Scan(values...))

				if isPtr {
					db.Statement.ReflectValue.Set(reflect.Append(db.Statement.ReflectValue, elem.Addr()))
				} else {
					db.Statement.ReflectValue.Set(reflect.Append(db.Statement.ReflectValue, elem))
				}
			}
		case reflect.Struct:
			for idx, column := range columns {
				if field := db.Statement.Schema.LookUpField(column); field != nil && field.Readable {
					values[idx] = field.ReflectValueOf(db.Statement.ReflectValue).Addr().Interface()
				} else if names := strings.Split(column, "__"); len(names) > 1 {
					if rel, ok := db.Statement.Schema.Relationships.Relations[names[0]]; ok {
						relValue := rel.Field.ReflectValueOf(db.Statement.ReflectValue)
						if field := rel.FieldSchema.LookUpField(strings.Join(names[1:], "__")); field != nil && field.Readable {
							if relValue.Kind() == reflect.Ptr && relValue.IsNil() {
								relValue.Set(reflect.New(relValue.Type().Elem()))
							}

							values[idx] = field.ReflectValueOf(relValue).Addr().Interface()
							continue
						}
					}
					values[idx] = &sql.RawBytes{}
				} else {
					values[idx] = &sql.RawBytes{}
				}
			}

			if initialized || rows.Next() {
				db.RowsAffected++
				db.AddError(rows.Scan(values...))
			}
		}
	}

	if db.RowsAffected == 0 && db.Statement.RaiseErrorOnNotFound {
		db.AddError(ErrRecordNotFound)
	}
}
