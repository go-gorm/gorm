package callbacks

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// Query builds SELECT SQL and, unless DryRun or an earlier error prevents execution,
// queries the current connection with the statement context and scans rows into Dest.
// It owns and closes the result set and updates result counters. Preloading and
// AfterFind run in subsequent callbacks, not in this function.
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

			if db.Statement.Result != nil {
				db.Statement.Result.RowsAffected = db.RowsAffected
			}
		}
	}
}

// BuildQuerySQL adds schema query clauses and builds SELECT SQL only when the statement
// SQL buffer is empty. It derives projections from Select/Omit, QueryFields,
// destination shape, and joined relations; nonzero destination primary keys can add
// WHERE predicates.
//
// Relationship joins derive key predicates and nested column aliases for scanning,
// include related query clauses such as soft-delete filters, and merge supplied ON
// conditions. A join name that is not a complete recognized relationship path is
// treated as raw SQL. Raw expressions must be trusted; bind data through Vars. The
// function mutates statement clauses and SQL without executing a query.
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
			if len(db.Statement.Selects) == 0 && len(db.Statement.Omits) == 0 && db.Statement.Schema != nil {
				clauseSelect.Columns = make([]clause.Column, len(db.Statement.Schema.DBNames))
				for idx, dbName := range db.Statement.Schema.DBNames {
					clauseSelect.Columns[idx] = clause.Column{Table: db.Statement.Table, Name: dbName}
				}
			}

			specifiedRelationsName := map[string]string{clause.CurrentTable: clause.CurrentTable}
			for _, join := range db.Statement.Joins {
				if db.Statement.Schema != nil {
					var isRelations bool // is relations or raw sql
					var relations []*schema.Relationship
					relation, ok := db.Statement.Schema.Relationships.Relations[join.Name]
					if ok {
						isRelations = true
						relations = append(relations, relation)
					} else {
						// handle nested join like "Manager.Company"
						nestedJoinNames := strings.Split(join.Name, ".")
						if len(nestedJoinNames) > 1 {
							isNestedJoin := true
							guessNestedRelations := make([]*schema.Relationship, 0, len(nestedJoinNames))
							currentRelations := db.Statement.Schema.Relationships.Relations
							for _, relname := range nestedJoinNames {
								// incomplete match, only treated as raw sql
								if relation, ok = currentRelations[relname]; ok {
									guessNestedRelations = append(guessNestedRelations, relation)
									currentRelations = relation.FieldSchema.Relationships.Relations
								} else {
									isNestedJoin = false
									break
								}
							}

							if isNestedJoin {
								isRelations = true
								relations = guessNestedRelations
							}
						}
					}

					if isRelations {
						genJoinClause := func(joinType clause.JoinType, tableAliasName string, parentTableName string, relation *schema.Relationship) clause.Join {
							columnStmt := gorm.Statement{
								Table: tableAliasName, DB: db, Schema: relation.FieldSchema,
								Selects: join.Selects, Omits: join.Omits,
							}

							selectColumns, restricted := columnStmt.SelectAndOmitColumns(false, false)
							for _, s := range relation.FieldSchema.DBNames {
								if v, ok := selectColumns[s]; (ok && v) || (!ok && !restricted) {
									clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{
										Table: tableAliasName,
										Name:  s,
										Alias: utils.NestedRelationName(tableAliasName, s),
									})
								}
							}

							if join.Expression != nil {
								return clause.Join{
									Type:       join.JoinType,
									Expression: join.Expression,
								}
							}

							exprs := make([]clause.Expression, len(relation.References))
							for idx, ref := range relation.References {
								if ref.OwnPrimaryKey {
									exprs[idx] = clause.Eq{
										Column: clause.Column{Table: parentTableName, Name: ref.PrimaryKey.DBName},
										Value:  clause.Column{Table: tableAliasName, Name: ref.ForeignKey.DBName},
									}
								} else {
									if ref.PrimaryValue == "" {
										exprs[idx] = clause.Eq{
											Column: clause.Column{Table: parentTableName, Name: ref.ForeignKey.DBName},
											Value:  clause.Column{Table: tableAliasName, Name: ref.PrimaryKey.DBName},
										}
									} else {
										exprs[idx] = clause.Eq{
											Column: clause.Column{Table: tableAliasName, Name: ref.ForeignKey.DBName},
											Value:  ref.PrimaryValue,
										}
									}
								}
							}

							{
								onStmt := gorm.Statement{Table: tableAliasName, DB: db, Clauses: map[string]clause.Clause{}}
								for _, c := range relation.FieldSchema.QueryClauses {
									onStmt.AddClause(c)
								}

								if join.On != nil {
									onStmt.AddClause(join.On)
								}

								if cs, ok := onStmt.Clauses["WHERE"]; ok {
									if where, ok := cs.Expression.(clause.Where); ok {
										where.Build(&onStmt)

										if onSQL := onStmt.SQL.String(); onSQL != "" {
											vars := onStmt.Vars
											for idx, v := range vars {
												bindvar := strings.Builder{}
												onStmt.Vars = vars[0 : idx+1]
												db.Dialector.BindVarTo(&bindvar, &onStmt, v)
												onSQL = strings.Replace(onSQL, bindvar.String(), "?", 1)
											}

											exprs = append(exprs, clause.Expr{SQL: onSQL, Vars: vars})
										}
									}
								}
							}

							return clause.Join{
								Type:  joinType,
								Table: clause.Table{Name: relation.FieldSchema.Table, Alias: tableAliasName},
								ON:    clause.Where{Exprs: exprs},
							}
						}

						parentTableName := clause.CurrentTable
						for idx, rel := range relations {
							// joins table alias like "Manager, Company, Manager__Company"
							curAliasName := rel.Name
							if parentTableName != clause.CurrentTable {
								curAliasName = utils.NestedRelationName(parentTableName, curAliasName)
							}

							if _, ok := specifiedRelationsName[curAliasName]; !ok {
								aliasName := curAliasName
								if idx == len(relations)-1 && join.Alias != "" {
									aliasName = join.Alias
								}

								fromClause.Joins = append(fromClause.Joins, genJoinClause(join.JoinType, aliasName, specifiedRelationsName[parentTableName], rel))
								specifiedRelationsName[curAliasName] = aliasName
							}

							parentTableName = curAliasName
						}
					} else {
						fromClause.Joins = append(fromClause.Joins, clause.Join{
							Expression: clause.NamedExpr{SQL: join.Name, Vars: join.Conds},
						})
					}
				} else {
					fromClause.Joins = append(fromClause.Joins, clause.Join{
						Expression: clause.NamedExpr{SQL: join.Name, Vars: join.Conds},
					})
				}
			}

			db.Statement.AddClause(fromClause)
		} else {
			db.Statement.AddClauseIfNotExists(clause.From{})
		}

		db.Statement.AddClauseIfNotExists(clauseSelect)

		db.Statement.Build(db.Statement.BuildClauses...)
	}
}

