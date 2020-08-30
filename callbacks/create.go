package callbacks

import (
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func BeforeCreate(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && (db.Statement.Schema.BeforeSave || db.Statement.Schema.BeforeCreate) {
		callMethod(db, func(value interface{}, tx *gorm.DB) (called bool) {
			if db.Statement.Schema.BeforeSave {
				if i, ok := value.(BeforeSaveInterface); ok {
					called = true
					db.AddError(i.BeforeSave(tx))
				}
			}

			if db.Statement.Schema.BeforeCreate {
				if i, ok := value.(BeforeCreateInterface); ok {
					called = true
					db.AddError(i.BeforeCreate(tx))
				}
			}
			return called
		})
	}
}

func Create(config *Config) func(db *gorm.DB) {
	if config.WithReturning {
		return CreateWithReturning
	} else {
		return func(db *gorm.DB) {
			if db.Error == nil {
				if db.Statement.Schema != nil && !db.Statement.Unscoped {
					for _, c := range db.Statement.Schema.CreateClauses {
						db.Statement.AddClause(c)
					}
				}

				if db.Statement.SQL.String() == "" {
					db.Statement.SQL.Grow(180)
					db.Statement.AddClauseIfNotExists(clause.Insert{})
					db.Statement.AddClause(ConvertToCreateValues(db.Statement))

					db.Statement.Build("INSERT", "VALUES", "ON CONFLICT")
				}

				if !db.DryRun && db.Error == nil {
					result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

					if err == nil {
						db.RowsAffected, _ = result.RowsAffected()
						if db.RowsAffected > 0 {
							if db.Statement.Schema != nil && db.Statement.Schema.PrioritizedPrimaryField != nil && db.Statement.Schema.PrioritizedPrimaryField.HasDefaultValue {
								if insertID, err := result.LastInsertId(); err == nil {
									switch db.Statement.ReflectValue.Kind() {
									case reflect.Slice, reflect.Array:
										if config.LastInsertIDReversed {
											for i := db.Statement.ReflectValue.Len() - 1; i >= 0; i-- {
												rv := db.Statement.ReflectValue.Index(i)
												if reflect.Indirect(rv).Kind() != reflect.Struct {
													break
												}

												_, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(rv)
												if isZero {
													db.Statement.Schema.PrioritizedPrimaryField.Set(rv, insertID)
													insertID--
												}
											}
										} else {
											for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
												rv := db.Statement.ReflectValue.Index(i)
												if reflect.Indirect(rv).Kind() != reflect.Struct {
													break
												}

												if _, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(rv); isZero {
													db.Statement.Schema.PrioritizedPrimaryField.Set(rv, insertID)
													insertID++
												}
											}
										}
									case reflect.Struct:
										if insertID > 0 {
											db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.ReflectValue, insertID)
										}
									}
								} else {
									db.AddError(err)
								}
							}
						}
					} else {
						db.AddError(err)
					}
				}
			}
		}
	}
}

func CreateWithReturning(db *gorm.DB) {
	if db.Error == nil {
		if db.Statement.Schema != nil && !db.Statement.Unscoped {
			for _, c := range db.Statement.Schema.CreateClauses {
				db.Statement.AddClause(c)
			}
		}

		if db.Statement.SQL.String() == "" {
			db.Statement.AddClauseIfNotExists(clause.Insert{})
			db.Statement.AddClause(ConvertToCreateValues(db.Statement))

			db.Statement.Build("INSERT", "VALUES", "ON CONFLICT")
		}

		if sch := db.Statement.Schema; sch != nil && len(sch.FieldsWithDefaultDBValue) > 0 {
			db.Statement.WriteString(" RETURNING ")

			var (
				fields = make([]*schema.Field, len(sch.FieldsWithDefaultDBValue))
				values = make([]interface{}, len(sch.FieldsWithDefaultDBValue))
			)

			for idx, field := range sch.FieldsWithDefaultDBValue {
				if idx > 0 {
					db.Statement.WriteByte(',')
				}

				fields[idx] = field
				db.Statement.WriteQuoted(field.DBName)
			}

			if !db.DryRun && db.Error == nil {
				rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

				if err == nil {
					defer rows.Close()

					switch db.Statement.ReflectValue.Kind() {
					case reflect.Slice, reflect.Array:
						c := db.Statement.Clauses["ON CONFLICT"]
						onConflict, _ := c.Expression.(clause.OnConflict)

						for rows.Next() {
						BEGIN:
							reflectValue := db.Statement.ReflectValue.Index(int(db.RowsAffected))
							if reflect.Indirect(reflectValue).Kind() != reflect.Struct {
								break
							}

							for idx, field := range fields {
								fieldValue := field.ReflectValueOf(reflectValue)

								if onConflict.DoNothing && !fieldValue.IsZero() {
									db.RowsAffected++

									if int(db.RowsAffected) >= db.Statement.ReflectValue.Len() {
										return
									}

									goto BEGIN
								}

								values[idx] = fieldValue.Addr().Interface()
							}

							db.RowsAffected++
							if err := rows.Scan(values...); err != nil {
								db.AddError(err)
							}
						}
					case reflect.Struct:
						for idx, field := range fields {
							values[idx] = field.ReflectValueOf(db.Statement.ReflectValue).Addr().Interface()
						}

						if rows.Next() {
							db.RowsAffected++
							db.AddError(rows.Scan(values...))
						}
					}
				} else {
					db.AddError(err)
				}
			}
		} else if !db.DryRun && db.Error == nil {
			if result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...); err == nil {
				db.RowsAffected, _ = result.RowsAffected()
			} else {
				db.AddError(err)
			}
		}
	}
}

