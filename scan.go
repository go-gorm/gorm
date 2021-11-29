package gorm

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm/schema"
)

func prepareValues(values []interface{}, db *DB, columnTypes []*sql.ColumnType, columns []string) {
	if db.Statement.Schema != nil {
		for idx, name := range columns {
			if field := db.Statement.Schema.LookUpField(name); field != nil {
				values[idx] = reflect.New(reflect.PtrTo(field.FieldType)).Interface()
				continue
			}
			values[idx] = new(interface{})
		}
	} else if len(columnTypes) > 0 {
		for idx, columnType := range columnTypes {
			if columnType.ScanType() != nil {
				values[idx] = reflect.New(reflect.PtrTo(columnType.ScanType())).Interface()
			} else {
				values[idx] = new(interface{})
			}
		}
	} else {
		for idx := range columns {
			values[idx] = new(interface{})
		}
	}
}

func scanIntoMap(mapValue map[string]interface{}, values []interface{}, columns []string) {
	for idx, column := range columns {
		if reflectValue := reflect.Indirect(reflect.Indirect(reflect.ValueOf(values[idx]))); reflectValue.IsValid() {
			mapValue[column] = reflectValue.Interface()
			if valuer, ok := mapValue[column].(driver.Valuer); ok {
				mapValue[column], _ = valuer.Value()
			} else if b, ok := mapValue[column].(sql.RawBytes); ok {
				mapValue[column] = string(b)
			}
		} else {
			mapValue[column] = nil
		}
	}
}

func (db *DB) scanIntoStruct(sch *schema.Schema, rows *sql.Rows, reflectValue reflect.Value, values []interface{}, columns []string, fields []*schema.Field, joinFields [][2]*schema.Field) {
	for idx, column := range columns {
		if sch == nil {
			values[idx] = reflectValue.Interface()
		} else if field := sch.LookUpField(column); field != nil && field.Readable {
			values[idx] = reflect.New(reflect.PtrTo(field.IndirectFieldType)).Interface()
		} else if names := strings.Split(column, "__"); len(names) > 1 {
			if rel, ok := sch.Relationships.Relations[names[0]]; ok {
				if field := rel.FieldSchema.LookUpField(strings.Join(names[1:], "__")); field != nil && field.Readable {
					values[idx] = reflect.New(reflect.PtrTo(field.IndirectFieldType)).Interface()
					continue
				}
			}
			values[idx] = &sql.RawBytes{}
		} else if len(columns) == 1 {
			sch = nil
			values[idx] = reflectValue.Interface()
		} else {
			values[idx] = &sql.RawBytes{}
		}
	}

	db.RowsAffected++
	db.AddError(rows.Scan(values...))

	if sch != nil {
		for idx, column := range columns {
			if field := sch.LookUpField(column); field != nil && field.Readable {
				field.Set(reflectValue, values[idx])
			} else if names := strings.Split(column, "__"); len(names) > 1 {
				if rel, ok := sch.Relationships.Relations[names[0]]; ok {
					if field := rel.FieldSchema.LookUpField(strings.Join(names[1:], "__")); field != nil && field.Readable {
						relValue := rel.Field.ReflectValueOf(reflectValue)
						value := reflect.ValueOf(values[idx]).Elem()

						if relValue.Kind() == reflect.Ptr && relValue.IsNil() {
							if value.IsNil() {
								continue
							}
							relValue.Set(reflect.New(relValue.Type().Elem()))
						}

						field.Set(relValue, values[idx])
					}
				}
			}
		}
	}
}

type ScanMode uint8

const (
	ScanInitialized         ScanMode = 1 << 0 // 1
	ScanUpdate              ScanMode = 1 << 1 // 2
	ScanOnConflictDoNothing ScanMode = 1 << 2 // 4
)

