package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func BeforeCreate(db *gorm.DB) {
	// before save
	// before create
}

func SaveBeforeAssociations(db *gorm.DB) {
}

func Create(db *gorm.DB) {
	db.Statement.AddClauseIfNotExists(clause.Insert{
		Table: clause.Table{Name: db.Statement.Table},
	})
	db.Statement.AddClause(ConvertToCreateValues(db.Statement))

	db.Statement.Build("INSERT", "VALUES", "ON_CONFLICT")
	result, err := db.DB.ExecContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)

	if err == nil {
		if db.Statement.Schema != nil {
			if insertID, err := result.LastInsertId(); err == nil {
				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					for i := db.Statement.ReflectValue.Len() - 1; i >= 0; i-- {
						db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.ReflectValue.Index(i), insertID)
						insertID--
					}
				case reflect.Struct:
					db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.ReflectValue, insertID)
				}
			}
		}
		db.RowsAffected, _ = result.RowsAffected()
	} else {
		db.AddError(err)
	}
}

func SaveAfterAssociations(db *gorm.DB) {
}

func AfterCreate(db *gorm.DB) {
	// after save
	// after create
}

// ConvertToCreateValues convert to create values
func ConvertToCreateValues(stmt *gorm.Statement) clause.Values {
	switch value := stmt.Dest.(type) {
	case map[string]interface{}:
		return ConvertMapToValues(stmt, value)
	case []map[string]interface{}:
		return ConvertSliceOfMapToValues(stmt, value)
	default:
		var (
			values                    = clause.Values{}
			selectColumns, restricted = SelectAndOmitColumns(stmt)
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
