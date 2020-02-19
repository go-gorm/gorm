package callbacks

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func BeforeCreate(db *gorm.DB) {
	// before save
	// before create

	// assign timestamp
}

func SaveBeforeAssociations(db *gorm.DB) {
}

func Create(db *gorm.DB) {
	db.Statement.AddClauseIfNotExists(clause.Insert{
		Table: clause.Table{Name: db.Statement.Table},
	})
	values, _ := ConvertToCreateValues(db.Statement)
	db.Statement.AddClause(values)

	db.Statement.Build("INSERT", "VALUES", "ON_CONFLICT")
	result, err := db.DB.ExecContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)

	fmt.Printf("%+v\n", values)
	fmt.Println(err)
	fmt.Println(result)
	fmt.Println(db.Statement.SQL.String(), db.Statement.Vars)
}

func SaveAfterAssociations(db *gorm.DB) {
}

func AfterCreate(db *gorm.DB) {
	// after save
	// after create
}

// ConvertToCreateValues convert to create values
func ConvertToCreateValues(stmt *gorm.Statement) (clause.Values, []map[string]interface{}) {
	switch value := stmt.Dest.(type) {
	case map[string]interface{}:
		return ConvertMapToValues(stmt, value), nil
	case []map[string]interface{}:
		return ConvertSliceOfMapToValues(stmt, value), nil
	default:
		var (
			values                    = clause.Values{}
			selectColumns, restricted = SelectAndOmitColumns(stmt)
			curTime                   = stmt.DB.NowFunc()
			isZero                    = false
			returnningValues          []map[string]interface{}
		)

		for _, db := range stmt.Schema.DBNames {
			if v, ok := selectColumns[db]; (ok && v) || (!ok && !restricted) {
				values.Columns = append(values.Columns, clause.Column{Name: db})
			}
		}

		reflectValue := reflect.Indirect(reflect.ValueOf(stmt.Dest))
		switch reflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			values.Values = make([][]interface{}, reflectValue.Len())
			for i := 0; i < reflectValue.Len(); i++ {
				rv := reflect.Indirect(reflectValue.Index(i))
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
						} else if field.HasDefaultValue {
							if len(returnningValues) == 0 {
								returnningValues = make([]map[string]interface{}, reflectValue.Len())
							}

							if returnningValues[i] == nil {
								returnningValues[i] = map[string]interface{}{}
							}

							// FIXME
							returnningValues[i][column.Name] = field.ReflectValueOf(reflectValue).Addr().Interface()
						}
					}
				}
			}
		case reflect.Struct:
			values.Values = [][]interface{}{make([]interface{}, len(values.Columns))}
			for idx, column := range values.Columns {
				field := stmt.Schema.FieldsByDBName[column.Name]
				if values.Values[0][idx], _ = field.ValueOf(reflectValue); isZero {
					if field.DefaultValueInterface != nil {
						values.Values[0][idx] = field.DefaultValueInterface
						field.Set(reflectValue, field.DefaultValueInterface)
					} else if field.AutoCreateTime > 0 || field.AutoUpdateTime > 0 {
						field.Set(reflectValue, curTime)
						values.Values[0][idx], _ = field.ValueOf(reflectValue)
					} else if field.HasDefaultValue {
						if len(returnningValues) == 0 {
							returnningValues = make([]map[string]interface{}, 1)
						}

						values.Values[0][idx] = clause.Expr{SQL: "DEFAULT"}
						returnningValues[0][column.Name] = field.ReflectValueOf(reflectValue).Addr().Interface()
					} else if field.PrimaryKey {
					}
				}
			}
		}
		return values, returnningValues
	}
}
