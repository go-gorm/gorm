package callbacks

import (
	"reflect"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func SetupUpdateReflectValue(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil {
		if !db.Statement.ReflectValue.CanAddr() || db.Statement.Model != db.Statement.Dest {
			db.Statement.ReflectValue = reflect.ValueOf(db.Statement.Model)
			for db.Statement.ReflectValue.Kind() == reflect.Ptr {
				db.Statement.ReflectValue = db.Statement.ReflectValue.Elem()
			}

			if dest, ok := db.Statement.Dest.(map[string]interface{}); ok {
				for _, rel := range db.Statement.Schema.Relationships.BelongsTo {
					if _, ok := dest[rel.Name]; ok {
						rel.Field.Set(db.Statement.ReflectValue, dest[rel.Name])
					}
				}
			}
		}
	}
}

func BeforeUpdate(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && (db.Statement.Schema.BeforeSave || db.Statement.Schema.BeforeUpdate) {
		callMethod(db, func(value interface{}, tx *gorm.DB) (called bool) {
			if db.Statement.Schema.BeforeSave {
				if i, ok := value.(BeforeSaveInterface); ok {
					called = true
					db.AddError(i.BeforeSave(tx))
				}
			}

			if db.Statement.Schema.BeforeUpdate {
				if i, ok := value.(BeforeUpdateInterface); ok {
					called = true
					db.AddError(i.BeforeUpdate(tx))
				}
			}

			return called
		})
	}
}

func Update(db *gorm.DB) {
	if db.Error != nil {
		return
	}

	if db.Statement.Schema != nil && !db.Statement.Unscoped {
		for _, c := range db.Statement.Schema.UpdateClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.String() == "" {
		db.Statement.SQL.Grow(180)
		db.Statement.AddClauseIfNotExists(clause.Update{})
		if set := ConvertToAssignments(db.Statement); len(set) != 0 {
			db.Statement.AddClause(set)
		} else {
			return
		}
		db.Statement.Build(db.Statement.BuildClauses...)
	}

	if _, ok := db.Statement.Clauses["WHERE"]; !db.AllowGlobalUpdate && !ok {
		db.AddError(gorm.ErrMissingWhereClause)
		return
	}

	if !db.DryRun && db.Error == nil {
		result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

		if err == nil {
			db.RowsAffected, _ = result.RowsAffected()
		} else {
			db.AddError(err)
		}
	}
}

func AfterUpdate(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && (db.Statement.Schema.AfterSave || db.Statement.Schema.AfterUpdate) {
		callMethod(db, func(value interface{}, tx *gorm.DB) (called bool) {
			if db.Statement.Schema.AfterSave {
				if i, ok := value.(AfterSaveInterface); ok {
					called = true
					db.AddError(i.AfterSave(tx))
				}
			}

			if db.Statement.Schema.AfterUpdate {
				if i, ok := value.(AfterUpdateInterface); ok {
					called = true
					db.AddError(i.AfterUpdate(tx))
				}
			}
			return called
		})
	}
}

// ConvertToAssignments convert to update assignments
func ConvertToAssignments(stmt *gorm.Statement) (set clause.Set) {
	var (
		selectColumns, restricted = stmt.SelectAndOmitColumns(false, true)
		assignValue               func(field *schema.Field, value interface{})
	)

	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		assignValue = func(field *schema.Field, value interface{}) {
			for i := 0; i < stmt.ReflectValue.Len(); i++ {
				field.Set(stmt.ReflectValue.Index(i), value)
			}
		}
	case reflect.Struct:
		assignValue = func(field *schema.Field, value interface{}) {
			if stmt.ReflectValue.CanAddr() {
				field.Set(stmt.ReflectValue, value)
			}
		}
	default:
		assignValue = func(field *schema.Field, value interface{}) {
		}
	}

	updatingValue := reflect.ValueOf(stmt.Dest)
	for updatingValue.Kind() == reflect.Ptr {
		updatingValue = updatingValue.Elem()
	}

	if !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
		switch stmt.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			var primaryKeyExprs []clause.Expression
			for i := 0; i < stmt.ReflectValue.Len(); i++ {
				var exprs = make([]clause.Expression, len(stmt.Schema.PrimaryFields))
				var notZero bool
				for idx, field := range stmt.Schema.PrimaryFields {
					value, isZero := field.ValueOf(stmt.ReflectValue.Index(i))
					exprs[idx] = clause.Eq{Column: field.DBName, Value: value}
					notZero = notZero || !isZero
				}
				if notZero {
					primaryKeyExprs = append(primaryKeyExprs, clause.And(exprs...))
				}
			}
			stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Or(primaryKeyExprs...)}})
		case reflect.Struct:
			for _, field := range stmt.Schema.PrimaryFields {
				if value, isZero := field.ValueOf(stmt.ReflectValue); !isZero {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
				}
			}
		}
	}

	switch value := updatingValue.Interface().(type) {
	case map[string]interface{}:
		set = make([]clause.Assignment, 0, len(value))

		keys := make([]string, 0, len(value))
		for k := range value {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			kv := value[k]
			if _, ok := kv.(*gorm.DB); ok {
				kv = []interface{}{kv}
			}

			if stmt.Schema != nil {
				if field := stmt.Schema.LookUpField(k); field != nil {
					if field.DBName != "" {
						if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: kv})
							assignValue(field, value[k])
						}
					} else if v, ok := selectColumns[field.Name]; (ok && v) || (!ok && !restricted) {
						assignValue(field, value[k])
					}
					continue
				}
			}

			if v, ok := selectColumns[k]; (ok && v) || (!ok && !restricted) {
				set = append(set, clause.Assignment{Column: clause.Column{Name: k}, Value: kv})
			}
		}

		if !stmt.SkipHooks && stmt.Schema != nil {
			for _, dbName := range stmt.Schema.DBNames {
				field := stmt.Schema.LookUpField(dbName)
				if field.AutoUpdateTime > 0 && value[field.Name] == nil && value[field.DBName] == nil {
					if v, ok := selectColumns[field.DBName]; (ok && v) || !ok {
						now := stmt.DB.NowFunc()
						assignValue(field, now)

						if field.AutoUpdateTime == schema.UnixNanosecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.UnixNano()})
						} else if field.AutoUpdateTime == schema.UnixMillisecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.UnixNano() / 1e6})
						} else if field.GORMDataType == schema.Time {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now})
						} else {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.Unix()})
						}
					}
				}
			}
		}
	default:
		var updatingSchema = stmt.Schema
		if !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
			// different schema
			updatingStmt := &gorm.Statement{DB: stmt.DB}
			if err := updatingStmt.Parse(stmt.Dest); err == nil {
				updatingSchema = updatingStmt.Schema
			}
		}

		switch updatingValue.Kind() {
		case reflect.Struct:
			set = make([]clause.Assignment, 0, len(stmt.Schema.FieldsByDBName))
			for _, dbName := range stmt.Schema.DBNames {
				if field := updatingSchema.LookUpField(dbName); field != nil && field.Updatable {
					if !field.PrimaryKey || !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
						if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && (!restricted || (!stmt.SkipHooks && field.AutoUpdateTime > 0))) {
							value, isZero := field.ValueOf(updatingValue)
							if !stmt.SkipHooks && field.AutoUpdateTime > 0 {
								if field.AutoUpdateTime == schema.UnixNanosecond {
									value = stmt.DB.NowFunc().UnixNano()
								} else if field.AutoUpdateTime == schema.UnixMillisecond {
									value = stmt.DB.NowFunc().UnixNano() / 1e6
								} else if field.GORMDataType == schema.Time {
									value = stmt.DB.NowFunc()
								} else {
									value = stmt.DB.NowFunc().Unix()
								}
								isZero = false
							}

							if ok || !isZero {
								set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: value})
								assignValue(field, value)
							}
						}
					} else {
						if value, isZero := field.ValueOf(updatingValue); !isZero {
							stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
						}
					}
				}
			}
		default:
			stmt.AddError(gorm.ErrInvalidData)
		}
	}

	return
}
