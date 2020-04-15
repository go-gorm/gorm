package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
)

func BeforeCreate(db *gorm.DB) {
	if db.Statement.Schema != nil && (db.Statement.Schema.BeforeSave || db.Statement.Schema.BeforeCreate) {
		callMethod := func(value interface{}) bool {
			var ok bool
			if db.Statement.Schema.BeforeSave {
				if i, ok := value.(gorm.BeforeSaveInterface); ok {
					ok = true
					i.BeforeSave(db)
				}
			}

			if db.Statement.Schema.BeforeCreate {
				if i, ok := value.(gorm.BeforeCreateInterface); ok {
					ok = true
					i.BeforeCreate(db)
				}
			}
			return ok
		}

		if ok := callMethod(db.Statement.Dest); !ok {
			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i <= db.Statement.ReflectValue.Len(); i++ {
					callMethod(db.Statement.ReflectValue.Index(i).Interface())
				}
			case reflect.Struct:
				callMethod(db.Statement.ReflectValue.Interface())
			}
		}
	}
}

func SaveBeforeAssociations(db *gorm.DB) {
	if db.Statement.Schema != nil {
		for _, rel := range db.Statement.Schema.Relationships.BelongsTo {
			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice:
			case reflect.Struct:
				if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
					f := rel.Field.ReflectValueOf(db.Statement.ReflectValue)
					if f.Kind() == reflect.Ptr {
						db.Session(&gorm.Session{}).Create(f.Interface())
					} else {
						db.Session(&gorm.Session{}).Create(f.Addr().Interface())
					}

					for _, ref := range rel.References {
						if !ref.OwnPrimaryKey {
							fv, _ := ref.PrimaryKey.ValueOf(f)
							ref.ForeignKey.Set(db.Statement.ReflectValue, fv)
						}
					}
				}
			}
		}
	}
}

func Create(config *Config) func(db *gorm.DB) {
	if config.WithReturning {
		return CreateWithReturning
	} else {
		return func(db *gorm.DB) {
			db.Statement.AddClauseIfNotExists(clause.Insert{
				Table: clause.Table{Name: db.Statement.Table},
			})
			db.Statement.AddClause(ConvertToCreateValues(db.Statement))

			db.Statement.Build("INSERT", "VALUES", "ON_CONFLICT")
			result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

			if err == nil {
				if db.Statement.Schema != nil {
					if insertID, err := result.LastInsertId(); err == nil {
						switch db.Statement.ReflectValue.Kind() {
						case reflect.Slice, reflect.Array:
							if config.LastInsertIDReversed {
								for i := db.Statement.ReflectValue.Len() - 1; i >= 0; i-- {
									db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.ReflectValue.Index(i), insertID)
									insertID--
								}
							} else {
								for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
									db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.ReflectValue.Index(i), insertID)
									insertID++
								}
							}
						case reflect.Struct:
							db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.ReflectValue, insertID)
						}
					} else {
						db.AddError(err)
					}
				}
				db.RowsAffected, _ = result.RowsAffected()
			} else {
				db.AddError(err)
			}
		}
	}
}

func CreateWithReturning(db *gorm.DB) {
	db.Statement.AddClauseIfNotExists(clause.Insert{
		Table: clause.Table{Name: db.Statement.Table},
	})
	db.Statement.AddClause(ConvertToCreateValues(db.Statement))

	db.Statement.Build("INSERT", "VALUES", "ON_CONFLICT")

	if sch := db.Statement.Schema; sch != nil && len(sch.FieldsWithDefaultDBValue) > 0 {
		db.Statement.WriteString(" RETURNING ")

		var (
			idx    int
			fields = make([]*schema.Field, len(sch.FieldsWithDefaultDBValue))
			values = make([]interface{}, len(sch.FieldsWithDefaultDBValue))
		)

		for dbName, field := range sch.FieldsWithDefaultDBValue {
			if idx != 0 {
				db.Statement.WriteByte(',')
			}

			fields[idx] = field
			db.Statement.WriteQuoted(dbName)
			idx++
		}

		rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

		if err == nil {
			defer rows.Close()

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for rows.Next() {
					for idx, field := range fields {
						values[idx] = field.ReflectValueOf(db.Statement.ReflectValue.Index(int(db.RowsAffected))).Addr().Interface()
					}
					if err := rows.Scan(values...); err != nil {
						db.AddError(err)
					}
					db.RowsAffected++
				}
			case reflect.Struct:
				for idx, field := range fields {
					values[idx] = field.ReflectValueOf(db.Statement.ReflectValue).Addr().Interface()
				}

				if rows.Next() {
					err = rows.Scan(values...)
				}
			}
		}

		if err != nil {
			db.AddError(err)
		}
	} else {
		if result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...); err == nil {
			db.RowsAffected, _ = result.RowsAffected()
		} else {
			db.AddError(err)
		}
	}
}

