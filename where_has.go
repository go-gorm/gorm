package gorm

import (
	"fmt"
	"reflect"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// newWhereHasSession creates a clean *DB session for building EXISTS subqueries,
// preserving the parent's Context and SkipHooks settings.
func newWhereHasSession(db *DB) *DB {
	return db.Session(&Session{
		NewDB:       true,
		Context:     db.Statement.Context,
		SkipHooks:   db.Statement.SkipHooks,
		Initialized: true,
	})
}

// BuildWhereHasClauses processes stored WhereHas/WhereDoesntHave conditions
// and injects the corresponding WHERE EXISTS / WHERE NOT EXISTS subqueries.
// It must be called after the schema has been parsed (e.g., inside BuildQuerySQL).
func (db *DB) BuildWhereHasClauses() {
	if len(db.Statement.whereHasConds) == 0 {
		return
	}

	if db.Statement.Schema == nil {
		_ = db.AddError(fmt.Errorf("%w when using WhereHas", ErrModelValueRequired))
		return
	}

	for _, cond := range db.Statement.whereHasConds {
		if err := buildWhereHasClause(db, db.Statement.Schema, db.Statement.Table, cond); err != nil {
			_ = db.AddError(err)
			return
		}
	}
}

func buildWhereHasClause(db *DB, s *schema.Schema, parentTable string, cond whereHasCond) error {
	rel, ok := s.Relationships.Relations[cond.Association]
	if !ok {
		return fmt.Errorf("relation %q not found in schema %q", cond.Association, s.Name)
	}

	subQuery, err := buildRelationSubquery(db, rel, parentTable, cond.Args)
	if err != nil {
		return err
	}

	keyword := "EXISTS"
	if cond.NotExists {
		keyword = "NOT EXISTS"
	}

	db.Statement.AddClause(clause.Where{
		Exprs: []clause.Expression{
			clause.Expr{SQL: keyword + " (?)", Vars: []interface{}{subQuery}},
		},
	})

	return nil
}

func buildRelationSubquery(db *DB, rel *schema.Relationship, parentTable string, args []interface{}) (*DB, error) {
	switch rel.Type {
	case schema.HasOne, schema.HasMany:
		return buildHasSubquery(db, rel, parentTable, args)
	case schema.BelongsTo:
		return buildBelongsToSubquery(db, rel, parentTable, args)
	case schema.Many2Many:
		return buildMany2ManySubquery(db, rel, parentTable, args)
	default:
		return nil, fmt.Errorf("unsupported relationship type %q for WhereHas", rel.Type)
	}
}

// buildHasSubquery builds: SELECT 1 FROM <related_table> WHERE <fk> = <parent_table>.<pk> [AND polymorphic conditions] [AND user conditions]
func buildHasSubquery(db *DB, rel *schema.Relationship, parentTable string, args []interface{}) (*DB, error) {
	if len(rel.References) == 0 {
		return nil, fmt.Errorf("relation %q has no references", rel.Name)
	}

	relTable := rel.FieldSchema.Table
	subQuery := newWhereHasSession(db).
		Model(reflect.New(rel.FieldSchema.ModelType).Interface()).Select("1")

	for _, ref := range rel.References {
		if ref.OwnPrimaryKey {
			subQuery = subQuery.Where(clause.Eq{
				Column: clause.Column{Table: relTable, Name: ref.ForeignKey.DBName},
				Value:  clause.Column{Table: parentTable, Name: ref.PrimaryKey.DBName},
			})
		} else if ref.PrimaryValue != "" {
			subQuery = subQuery.Where(clause.Eq{
				Column: clause.Column{Table: relTable, Name: ref.ForeignKey.DBName},
				Value:  ref.PrimaryValue,
			})
		}
	}

	applyUserArgs(subQuery, args)
	return subQuery, nil
}

// buildBelongsToSubquery builds: SELECT 1 FROM <related_table> WHERE <pk> = <parent_table>.<fk> [AND user conditions]
func buildBelongsToSubquery(db *DB, rel *schema.Relationship, parentTable string, args []interface{}) (*DB, error) {
	if len(rel.References) == 0 {
		return nil, fmt.Errorf("relation %q has no references", rel.Name)
	}

	relTable := rel.FieldSchema.Table
	subQuery := newWhereHasSession(db).
		Model(reflect.New(rel.FieldSchema.ModelType).Interface()).Select("1")

	for _, ref := range rel.References {
		if ref.PrimaryValue != "" {
			subQuery = subQuery.Where(clause.Eq{
				Column: clause.Column{Table: relTable, Name: ref.ForeignKey.DBName},
				Value:  ref.PrimaryValue,
			})
		} else {
			subQuery = subQuery.Where(clause.Eq{
				Column: clause.Column{Table: relTable, Name: ref.PrimaryKey.DBName},
				Value:  clause.Column{Table: parentTable, Name: ref.ForeignKey.DBName},
			})
		}
	}

	applyUserArgs(subQuery, args)
	return subQuery, nil
}

// buildMany2ManySubquery builds:
//
//	SELECT 1 FROM <related_table>
//	  INNER JOIN <join_table> ON <join_table>.<related_fk> = <related_table>.<related_pk>
//	  WHERE <join_table>.<parent_fk> = <parent_table>.<parent_pk>
//	  [AND polymorphic conditions]
//	  [AND user conditions on related_table]
func buildMany2ManySubquery(db *DB, rel *schema.Relationship, parentTable string, args []interface{}) (*DB, error) {
	joinTable := rel.JoinTable.Table
	relTable := rel.FieldSchema.Table
	subQuery := newWhereHasSession(db).
		Model(reflect.New(rel.FieldSchema.ModelType).Interface()).Select("1")

	var joinON []clause.Expression

	for _, ref := range rel.References {
		switch {
		case ref.OwnPrimaryKey:
			// Parent side: join_table.parent_fk = parent_table.parent_pk  (goes to WHERE)
			subQuery = subQuery.Where(clause.Eq{
				Column: clause.Column{Table: joinTable, Name: ref.ForeignKey.DBName},
				Value:  clause.Column{Table: parentTable, Name: ref.PrimaryKey.DBName},
			})
		case ref.PrimaryValue != "":
			// Polymorphic value on join table
			subQuery = subQuery.Where(clause.Eq{
				Column: clause.Column{Table: joinTable, Name: ref.ForeignKey.DBName},
				Value:  ref.PrimaryValue,
			})
		default:
			// Related side: join_table.related_fk = related_table.related_pk  (goes to JOIN ON)
			joinON = append(joinON, clause.Eq{
				Column: clause.Column{Table: joinTable, Name: ref.ForeignKey.DBName},
				Value:  clause.Column{Table: relTable, Name: ref.PrimaryKey.DBName},
			})
		}
	}

	// INNER JOIN join_table ON join_table.related_fk = related_table.related_pk
	if len(joinON) > 0 {
		subQuery.Statement.AddClause(clause.From{
			Joins: []clause.Join{{
				Type:  clause.InnerJoin,
				Table: clause.Table{Name: joinTable},
				ON:    clause.Where{Exprs: joinON},
			}},
		})
	}

	applyUserArgs(subQuery, args)
	return subQuery, nil
}

// applyUserArgs merges user-provided arguments into the subquery.
// Supported argument types:
//   - *DB: merges WHERE clauses and nested whereHasConds
//   - func(*DB) *DB: scope function applied to the subquery
//   - other: treated as inline conditions passed to Where()
func applyUserArgs(subQuery *DB, args []interface{}) {
	if len(args) == 0 {
		return
	}

	var inlineConds []interface{}
	for _, arg := range args {
		switch v := arg.(type) {
		case *DB:
			mergeDBConditions(subQuery, v)
		case func(*DB) *DB:
			*subQuery = *v(subQuery)
		default:
			inlineConds = append(inlineConds, arg)
		}
	}

	if len(inlineConds) > 0 {
		subQuery.Where(inlineConds[0], inlineConds[1:]...)
	}
}

// mergeDBConditions merges WHERE clauses and whereHasConds from a scope *DB into the subquery.
func mergeDBConditions(subQuery *DB, scopeDB *DB) {
	// Merge WHERE clauses
	if cs, ok := scopeDB.Statement.Clauses["WHERE"]; ok {
		if where, ok := cs.Expression.(clause.Where); ok && len(where.Exprs) > 0 {
			subQuery.Statement.AddClause(clause.Where{Exprs: where.Exprs})
		}
	}

	// Merge WhereHas/WhereDoesntHave conditions
	if len(scopeDB.Statement.whereHasConds) > 0 {
		subQuery.Statement.whereHasConds = append(subQuery.Statement.whereHasConds, scopeDB.Statement.whereHasConds...)
	}
}
