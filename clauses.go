package gorm

import (
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

func CreateUpdateClause(stmt *Statement) {
	updateClause := clause.Update{}
	if v, ok := stmt.Clauses["UPDATE"].Expression.(clause.Update); ok {
		updateClause = v
	}

	if len(stmt.Joins) != 0 || len(updateClause.Joins) != 0 {
		updateClause.Joins = append(updateClause.Joins, GenJoinClauses(stmt.DB, &clause.Select{})...)
		stmt.AddClause(updateClause)
	} else {
		stmt.AddClauseIfNotExists(clause.Update{})
	}
}

func GenJoinClauses(db *DB, clauseSelect *clause.Select) []clause.Join {
	joinClauses := []clause.Join{}

	if len(db.Statement.Selects) == 0 && len(db.Statement.Omits) == 0 && db.Statement.Schema != nil {
		clauseSelect.Columns = make([]clause.Column, len(db.Statement.Schema.DBNames))
		for idx, dbName := range db.Statement.Schema.DBNames {
			clauseSelect.Columns[idx] = clause.Column{Table: db.Statement.Table, Name: dbName}
		}
	}

	specifiedRelationsName := make(map[string]interface{})
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

					columnStmt := Statement{
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
						onStmt := Statement{Table: tableAliasName, DB: db, Clauses: map[string]clause.Clause{}}
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
						joinClauses = append(joinClauses, genJoinClause(join.JoinType, parentTableName, rel))
						specifiedRelationsName[nestedAlias] = nil
					}

					if parentTableName != clause.CurrentTable {
						parentTableName = utils.NestedRelationName(parentTableName, rel.Name)
					} else {
						parentTableName = rel.Name
					}
				}
			} else {
				joinClauses = append(joinClauses, clause.Join{
					Expression: clause.NamedExpr{SQL: join.Name, Vars: join.Conds},
				})
			}
		} else {
			joinClauses = append(joinClauses, clause.Join{
				Expression: clause.NamedExpr{SQL: join.Name, Vars: join.Conds},
			})
		}
	}

	db.Statement.Joins = nil

	return joinClauses
}