func SaveAfterAssociations(db *gorm.DB) {
}

func AfterCreate(db *gorm.DB) {
	if db.Statement.Schema != nil && (db.Statement.Schema.AfterSave || db.Statement.Schema.AfterCreate) {
		callMethod := func(value interface{}) bool {
			var ok bool
			if db.Statement.Schema.AfterSave {
				if i, ok := value.(gorm.AfterSaveInterface); ok {
					ok = true
					i.AfterSave(db)
				}
			}

			if db.Statement.Schema.AfterCreate {
				if i, ok := value.(gorm.AfterCreateInterface); ok {
					ok = true
					i.AfterCreate(db)
				}
			}
			return ok
		}

		if ok := callMethod(db.Statement.Dest); !ok {
			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i <= db.Statement.ReflectValue.Len(); i++ {
					callMethod(db.Statement.ReflectValue.Index(i).Interface())
				}
			case reflect.Struct:
				callMethod(db.Statement.ReflectValue.Interface())
			}
		}
	}
}

// ConvertToCreateValues convert to create values
func ConvertToCreateValues(stmt *gorm.Statement) clause.Values {
	switch value := stmt.Dest.(type) {
	case map[string]interface{}:
		return ConvertMapToValuesForCreate(stmt, value)
	case []map[string]interface{}:
		return ConvertSliceOfMapToValuesForCreate(stmt, value)
	default:
		var (
			values                    = clause.Values{}
			selectColumns, restricted = SelectAndOmitColumns(stmt, true, false)
			curTime                   = stmt.DB.NowFunc()
			isZero                    = false
		)

		for _, db := range stmt.Schema.DBNames {
			if stmt.Schema.FieldsWithDefaultDBValue[db] == nil {
				if v, ok := selectColumns[db]; (ok && v) || (!ok && !restricted) {
					values.Columns = append(values.Columns, clause.Column{Name: db})
				}
			}
		}

		switch stmt.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			values.Values = make([][]interface{}, stmt.ReflectValue.Len())
			defaultValueFieldsHavingValue := map[string][]interface{}{}
			for i := 0; i < stmt.ReflectValue.Len(); i++ {
				rv := reflect.Indirect(stmt.ReflectValue.Index(i))
				values.Values[i] = make([]interface{}, len(values.Columns))
				for idx, column := range values.Columns {
					field := stmt.Schema.FieldsByDBName[column.Name]
					if values.Values[i][idx], isZero = field.ValueOf(rv); isZero {
						if field.DefaultValueInterface != nil {
							values.Values[i][idx] = field.DefaultValueInterface
							field.Set(rv, field.DefaultValueInterface)
						} else if field.AutoCreateTime > 0 || field.AutoUpdateTime > 0 {
							field.Set(rv, curTime)
							values.Values[i][idx], _ = field.ValueOf(rv)
						}
					}
				}

				for db, field := range stmt.Schema.FieldsWithDefaultDBValue {
					if v, ok := selectColumns[db]; (ok && v) || (!ok && !restricted) {
						if v, isZero := field.ValueOf(rv); !isZero {
							if len(defaultValueFieldsHavingValue[db]) == 0 {
								defaultValueFieldsHavingValue[db] = make([]interface{}, stmt.ReflectValue.Len())
							}
							defaultValueFieldsHavingValue[db][i] = v
						}
					}
				}
			}

			for db, vs := range defaultValueFieldsHavingValue {
				values.Columns = append(values.Columns, clause.Column{Name: db})
				for idx := range values.Values {
					if vs[idx] == nil {
						values.Values[idx] = append(values.Values[idx], clause.Expr{SQL: "DEFAULT"})
					} else {
						values.Values[idx] = append(values.Values[idx], vs[idx])
					}
				}
			}
		case reflect.Struct:
			values.Values = [][]interface{}{make([]interface{}, len(values.Columns))}
			for idx, column := range values.Columns {
				field := stmt.Schema.FieldsByDBName[column.Name]
				if values.Values[0][idx], isZero = field.ValueOf(stmt.ReflectValue); isZero {
					if field.DefaultValueInterface != nil {
						values.Values[0][idx] = field.DefaultValueInterface
						field.Set(stmt.ReflectValue, field.DefaultValueInterface)
					} else if field.AutoCreateTime > 0 || field.AutoUpdateTime > 0 {
						field.Set(stmt.ReflectValue, curTime)
						values.Values[0][idx], _ = field.ValueOf(stmt.ReflectValue)
					}
				}
			}

			for db, field := range stmt.Schema.FieldsWithDefaultDBValue {
				if v, ok := selectColumns[db]; (ok && v) || (!ok && !restricted) {
					if v, isZero := field.ValueOf(stmt.ReflectValue); !isZero {
						values.Columns = append(values.Columns, clause.Column{Name: db})
						values.Values[0] = append(values.Values[0], v)
					}
				}
			}
		}

		return values
	}
}
