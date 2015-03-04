package gorm

import (
	"fmt"
	"strings"
)

type JoinTableHandler interface {
	Table(*DB, *Relationship) string
	Add(*DB, *Relationship, interface{}, interface{}) error
	Delete(*DB, *Relationship) error
	Scope(*DB, *Relationship) *DB
}

type defaultJoinTableHandler struct{}

func (s *defaultJoinTableHandler) Table(db *DB, relationship *Relationship) string {
	return relationship.JoinTable
}

func (s *defaultJoinTableHandler) Add(db *DB, relationship *Relationship, foreignValue interface{}, associationValue interface{}) error {
	scope := db.NewScope("")
	quotedForeignDBName := scope.Quote(relationship.ForeignDBName)
	quotedAssociationDBName := scope.Quote(relationship.AssociationForeignDBName)
	table := s.Table(db, relationship)

	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) SELECT ?,? %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v = ? AND %v = ?);",
		scope.Quote(table),
		strings.Join([]string{quotedForeignDBName, quotedAssociationDBName}, ","),
		scope.Dialect().SelectFromDummyTable(),
		scope.Quote(table),
		quotedForeignDBName,
		quotedAssociationDBName,
	)

	return db.Exec(sql, foreignValue, associationValue, foreignValue, associationValue).Error
}

func (s *defaultJoinTableHandler) Delete(db *DB, relationship *Relationship) error {
	return db.Table(s.Table(db, relationship)).Delete("").Error
}

func (s *defaultJoinTableHandler) Scope(db *DB, relationship *Relationship) *DB {
	return db.Table(s.Table(db, relationship))
}

var DefaultJoinTableHandler = &defaultJoinTableHandler{}
