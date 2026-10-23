package callbacks

import (
	"reflect"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// SetupUpdateReflectValue chooses the model value used by update hooks and associations
// when the current reflection value is unaddressable or Model differs from Dest. For
// map updates it also copies explicitly supplied belongs-to association values into the
// model. It runs before BeforeUpdate in the default chain.
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
						db.AddError(rel.Field.Set(db.Statement.Context, db.Statement.ReflectValue, dest[rel.Name]))
					}
				}
			}
		}
	}
}

// BeforeUpdate invokes BeforeSave followed by BeforeUpdate on models when db has no
// error, schema metadata enables the hooks, and SkipHooks is false. Errors are recorded
// on db and cause rollback of an owned default transaction.
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

// Update returns the dialector-configured UPDATE callback. It adds schema update
// clauses (the built-in soft-delete modifier checks Unscoped), builds UPDATE/SET from
// the destination when SQL is empty,
// and returns without execution when there are no assignments. It checks the
// missing-WHERE guard even in DryRun; execution requires no error and DryRun=false.
//
// Supported RETURNING queries scan into the model and close their rows; other updates
// use ExecContext and report RowsAffected. The surrounding processor handles hooks,
// associations, and transactions. Explicit SQL and expressions remain trusted caller
// input.
func Update(config *Config) func(db *gorm.DB) {
	supportReturning := utils.Contains(config.UpdateClauses, "RETURNING")

	return func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		if db.Statement.Schema != nil {
			for _, c := range db.Statement.Schema.UpdateClauses {
				db.Statement.AddClause(c)
			}
		}

		if db.Statement.SQL.Len() == 0 {
			db.Statement.SQL.Grow(180)
			db.Statement.AddClauseIfNotExists(clause.Update{})
			if _, ok := db.Statement.Clauses["SET"]; !ok {
				if set := ConvertToAssignments(db.Statement); len(set) != 0 {
					defer delete(db.Statement.Clauses, "SET")
					db.Statement.AddClause(set)
				} else {
					return
				}
			}

			db.Statement.Build(db.Statement.BuildClauses...)
		}

		checkMissingWhereConditions(db)

		if !db.DryRun && db.Error == nil {
			if ok, mode := hasReturning(db, supportReturning); ok {
				if rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...); db.AddError(err) == nil {
					defer func() {
						if err := rows.Close(); err != nil {
							_ = db.AddError(err)
						}
					}()

					dest := db.Statement.Dest
					db.Statement.Dest = db.Statement.ReflectValue.Addr().Interface()
					gorm.Scan(rows, db, mode)
					db.Statement.Dest = dest

					if db.Statement.Result != nil {
						db.Statement.Result.RowsAffected = db.RowsAffected
					}
				}
			} else {
				result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

				if db.AddError(err) == nil {
					db.RowsAffected, _ = result.RowsAffected()
				}

				if db.Statement.Result != nil {
					db.Statement.Result.Result = result
					db.Statement.Result.RowsAffected = db.RowsAffected
				}
			}
		}
	}
}

// AfterUpdate invokes AfterUpdate followed by AfterSave when db has no error, schema
// metadata enables the hooks, and SkipHooks is false. It runs after association saving
// and before the default transaction commits; it does not prove a transaction has
// committed.
func AfterUpdate(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && (db.Statement.Schema.AfterSave || db.Statement.Schema.AfterUpdate) {
		callMethod(db, func(value interface{}, tx *gorm.DB) (called bool) {
			if db.Statement.Schema.AfterUpdate {
				if i, ok := value.(AfterUpdateInterface); ok {
					called = true
					db.AddError(i.AfterUpdate(tx))
				}
			}

			if db.Statement.Schema.AfterSave {
				if i, ok := value.(AfterSaveInterface); ok {
					called = true
					db.AddError(i.AfterSave(tx))
				}
			}

			return called
		})
	}
}

