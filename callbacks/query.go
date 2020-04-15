package callbacks

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

func Query(db *gorm.DB) {
	if db.Statement.SQL.String() == "" {
		clauseSelect := clause.Select{}

		if len(db.Statement.Selects) > 0 {
			for _, name := range db.Statement.Selects {
				if f := db.Statement.Schema.LookUpField(name); f != nil {
					clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
						Name: f.DBName,
					})
				}
			}
		}

		if len(db.Statement.Joins) != 0 {
			joins := []clause.Join{}

			if len(db.Statement.Selects) == 0 {
				for _, dbName := range db.Statement.Schema.DBNames {
					clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
						Name: dbName,
					})
				}
			}

			for name, conds := range db.Statement.Joins {
				if relation, ok := db.Statement.Schema.Relationships.Relations[name]; ok {
					for _, s := range relation.FieldSchema.DBNames {
						clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
							Table: relation.FieldSchema.Table,
							Name:  s,
						})
					}

					var exprs []clause.Expression
					for _, ref := range relation.References {
						if ref.OwnPrimaryKey {
							exprs = append(exprs, clause.Expr{
								SQL: fmt.Sprintf("%s.%s = %s.%s", db.Statement.Schema.Table, ref.PrimaryKey.DBName, relation.FieldSchema.Table, ref.ForeignKey.DBName),
							})
						} else {
							if ref.PrimaryValue == "" {
								exprs = append(exprs, clause.Expr{
									SQL: fmt.Sprintf("%s.%s = %s.%s", db.Statement.Schema.Table, ref.ForeignKey.DBName, relation.FieldSchema.Table, ref.PrimaryKey.DBName),
								})
							} else {
								exprs = append(exprs, clause.Expr{
									SQL:  fmt.Sprintf("%s.%s = ?", relation.FieldSchema.Table, ref.PrimaryKey.DBName),
									Vars: []interface{}{ref.PrimaryValue},
								})
							}
						}
					}

					joins = append(joins, clause.Join{
						Type:  clause.LeftJoin,
						Table: clause.Table{Name: relation.FieldSchema.Table},
						ON:    clause.Where{Exprs: exprs},
					})
				} else {
					joins = append(joins, clause.Join{
						Expression: clause.Expr{SQL: name, Vars: conds},
					})
				}
			}

			db.Statement.AddClause(clause.From{Joins: joins})
		} else {
			db.Statement.AddClauseIfNotExists(clause.From{})
		}

		db.Statement.AddClauseIfNotExists(clauseSelect)
		db.Statement.Build("SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR")
	}

	rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	if err != nil {
		db.AddError(err)
		return
	}
	defer rows.Close()

	Scan(rows, db)
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
