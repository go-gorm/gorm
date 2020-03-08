package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func BeforeDelete(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.BeforeDelete {
		callMethod := func(value interface{}) bool {
			if db.Statement.Schema.BeforeDelete {
				if i, ok := value.(gorm.BeforeDeleteInterface); ok {
					i.BeforeDelete(db)
					return true
				}
			}
			return false
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

func Delete(db *gorm.DB) {
	if db.Statement.SQL.String() == "" {
		db.Statement.AddClauseIfNotExists(clause.Delete{})

		values := []reflect.Value{db.Statement.ReflectValue}
		if db.Statement.Dest != db.Statement.Model {
			values = append(values, reflect.ValueOf(db.Statement.Model))
		}
		for _, field := range db.Statement.Schema.PrimaryFields {
			for _, value := range values {
				if value, isZero := field.ValueOf(value); !isZero {
					db.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
				}
			}
		}

		if _, ok := db.Statement.Clauses["WHERE"]; !ok {
			db.AddError(gorm.ErrMissingWhereClause)
			return
		}

		db.Statement.AddClauseIfNotExists(clause.From{})
		db.Statement.Build("DELETE", "FROM", "WHERE")
	}

	result, err := db.DB.ExecContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)

	if err == nil {
		db.RowsAffected, _ = result.RowsAffected()
	} else {
		db.AddError(err)
	}
}

func AfterDelete(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.AfterDelete {
		callMethod := func(value interface{}) bool {
			if db.Statement.Schema.AfterDelete {
				if i, ok := value.(gorm.AfterDeleteInterface); ok {
					i.AfterDelete(db)
					return true
				}
			}
			return false
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