func Scan(rows *sql.Rows, db *DB, mode ScanMode) {
	var (
		columns, _          = rows.Columns()
		values              = make([]interface{}, len(columns))
		initialized         = mode&ScanInitialized != 0
		update              = mode&ScanUpdate != 0
		onConflictDonothing = mode&ScanOnConflictDoNothing != 0
	)

	db.RowsAffected = 0

	switch dest := db.Statement.Dest.(type) {
	case map[string]interface{}, *map[string]interface{}:
		if initialized || rows.Next() {
			columnTypes, _ := rows.ColumnTypes()
			prepareValues(values, db, columnTypes, columns)

			db.RowsAffected++
			db.AddError(rows.Scan(values...))

			mapValue, ok := dest.(map[string]interface{})
			if !ok {
				if v, ok := dest.(*map[string]interface{}); ok {
					if *v == nil {
						*v = map[string]interface{}{}
					}
					mapValue = *v
				}
			}
			scanIntoMap(mapValue, values, columns)
		}
	case *[]map[string]interface{}, []map[string]interface{}:
		columnTypes, _ := rows.ColumnTypes()
		for initialized || rows.Next() {
			prepareValues(values, db, columnTypes, columns)

			initialized = false
			db.RowsAffected++
			db.AddError(rows.Scan(values...))

			mapValue := map[string]interface{}{}
			scanIntoMap(mapValue, values, columns)
			if values, ok := dest.([]map[string]interface{}); ok {
				values = append(values, mapValue)
			} else if values, ok := dest.(*[]map[string]interface{}); ok {
				*values = append(*values, mapValue)
			}
		}
	case *int, *int8, *int16, *int32, *int64,
		*uint, *uint8, *uint16, *uint32, *uint64, *uintptr,
		*float32, *float64,
		*bool, *string, *time.Time,
		*sql.NullInt32, *sql.NullInt64, *sql.NullFloat64,
		*sql.NullBool, *sql.NullString, *sql.NullTime:
		for initialized || rows.Next() {
			initialized = false
			db.RowsAffected++
			db.AddError(rows.Scan(dest))
		}
	default:
		var (
			fields       = make([]*schema.Field, len(columns))
			joinFields   [][2]*schema.Field
			sch          = db.Statement.Schema
			reflectValue = db.Statement.ReflectValue
		)

		if reflectValue.Kind() == reflect.Interface {
			reflectValue = reflectValue.Elem()
		}

		reflectValueType := reflectValue.Type()
		switch reflectValueType.Kind() {
		case reflect.Array, reflect.Slice:
			reflectValueType = reflectValueType.Elem()
		}
		isPtr := reflectValueType.Kind() == reflect.Ptr
		if isPtr {
			reflectValueType = reflectValueType.Elem()
		}

		if sch != nil {
			if reflectValueType != sch.ModelType && reflectValueType.Kind() == reflect.Struct {
				sch, _ = schema.Parse(db.Statement.Dest, db.cacheStore, db.NamingStrategy)
			}

			for idx, column := range columns {
				if field := sch.LookUpField(column); field != nil && field.Readable {
					fields[idx] = field
				} else if names := strings.Split(column, "__"); len(names) > 1 {
					if rel, ok := sch.Relationships.Relations[names[0]]; ok {
						if field := rel.FieldSchema.LookUpField(strings.Join(names[1:], "__")); field != nil && field.Readable {
							fields[idx] = field

							if len(joinFields) == 0 {
								joinFields = make([][2]*schema.Field, len(columns))
							}
							joinFields[idx] = [2]*schema.Field{rel.Field, field}
							continue
						}
					}
					values[idx] = &sql.RawBytes{}
				} else {
					values[idx] = &sql.RawBytes{}
				}
			}

			if len(columns) == 1 {
				// isPluck
				if _, ok := reflect.New(reflectValueType).Interface().(sql.Scanner); (reflectValueType != sch.ModelType && ok) || // is scanner
					reflectValueType.Kind() != reflect.Struct || // is not struct
					sch.ModelType.ConvertibleTo(schema.TimeReflectType) { // is time
					sch = nil
				}
			}
		}

		switch reflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			var elem reflect.Value

			if !update || reflectValue.Len() == 0 {
				update = false
				db.Statement.ReflectValue.Set(reflect.MakeSlice(reflectValue.Type(), 0, 20))
			}

			for initialized || rows.Next() {
			BEGIN:
				initialized = false

				if update {
					if int(db.RowsAffected) >= reflectValue.Len() {
						return
					}
					elem = reflectValue.Index(int(db.RowsAffected))
					if onConflictDonothing {
						for _, field := range fields {
							if _, ok := field.ValueOf(elem); !ok {
								db.RowsAffected++
								goto BEGIN
							}
						}
					}
				} else {
					elem = reflect.New(reflectValueType)
				}

				db.scanIntoStruct(sch, rows, elem, values, columns, fields, joinFields)

				if !update {
					if isPtr {
						reflectValue = reflect.Append(reflectValue, elem)
					} else {
						reflectValue = reflect.Append(reflectValue, elem.Elem())
					}
				}
			}

			if !update {
				db.Statement.ReflectValue.Set(reflectValue)
			}
		case reflect.Struct, reflect.Ptr:
			if initialized || rows.Next() {
				db.scanIntoStruct(sch, rows, reflectValue, values, columns, fields, joinFields)
			}
		default:
			db.AddError(rows.Scan(dest))
		}
	}

	if err := rows.Err(); err != nil && err != db.Error {
		db.AddError(err)
	}

	if db.RowsAffected == 0 && db.Statement.RaiseErrorOnNotFound && db.Error == nil {
		db.AddError(ErrRecordNotFound)
	}
}