func AfterCreate(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && (db.Statement.Schema.AfterSave || db.Statement.Schema.AfterCreate) {
		callMethod(db, func(value interface{}, tx *gorm.DB) (called bool) {
			if db.Statement.Schema.AfterSave {
				if i, ok := value.(AfterSaveInterface); ok {
					called = true
					db.AddError(i.AfterSave(tx))
				}
			}

			if db.Statement.Schema.AfterCreate {
				if i, ok := value.(AfterCreateInterface); ok {
					called = true
					db.AddError(i.AfterCreate(tx))
				}
			}
			return called
		})
	}
}

// ConvertToCreateValues convert to create values
func ConvertToCreateValues(stmt *gorm.Statement) (values clause.Values) {
	switch value := stmt.Dest.(type) {
	case map[string]interface{}:
		values = ConvertMapToValuesForCreate(stmt, value)
	case *map[string]interface{}:
		values = ConvertMapToValuesForCreate(stmt, *value)
	case []map[string]interface{}:
		values = ConvertSliceOfMapToValuesForCreate(stmt, value)
	case *[]map[string]interface{}:
		values = ConvertSliceOfMapToValuesForCreate(stmt, *value)
	default:
		var (
			selectColumns, restricted = stmt.SelectAndOmitColumns(true, false)
			curTime                   = stmt.DB.NowFunc()
			isZero                    bool
		)
		values = clause.Values{Columns: make([]clause.Column, 0, len(stmt.Schema.DBNames))}

		for _, db := range stmt.Schema.DBNames {
			if field := stmt.Schema.FieldsByDBName[db]; !field.HasDefaultValue || field.DefaultValueInterface != nil {
				if v, ok := selectColumns[db]; (ok && v) || (!ok && !restricted) {
					values.Columns = append(values.Columns, clause.Column{Name: db})
				}
			}
		}

		switch stmt.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			stmt.SQL.Grow(stmt.ReflectValue.Len() * 15)
			values.Values = make([][]interface{}, stmt.ReflectValue.Len())
			defaultValueFieldsHavingValue := map[*schema.Field][]interface{}{}
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

				for _, field := range stmt.Schema.FieldsWithDefaultDBValue {
					if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
						if v, isZero := field.ValueOf(rv); !isZero {
							if len(defaultValueFieldsHavingValue[field]) == 0 {
								defaultValueFieldsHavingValue[field] = make([]interface{}, stmt.ReflectValue.Len())
							}
							defaultValueFieldsHavingValue[field][i] = v
						}
					}
				}
			}

			for field, vs := range defaultValueFieldsHavingValue {
				values.Columns = append(values.Columns, clause.Column{Name: field.DBName})
				for idx := range values.Values {
					if vs[idx] == nil {
						values.Values[idx] = append(values.Values[idx], stmt.Dialector.DefaultValueOf(field))
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

			for _, field := range stmt.Schema.FieldsWithDefaultDBValue {
				if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
					if v, isZero := field.ValueOf(stmt.ReflectValue); !isZero {
						values.Columns = append(values.Columns, clause.Column{Name: field.DBName})
						values.Values[0] = append(values.Values[0], v)
					}
				}
			}
		default:
			stmt.AddError(gorm.ErrInvalidData)
		}
	}

	if stmt.UpdatingColumn {
		if stmt.Schema != nil && len(values.Columns) > 1 {
			columns := make([]string, 0, len(values.Columns)-1)
			for _, column := range values.Columns {
				if field := stmt.Schema.LookUpField(column.Name); field != nil {
					if !field.PrimaryKey && !field.HasDefaultValue && field.AutoCreateTime == 0 {
						columns = append(columns, column.Name)
					}
				}
			}

			onConflict := clause.OnConflict{
				Columns:   make([]clause.Column, len(stmt.Schema.PrimaryFieldDBNames)),
				DoUpdates: clause.AssignmentColumns(columns),
			}

			for idx, field := range stmt.Schema.PrimaryFields {
				onConflict.Columns[idx] = clause.Column{Name: field.DBName}
			}
			stmt.AddClause(onConflict)
		}
	}

	return values
}
