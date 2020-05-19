package gorm

import (
	"fmt"

	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
)

// Association Mode contains some helper methods to handle relationship things easily.
type Association struct {
	DB           *DB
	Relationship *schema.Relationship
	Error        error
}

func (db *DB) Association(column string) *Association {
	association := &Association{DB: db}

	if err := db.Statement.Parse(db.Statement.Model); err == nil {
		association.Relationship = db.Statement.Schema.Relationships.Relations[column]

		if association.Relationship == nil {
			association.Error = fmt.Errorf("%w: %v", ErrUnsupportedRelation, column)
		}
	} else {
		association.Error = err
	}

	return association
}

func (association *Association) Find(out interface{}, conds ...interface{}) error {
	if association.Error == nil {
	}

	return association.Error
}

func (association *Association) Append(values ...interface{}) error {
	return association.Error
}

func (association *Association) Replace(values ...interface{}) error {
	return association.Error
}

func (association *Association) Delete(values ...interface{}) error {
	return association.Error
}

func (association *Association) Clear() error {
	return association.Replace()
}

func (association *Association) Count() (count int) {
	if association.Error == nil {
		var (
			tx    = association.DB
			conds = association.Relationship.ToQueryConditions(tx.Statement.ReflectValue)
		)

		if association.Relationship.JoinTable != nil {
			tx.Clauses(clause.From{Joins: []clause.Join{{
				Table: clause.Table{Name: association.Relationship.JoinTable.Table},
				ON:    clause.Where{Exprs: conds},
			}}})
		} else {
			tx.Clauses(clause.Where{Exprs: conds})
		}

		association.Error = tx.Count(&count).Error
	}

	return
}
