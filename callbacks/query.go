package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func Query(db *gorm.DB) {
	if db.Statement.SQL.String() == "" {
		db.Statement.AddClauseIfNotExists(clause.Select{})
		db.Statement.AddClauseIfNotExists(clause.From{})
		db.Statement.Build("SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR")
	}

	rows, err := db.DB.QueryContext(db.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	db.AddError(err)
	_ = rows
	// scan rows
}

func Preload(db *gorm.DB) {
}

func AfterQuery(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.AfterFind {
		callMethod := func(value interface{}) bool {
			if db.Statement.Schema.AfterFind {
				if i, ok := value.(gorm.AfterFindInterface); ok {
					i.AfterFind(db)
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