// Preload resolves requested association paths after the parent query has populated
// Dest. It requires a parsed model schema, skips work after an error, and records
// preload errors on db. Joined relationships are traversed using already loaded values;
// other relationships issue separate queries with the operation's context, hook policy,
// and Unscoped setting. Preload does not by itself give the parent and association
// queries a transactional snapshot.
func Preload(db *gorm.DB) {
	if db.Error == nil && len(db.Statement.Preloads) > 0 {
		if db.Statement.Schema == nil {
			db.AddError(fmt.Errorf("%w when using preload", gorm.ErrModelValueRequired))
			return
		}

		joins := make([]string, 0, len(db.Statement.Joins))
		for _, join := range db.Statement.Joins {
			joins = append(joins, join.Name)
		}

		tx := preloadDB(db, db.Statement.ReflectValue, db.Statement.Dest)
		if tx.Error != nil {
			return
		}

		db.AddError(preloadEntryPoint(tx, joins, &tx.Statement.Schema.Relationships, db.Statement.Preloads, db.Statement.Preloads[clause.Associations]))
	}
}

// AfterQuery removes joins appended during query construction from the stored FROM
// clause, then invokes AfterFind on returned models when RowsAffected is positive,
// schema metadata enables it, no error exists, and SkipHooks is false. It follows
// preloading; hook errors are added to db.
func AfterQuery(db *gorm.DB) {
	// clear the joins after query because preload need it
	if v, ok := db.Statement.Clauses["FROM"].Expression.(clause.From); ok {
		fromClause := db.Statement.Clauses["FROM"]
		fromClause.Expression = clause.From{Tables: v.Tables, Joins: utils.RTrimSlice(v.Joins, len(db.Statement.Joins))} // keep the original From Joins
		db.Statement.Clauses["FROM"] = fromClause
	}
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
