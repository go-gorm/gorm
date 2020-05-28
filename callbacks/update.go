package callbacks

import (
	"reflect"
	"sort"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
)

func BeforeUpdate(db *gorm.DB) {
	if db.Statement.Schema != nil && (db.Statement.Schema.BeforeSave || db.Statement.Schema.BeforeUpdate) {
		callMethod := func(value interface{}) bool {
			var ok bool
			if db.Statement.Schema.BeforeSave {
				if i, ok := value.(gorm.BeforeSaveInterface); ok {
					ok = true
					i.BeforeSave(db)
				}
			}

			if db.Statement.Schema.BeforeUpdate {
				if i, ok := value.(gorm.BeforeUpdateInterface); ok {
					ok = true
					i.BeforeUpdate(db)
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

func Update(db *gorm.DB) {
	if db.Statement.Schema != nil && !db.Statement.Unscoped {
		for _, c := range db.Statement.Schema.UpdateClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.String() == "" {
		db.Statement.AddClauseIfNotExists(clause.Update{})
		if set := ConvertToAssignments(db.Statement); len(set) != 0 {
			db.Statement.AddClause(set)
		} else {
			return
		}
		db.Statement.Build("UPDATE", "SET", "WHERE")
	}

	result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

	if err == nil {
		db.RowsAffected, _ = result.RowsAffected()
	} else {
		db.AddError(err)
	}
}

func AfterUpdate(db *gorm.DB) {
	if db.Statement.Schema != nil && (db.Statement.Schema.AfterSave || db.Statement.Schema.AfterUpdate) {
		callMethod := func(value interface{}) bool {
			var ok bool
			if db.Statement.Schema.AfterSave {
				if i, ok := value.(gorm.AfterSaveInterface); ok {
					ok = true
					i.AfterSave(db)
				}
			}

			if db.Statement.Schema.AfterUpdate {
				if i, ok := value.(gorm.AfterUpdateInterface); ok {
					ok = true
					i.AfterUpdate(db)
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

// ConvertToAssignments convert to update assignments
func ConvertToAssignments(stmt *gorm.Statement) (set clause.Set) {
	var (
		selectColumns, restricted = SelectAndOmitColumns(stmt, false, true)
		reflectModelValue         = reflect.Indirect(reflect.ValueOf(stmt.Model))
		assignValue               func(field *schema.Field, value interface{})
	)

	switch reflectModelValue.Kind() {
	case reflect.Slice, reflect.Array:
		assignValue = func(field *schema.Field, value interface{}) {
			for i := 0; i < reflectModelValue.Len(); i++ {
				field.Set(reflectModelValue.Index(i), value)
			}
		}
	case reflect.Struct:
		assignValue = func(field *schema.Field, value interface{}) {
			field.Set(reflectModelValue, value)
		}
	default:
		assignValue = func(field *schema.Field, value interface{}) {
		}
	}

	switch value := stmt.Dest.(type) {
	case map[string]interface{}:
		set = make([]clause.Assignment, 0, len(value))

		var keys []string
		for k, _ := range value {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if field := stmt.Schema.LookUpField(k); field != nil {
				if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
					if field.AutoUpdateTime > 0 {
						value[k] = time.Now()
					}
					set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: value[k]})
					assignValue(field, value[k])
				}
			} else if v, ok := selectColumns[k]; (ok && v) || (!ok && !restricted) {
				set = append(set, clause.Assignment{Column: clause.Column{Name: k}, Value: value[k]})
			}
		}

		for _, field := range stmt.Schema.FieldsByDBName {
			if field.AutoUpdateTime > 0 && value[field.Name] == nil && value[field.DBName] == nil {
				now := time.Now()
				set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now})
				assignValue(field, now)
			}
		}
	default:
		switch stmt.ReflectValue.Kind() {
		case reflect.Struct:
			set = make([]clause.Assignment, 0, len(stmt.Schema.FieldsByDBName))
			for _, field := range stmt.Schema.FieldsByDBName {
				if !field.PrimaryKey || stmt.Dest != stmt.Model {
					if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
						value, isZero := field.ValueOf(stmt.ReflectValue)
						if field.AutoUpdateTime > 0 {
							value = time.Now()
							isZero = false
						}

						if ok || !isZero {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: value})
							assignValue(field, value)
						}
					}
				} else {
					if value, isZero := field.ValueOf(stmt.ReflectValue); !isZero {
						stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
					}
				}
			}
		}
	}

	if stmt.Dest != stmt.Model {
		reflectValue := reflect.Indirect(reflect.ValueOf(stmt.Model))
		switch reflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			var priamryKeyExprs []clause.Expression
			for i := 0; i < reflectValue.Len(); i++ {
				var exprs = make([]clause.Expression, len(stmt.Schema.PrimaryFields))
				var notZero bool
				for idx, field := range stmt.Schema.PrimaryFields {
					value, isZero := field.ValueOf(reflectValue.Index(i))
					exprs[idx] = clause.Eq{Column: field.DBName, Value: value}
					notZero = notZero || !isZero
				}
				if notZero {
					priamryKeyExprs = append(priamryKeyExprs, clause.And(exprs...))
				}
			}
			stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Or(priamryKeyExprs...)}})
		case reflect.Struct:
			for _, field := range stmt.Schema.PrimaryFields {
				if value, isZero := field.ValueOf(reflectValue); !isZero {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
				}
			}
		}
	}

	return
}
