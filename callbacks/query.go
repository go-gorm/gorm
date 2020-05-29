package callbacks

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
)

func Query(db *gorm.DB) {
	if db.Statement.Schema != nil && !db.Statement.Unscoped {
		for _, c := range db.Statement.Schema.QueryClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.String() == "" {
		clauseSelect := clause.Select{}

		if len(db.Statement.Selects) > 0 {
			for _, name := range db.Statement.Selects {
				if f := db.Statement.Schema.LookUpField(name); f != nil {
					clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
						Name: f.DBName,
					})
				} else {
					clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
						Name: name,
						Raw:  true,
					})
				}
			}
		}

		// inline joins
		if len(db.Statement.Joins) != 0 {
			joins := []clause.Join{}

			if len(db.Statement.Selects) == 0 {
				for _, dbName := range db.Statement.Schema.DBNames {
					clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
						Table: db.Statement.Table,
						Name:  dbName,
					})
				}
			}

			for name, conds := range db.Statement.Joins {
				if relation, ok := db.Statement.Schema.Relationships.Relations[name]; ok {
					tableAliasName := relation.Name

					for _, s := range relation.FieldSchema.DBNames {
						clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
							Table: tableAliasName,
							Name:  s,
							Alias: tableAliasName + "__" + s,
						})
					}

					var exprs []clause.Expression
					for _, ref := range relation.References {
						if ref.OwnPrimaryKey {
							exprs = append(exprs, clause.Eq{
								Column: clause.Column{Table: db.Statement.Schema.Table, Name: ref.PrimaryKey.DBName},
								Value:  clause.Column{Table: tableAliasName, Name: ref.ForeignKey.DBName},
							})
						} else {
							if ref.PrimaryValue == "" {
								exprs = append(exprs, clause.Eq{
									Column: clause.Column{Table: db.Statement.Schema.Table, Name: ref.ForeignKey.DBName},
									Value:  clause.Column{Table: tableAliasName, Name: ref.PrimaryKey.DBName},
								})
							} else {
								exprs = append(exprs, clause.Eq{
									Column: clause.Column{Table: tableAliasName, Name: ref.ForeignKey.DBName},
									Value:  ref.PrimaryValue,
								})
							}
						}
					}

					joins = append(joins, clause.Join{
						Type:  clause.LeftJoin,
						Table: clause.Table{Name: relation.FieldSchema.Table, Alias: tableAliasName},
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

		if len(clauseSelect.Columns) > 0 {
			db.Statement.AddClause(clauseSelect)
		} else {
			db.Statement.AddClauseIfNotExists(clauseSelect)
		}
		db.Statement.Build("SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR")
	}

	rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	if err != nil {
		db.AddError(err)
		return
	}
	defer rows.Close()

	gorm.Scan(rows, db, false)
}

func Preload(db *gorm.DB) {
	if len(db.Statement.Preloads) > 0 {
		preloadMap := map[string][]string{}
		for name := range db.Statement.Preloads {
			preloadFields := strings.Split(name, ".")
			for idx := range preloadFields {
				preloadMap[strings.Join(preloadFields[:idx+1], ".")] = preloadFields[:idx+1]
			}
		}

		preloadNames := make([]string, len(preloadMap))
		idx := 0
		for key := range preloadMap {
			preloadNames[idx] = key
			idx++
		}
		sort.Strings(preloadNames)

		for _, name := range preloadNames {
			var (
				curSchema     = db.Statement.Schema
				preloadFields = preloadMap[name]
				rels          = make([]*schema.Relationship, len(preloadFields))
			)

			for idx, preloadField := range preloadFields {
				if rel := curSchema.Relationships.Relations[preloadField]; rel != nil {
					rels[idx] = rel
					curSchema = rel.FieldSchema
				} else {
					db.AddError(fmt.Errorf("%v: %w", name, gorm.ErrUnsupportedRelation))
				}
			}

			preload(db, rels, db.Statement.Preloads[name])
		}
	}
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
