package callbacks

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func whereHasDb(db *gorm.DB) *gorm.DB {
	tx := db.Session(&gorm.Session{Context: db.Statement.Context, NewDB: true, SkipHooks: db.Statement.SkipHooks, Initialized: true})

	return tx
}

var relationHandlers = map[schema.RelationshipType]func(*gorm.DB, *gorm.DB, *schema.Relationship, []interface{}) (*gorm.DB, error){
	schema.Many2Many: existsMany2many,
	schema.BelongsTo: existsBelongsTo,
	schema.HasMany:   existsHasMany,
	schema.HasOne:    existsHasOne,
}

func newWhereHas(db *gorm.DB, isDoesntHave bool, relationName string, conds []interface{}, s *schema.Schema) (*clause.Where, error) {
	var err error

	rel, ok := s.Relationships.Relations[relationName]
	if !ok {
		return nil, fmt.Errorf("relation %s not found", relationName)
	}

	tx := whereHasDb(db)

	reflectResults := rel.FieldSchema.MakeSlice().Elem()

	tx = tx.Model(reflectResults.Addr().Interface()).Select("id")
	if err = tx.Statement.Parse(tx.Statement.Model); err != nil {
		return nil, err
	}

	handler, ok := relationHandlers[rel.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported relation type: %v", rel.Type)
	}

	tx, err = handler(db, tx, rel, conds)
	if err != nil {
		return nil, err
	}

	cond := "EXISTS(?)"
	if isDoesntHave {
		cond = "NOT " + cond
	}

	cl := clause.Where{
		Exprs: []clause.Expression{
			clause.Expr{
				SQL:                cond,
				Vars:               []interface{}{tx},
				WithoutParentheses: false,
			},
		},
	}

	return &cl, nil
}

func existsHasOne(mainQuery *gorm.DB, existsQuery *gorm.DB, rel *schema.Relationship, conds []interface{}) (*gorm.DB, error) {
	return existsHasMany(mainQuery, existsQuery, rel, conds)
}

func existsHasMany(mainQuery *gorm.DB, existsQuery *gorm.DB, rel *schema.Relationship, conds []interface{}) (*gorm.DB, error) {
	if len(rel.References) < 1 {
		return nil, fmt.Errorf("relation %s has no references", rel.Name)
	}

	for _, reference := range rel.References {
		if reference.PrimaryKey != nil {
			existsQuery.Statement.AddClause(clause.Where{
				Exprs: []clause.Expression{
					clause.Eq{
						Column: clause.Column{Table: existsQuery.Statement.Table, Name: reference.ForeignKey.DBName},
						Value:  clause.Column{Table: mainQuery.Statement.Table, Name: reference.PrimaryKey.DBName},
					},
				},
			})
		} else {
			existsQuery.Statement.AddClause(clause.Where{
				Exprs: []clause.Expression{
					clause.Eq{
						Column: clause.Column{Table: existsQuery.Statement.Table, Name: reference.ForeignKey.DBName},
						Value:  reference.PrimaryValue,
					},
				},
			})
		}
	}

	existsQuery = applyConds(existsQuery, conds)

	return existsQuery, nil
}

func existsBelongsTo(mainQuery *gorm.DB, existsQuery *gorm.DB, rel *schema.Relationship, conds []interface{}) (*gorm.DB, error) {
	if len(rel.References) < 1 {
		return nil, fmt.Errorf("relation %s has no references", rel.Name)
	}

	for _, reference := range rel.References {
		existsQuery.Statement.AddClause(clause.Where{
			Exprs: []clause.Expression{
				clause.Eq{
					Column: clause.Column{Table: existsQuery.Statement.Table, Name: reference.PrimaryKey.DBName},
					Value:  clause.Column{Table: mainQuery.Statement.Table, Name: reference.ForeignKey.DBName},
				},
			},
		})
	}

	existsQuery = applyConds(existsQuery, conds)

	return existsQuery, nil
}

func existsMany2many(mainQuery *gorm.DB, existsQuery *gorm.DB, rel *schema.Relationship, conds []interface{}) (*gorm.DB, error) {
	if rel.JoinTable != nil {
		var parentTableField *schema.Reference = nil
		var primaryTableField *schema.Reference = nil

		for _, reference := range rel.References {
			if !reference.OwnPrimaryKey {
				parentTableField = reference
			} else {
				primaryTableField = reference
			}
		}

		if parentTableField == nil {
			return nil, fmt.Errorf("relation %s has no parent table field", rel.Name)
		}

		if primaryTableField == nil {
			return nil, fmt.Errorf("relation %s has no primary table field", rel.Name)
		}

		fromClause := clause.From{
			Tables: nil,
			Joins: []clause.Join{
				{
					Type:  clause.InnerJoin,
					Table: clause.Table{Name: rel.JoinTable.Table},
					ON: clause.Where{
						Exprs: []clause.Expression{
							clause.Eq{
								Column: clause.Column{Table: rel.JoinTable.Table, Name: parentTableField.ForeignKey.DBName},
								Value:  clause.Column{Table: existsQuery.Statement.Table, Name: parentTableField.PrimaryKey.DBName},
							},
						},
					},
				},
			},
		}

		existsQuery.Statement.AddClause(fromClause)

		existsQuery.Statement.AddClause(clause.Where{
			Exprs: []clause.Expression{
				clause.Eq{
					Column: clause.Column{Table: rel.JoinTable.Table, Name: primaryTableField.ForeignKey.DBName},
					Value:  clause.Column{Table: mainQuery.Statement.Table, Name: primaryTableField.PrimaryKey.DBName},
				},
			},
		})

		existsQuery = applyConds(existsQuery, conds)
	}

	return existsQuery, nil
}

func applyConds(existsQuery *gorm.DB, conds []interface{}) *gorm.DB {
	inlineConds := make([]interface{}, 0)

	for _, cond := range conds {
		if fc, ok := cond.(func(*gorm.DB) *gorm.DB); ok {
			existsQuery = fc(existsQuery)
		} else {
			inlineConds = append(inlineConds, cond)
		}
	}

	if len(inlineConds) > 0 {
		existsQuery = existsQuery.Where(inlineConds[0], inlineConds[1:]...)
	}

	return existsQuery
}
