package gorm

import (
	"fmt"
	"strings"
)

type JoinTableHandler interface {
	Add(*DB, *Relationship, interface{}, interface{}) error
	Delete(*DB, *Relationship) error
	Scope(*DB, *Relationship) *DB
}

type defaultJoinTableHandler struct{}

func (*defaultJoinTableHandler) Add(db *DB, relationship *Relationship, foreignValue interface{}, associationValue interface{}) error {
	scope := db.NewScope("")
	quotedForeignDBName := scope.Quote(relationship.ForeignDBName)
	quotedAssociationDBName := scope.Quote(relationship.AssociationForeignDBName)

	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) SELECT ?,? %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v = ? AND %v = ?);",
		scope.Quote(relationship.JoinTable),
		strings.Join([]string{quotedForeignDBName, quotedAssociationDBName}, ","),
		scope.Dialect().SelectFromDummyTable(),
		scope.Quote(relationship.JoinTable),
		quotedForeignDBName,
		quotedAssociationDBName,
	)

	return db.Exec(sql, foreignValue, associationValue, foreignValue, associationValue).Error
}

func (*defaultJoinTableHandler) Delete(db *DB, relationship *Relationship) error {
	return db.Delete("").Error
}

func (*defaultJoinTableHandler) Scope(db *DB, relationship *Relationship) *DB {
	return db
}

var DefaultJoinTableHandler = &defaultJoinTableHandler{}
