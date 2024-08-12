package callbacks

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
	"strings"
)

func HandleJoins(db *gorm.DB, prejoinCallback func(db *gorm.DB), perFieldNameCallback func(db *gorm.DB, tableAliasName string, idx int, relation *schema.Relationship)) {
	// inline joins
	fromClause := clause.From{}
	if v, ok := db.Statement.Clauses["FROM"].Expression.(clause.From); ok {
		fromClause = v
	}

	if len(db.Statement.Joins) != 0 || len(fromClause.Joins) != 0 {
		prejoinCallback(db)

		specifiedRelationsName := make(map[string]interface{})
		for idx, join := range db.Statement.Joins {
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
						gussNestedRelations := make([]*schema.Relationship, 0, len(nestedJoinNames))
						currentRelations := db.Statement.Schema.Relationships.Relations
						for _, relname := range nestedJoinNames {
							// incomplete match, only treated as raw sql
							if relation, ok = currentRelations[relname]; ok {
								gussNestedRelations = append(gussNestedRelations, relation)
								currentRelations = relation.FieldSchema.Relationships.Relations
							} else {
								isNestedJoin = false
								break
							}
						}

						if isNestedJoin {
							isRelations = true
							relations = gussNestedRelations
						}
					}
				}

				if isRelations {
					genJoinClause := func(joinType clause.JoinType, parentTableName string, relation *schema.Relationship) clause.Join {
						tableAliasName := relation.Name
						if parentTableName != clause.CurrentTable {
							tableAliasName = utils.NestedRelationName(parentTableName, tableAliasName)
						}

						perFieldNameCallback(db, tableAliasName, idx, relation)

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
					for _, rel := range relations {
						// joins table alias like "Manager, Company, Manager__Company"
						nestedAlias := utils.NestedRelationName(parentTableName, rel.Name)
						if _, ok := specifiedRelationsName[nestedAlias]; !ok {
							fromClause.Joins = append(fromClause.Joins, genJoinClause(join.JoinType, parentTableName, rel))
							specifiedRelationsName[nestedAlias] = nil
						}

						if parentTableName != clause.CurrentTable {
							parentTableName = utils.NestedRelationName(parentTableName, rel.Name)
						} else {
							parentTableName = rel.Name
						}
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

}
