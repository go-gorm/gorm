package callbacks

import (
	"fmt"
	"reflect"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Query(db *gorm.DB) {
	if db.Error == nil {
		BuildQuerySQL(db)

		if !db.DryRun && db.Error == nil {
			rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
			if err != nil {
				db.AddError(err)
				return
			}
			defer func() {
				db.AddError(rows.Close())
			}()
			gorm.Scan(rows, db, 0)
		}
	}
}

func BuildQuerySQL(db *gorm.DB) {
	if db.Statement.Schema != nil {
		for _, c := range db.Statement.Schema.QueryClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.Len() == 0 {
		db.Statement.SQL.Grow(100)
		clauseSelect := clause.Select{Distinct: db.Statement.Distinct}

		if db.Statement.ReflectValue.Kind() == reflect.Struct && db.Statement.ReflectValue.Type() == db.Statement.Schema.ModelType {
			var conds []clause.Expression
			for _, primaryField := range db.Statement.Schema.PrimaryFields {
				if v, isZero := primaryField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); !isZero {
					conds = append(conds, clause.Eq{Column: clause.Column{Table: db.Statement.Table, Name: primaryField.DBName}, Value: v})
				}
			}

			if len(conds) > 0 {
				db.Statement.AddClause(clause.Where{Exprs: conds})
			}
		}

		if len(db.Statement.Selects) > 0 {
			clauseSelect.Columns = make([]clause.Column, len(db.Statement.Selects))
			for idx, name := range db.Statement.Selects {
				if db.Statement.Schema == nil {
					clauseSelect.Columns[idx] = clause.Column{Name: name, Raw: true}
				} else if f := db.Statement.Schema.LookUpField(name); f != nil {
					clauseSelect.Columns[idx] = clause.Column{Name: f.DBName}
				} else {
					clauseSelect.Columns[idx] = clause.Column{Name: name, Raw: true}
				}
			}
		} else if db.Statement.Schema != nil && len(db.Statement.Omits) > 0 {
			selectColumns, _ := db.Statement.SelectAndOmitColumns(false, false)
			clauseSelect.Columns = make([]clause.Column, 0, len(db.Statement.Schema.DBNames))
			for _, dbName := range db.Statement.Schema.DBNames {
				if v, ok := selectColumns[dbName]; (ok && v) || !ok {
					clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{Table: db.Statement.Table, Name: dbName})
				}
			}
		} else if db.Statement.Schema != nil && db.Statement.ReflectValue.IsValid() {
			queryFields := db.QueryFields
			if !queryFields {
				switch db.Statement.ReflectValue.Kind() {
				case reflect.Struct:
					queryFields = db.Statement.ReflectValue.Type() != db.Statement.Schema.ModelType
				case reflect.Slice:
					queryFields = db.Statement.ReflectValue.Type().Elem() != db.Statement.Schema.ModelType
				}
			}

			if queryFields {
				stmt := gorm.Statement{DB: db}
				// smaller struct
				if err := stmt.Parse(db.Statement.Dest); err == nil && (db.QueryFields || stmt.Schema.ModelType != db.Statement.Schema.ModelType) {
					clauseSelect.Columns = make([]clause.Column, len(stmt.Schema.DBNames))

					for idx, dbName := range stmt.Schema.DBNames {
						clauseSelect.Columns[idx] = clause.Column{Table: db.Statement.Table, Name: dbName}
					}
				}
			}
		}

		// inline joins
		fromClause := clause.From{}
		if v, ok := db.Statement.Clauses["FROM"].Expression.(clause.From); ok {
			fromClause = v
		}

		if len(db.Statement.Joins) != 0 || len(fromClause.Joins) != 0 {
			fromClause.Joins = append(fromClause.Joins, gorm.GenJoinClauses(db, &clauseSelect)...)
			db.Statement.AddClause(fromClause)
		} else {
			db.Statement.AddClauseIfNotExists(clause.From{})
		}

		db.Statement.AddClauseIfNotExists(clauseSelect)

		db.Statement.Build(db.Statement.BuildClauses...)
	}
}

func Preload(db *gorm.DB) {
	if db.Error == nil && len(db.Statement.Preloads) > 0 {
		if db.Statement.Schema == nil {
			db.AddError(fmt.Errorf("%w when using preload", gorm.ErrModelValueRequired))
			return
		}

		preloadMap := parsePreloadMap(db.Statement.Schema, db.Statement.Preloads)
		preloadNames := make([]string, 0, len(preloadMap))
		for key := range preloadMap {
			preloadNames = append(preloadNames, key)
		}
		sort.Strings(preloadNames)

		preloadDB := db.Session(&gorm.Session{Context: db.Statement.Context, NewDB: true, SkipHooks: db.Statement.SkipHooks, Initialized: true})
		db.Statement.Settings.Range(func(k, v interface{}) bool {
			preloadDB.Statement.Settings.Store(k, v)
			return true
		})

		if err := preloadDB.Statement.Parse(db.Statement.Dest); err != nil {
			if err := preloadDB.Statement.Parse(db.Statement.Model); err != nil {
				return
			}
		}
		preloadDB.Statement.ReflectValue = db.Statement.ReflectValue
		preloadDB.Statement.Unscoped = db.Statement.Unscoped

		for _, name := range preloadNames {
			if relations := preloadDB.Statement.Schema.Relationships.EmbeddedRelations[name]; relations != nil {
				db.AddError(preloadEmbedded(preloadDB.Table("").Session(&gorm.Session{Context: db.Statement.Context, SkipHooks: db.Statement.SkipHooks}), relations, db.Statement.Schema, preloadMap[name], db.Statement.Preloads[clause.Associations]))
			} else if rel := preloadDB.Statement.Schema.Relationships.Relations[name]; rel != nil {
				db.AddError(preload(preloadDB.Table("").Session(&gorm.Session{Context: db.Statement.Context, SkipHooks: db.Statement.SkipHooks}), rel, append(db.Statement.Preloads[name], db.Statement.Preloads[clause.Associations]...), preloadMap[name]))
			} else {
				db.AddError(fmt.Errorf("%s: %w for schema %s", name, gorm.ErrUnsupportedRelation, db.Statement.Schema.Name))
			}
		}
	}
}

func AfterQuery(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && db.Statement.Schema.AfterFind && db.RowsAffected > 0 {
		callMethod(db, func(value interface{}, tx *gorm.DB) bool {
			if i, ok := value.(AfterFindInterface); ok {
				db.AddError(i.AfterFind(tx))
				return true
			}
			return false
		})
	}
}