// ConvertToAssignments builds SET assignments and may add primary-key WHERE conditions
// from stmt.Model. Select/Omit and update permissions filter columns. Map keys are
// sorted and preserve explicitly supplied zero values; struct updates normally omit
// zero values unless selected. Automatic update timestamps are generated unless hooks
// are skipped.
//
// It maps Go field names to database columns, supports expressions as values, and
// copies applicable values back into the model used by hooks. When Model and Dest
// differ it can parse the destination's schema for field lookup. Unsupported data
// records ErrInvalidData. The returned assignments can be empty; callers should avoid
// executing an empty SET.
func ConvertToAssignments(stmt *gorm.Statement) (set clause.Set) {
	var (
		selectColumns, restricted = stmt.SelectAndOmitColumns(false, true)
		assignValue               func(field *schema.Field, value interface{})
	)

	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		assignValue = func(field *schema.Field, value interface{}) {
			for i := 0; i < stmt.ReflectValue.Len(); i++ {
				if stmt.ReflectValue.CanAddr() {
					field.Set(stmt.Context, stmt.ReflectValue.Index(i), value)
				}
			}
		}
	case reflect.Struct:
		assignValue = func(field *schema.Field, value interface{}) {
			if stmt.ReflectValue.CanAddr() {
				field.Set(stmt.Context, stmt.ReflectValue, value)
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
			if size := stmt.ReflectValue.Len(); size > 0 {
				var isZero bool
				for i := 0; i < size; i++ {
					for _, field := range stmt.Schema.PrimaryFields {
						_, isZero = field.ValueOf(stmt.Context, stmt.ReflectValue.Index(i))
						if !isZero {
							break
						}
					}
				}

				if !isZero {
					_, primaryValues := schema.GetIdentityFieldValuesMap(stmt.Context, stmt.ReflectValue, stmt.Schema.PrimaryFields)
					column, values := schema.ToQueryValues("", stmt.Schema.PrimaryFieldDBNames, primaryValues)
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}
			}
		case reflect.Struct:
			for _, field := range stmt.Schema.PrimaryFields {
				if value, isZero := field.ValueOf(stmt.Context, stmt.ReflectValue); !isZero {
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
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.UnixMilli()})
						} else if field.AutoUpdateTime == schema.UnixSecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.Unix()})
						} else {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now})
						}
					}
				}
			}
		}
	default:
		updatingSchema := stmt.Schema
		var isDiffSchema bool
		if !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
			// different schema
			updatingStmt := &gorm.Statement{DB: stmt.DB}
			if err := updatingStmt.Parse(stmt.Dest); err == nil {
				updatingSchema = updatingStmt.Schema
				isDiffSchema = true
			}
		}

		switch updatingValue.Kind() {
		case reflect.Struct:
			set = make([]clause.Assignment, 0, len(stmt.Schema.FieldsByDBName))
			for _, dbName := range stmt.Schema.DBNames {
				if field := updatingSchema.LookUpField(dbName); field != nil {
					if !field.PrimaryKey || !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
						if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && (!restricted || (!stmt.SkipHooks && field.AutoUpdateTime > 0))) {
							value, isZero := field.ValueOf(stmt.Context, updatingValue)
							if !stmt.SkipHooks && field.AutoUpdateTime > 0 {
								if field.AutoUpdateTime == schema.UnixNanosecond {
									value = stmt.DB.NowFunc().UnixNano()
								} else if field.AutoUpdateTime == schema.UnixMillisecond {
									value = stmt.DB.NowFunc().UnixMilli()
								} else if field.AutoUpdateTime == schema.UnixSecond {
									value = stmt.DB.NowFunc().Unix()
								} else {
									value = stmt.DB.NowFunc()
								}
								isZero = false
							}

							if (ok || !isZero) && field.Updatable {
								set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: value})
								assignField := field
								if isDiffSchema {
									if originField := stmt.Schema.LookUpField(dbName); originField != nil {
										assignField = originField
									}
								}
								assignValue(assignField, value)
							}
						}
					} else {
						if value, isZero := field.ValueOf(stmt.Context, updatingValue); !isZero {
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
