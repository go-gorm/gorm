package gorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type JoinTableHandlerInterface interface {
	Setup(relationship *Relationship, tableName string, source reflect.Type, destination reflect.Type)
	Table(db *DB) string
	Add(handler JoinTableHandlerInterface, db *DB, source interface{}, destination interface{}) error
	Delete(handler JoinTableHandlerInterface, db *DB, sources ...interface{}) error
	JoinWith(handler JoinTableHandlerInterface, db *DB, source interface{}) *DB
	SourceForeignKeys() []JoinTableForeignKey
	DestinationForeignKeys() []JoinTableForeignKey
}

type JoinTableForeignKey struct {
	DBName            string
	AssociationDBName string
}

type JoinTableSource struct {
	ModelType   reflect.Type
	ForeignKeys []JoinTableForeignKey
}

type JoinTableHandler struct {
	TableName   string          `sql:"-"`
	Source      JoinTableSource `sql:"-"`
	Destination JoinTableSource `sql:"-"`
}

func (s *JoinTableHandler) SourceForeignKeys() []JoinTableForeignKey {
	return s.Source.ForeignKeys
}

func (s *JoinTableHandler) DestinationForeignKeys() []JoinTableForeignKey {
	return s.Destination.ForeignKeys
}

func (s *JoinTableHandler) Setup(relationship *Relationship, tableName string, source reflect.Type, destination reflect.Type) {
	s.TableName = tableName

	s.Source = JoinTableSource{ModelType: source}
	sourceScope := &Scope{Value: reflect.New(source).Interface()}
	sourcePrimaryFields := sourceScope.GetModelStruct().PrimaryFields
	for _, primaryField := range sourcePrimaryFields {
		if relationship.ForeignDBName == "" {
			relationship.ForeignFieldName = source.Name() + primaryField.Name
			relationship.ForeignDBName = ToDBName(relationship.ForeignFieldName)
		}

		var dbName string
		if len(sourcePrimaryFields) == 1 || primaryField.DBName == "id" {
			dbName = relationship.ForeignDBName
		} else {
			dbName = ToDBName(source.Name() + primaryField.Name)
		}

		s.Source.ForeignKeys = append(s.Source.ForeignKeys, JoinTableForeignKey{
			DBName:            dbName,
			AssociationDBName: primaryField.DBName,
		})
	}

	s.Destination = JoinTableSource{ModelType: destination}
	destinationScope := &Scope{Value: reflect.New(destination).Interface()}
	destinationPrimaryFields := destinationScope.GetModelStruct().PrimaryFields
	for _, primaryField := range destinationPrimaryFields {
		var dbName string
		if len(sourcePrimaryFields) == 1 || primaryField.DBName == "id" {
			dbName = relationship.AssociationForeignDBName
		} else {
			dbName = ToDBName(destinationScope.GetModelStruct().ModelType.Name() + primaryField.Name)
		}

		s.Destination.ForeignKeys = append(s.Destination.ForeignKeys, JoinTableForeignKey{
			DBName:            dbName,
			AssociationDBName: primaryField.DBName,
		})
	}
}

func (s JoinTableHandler) Table(db *DB) string {
	return s.TableName
}

func (s JoinTableHandler) GetSearchMap(db *DB, sources ...interface{}) map[string]interface{} {
	values := map[string]interface{}{}

	for _, source := range sources {
		scope := db.NewScope(source)
		modelType := scope.GetModelStruct().ModelType

		if s.Source.ModelType == modelType {
			for _, foreignKey := range s.Source.ForeignKeys {
				values[foreignKey.DBName] = scope.Fields()[foreignKey.AssociationDBName].Field.Interface()
			}
		} else if s.Destination.ModelType == modelType {
			for _, foreignKey := range s.Destination.ForeignKeys {
				values[foreignKey.DBName] = scope.Fields()[foreignKey.AssociationDBName].Field.Interface()
			}
		}
	}
	return values
}

func (s JoinTableHandler) Add(handler JoinTableHandlerInterface, db *DB, source1 interface{}, source2 interface{}) error {
	scope := db.NewScope("")
	searchMap := s.GetSearchMap(db, source1, source2)

	var assignColumns, binVars, conditions []string
	var values []interface{}
	for key, value := range searchMap {
		assignColumns = append(assignColumns, key)
		binVars = append(binVars, `?`)
		conditions = append(conditions, fmt.Sprintf("%v = ?", scope.Quote(key)))
		values = append(values, value)
	}

	for _, value := range values {
		values = append(values, value)
	}

	quotedTable := handler.Table(db)
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) SELECT %v %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v)",
		quotedTable,
		strings.Join(assignColumns, ","),
		strings.Join(binVars, ","),
		scope.Dialect().SelectFromDummyTable(),
		quotedTable,
		strings.Join(conditions, " AND "),
	)

	return db.Exec(sql, values...).Error
}

func (s JoinTableHandler) Delete(handler JoinTableHandlerInterface, db *DB, sources ...interface{}) error {
	var conditions []string
	var values []interface{}

	for key, value := range s.GetSearchMap(db, sources...) {
		conditions = append(conditions, fmt.Sprintf("%v = ?", key))
		values = append(values, value)
	}

	return db.Table(handler.Table(db)).Where(strings.Join(conditions, " AND "), values...).Delete("").Error
}

func (s JoinTableHandler) JoinWith(handler JoinTableHandlerInterface, db *DB, source interface{}) *DB {
	quotedTable := handler.Table(db)

	scope := db.NewScope(source)
	modelType := scope.GetModelStruct().ModelType
	var joinConditions []string
	var queryConditions []string
	var values []interface{}
	if s.Source.ModelType == modelType {
		for _, foreignKey := range s.Destination.ForeignKeys {
			destinationTableName := db.NewScope(reflect.New(s.Destination.ModelType).Interface()).QuotedTableName()
			joinConditions = append(joinConditions, fmt.Sprintf("%v.%v = %v.%v", quotedTable, scope.Quote(foreignKey.DBName), destinationTableName, scope.Quote(foreignKey.AssociationDBName)))
		}

		for _, foreignKey := range s.Source.ForeignKeys {
			queryConditions = append(queryConditions, fmt.Sprintf("%v.%v = ?", quotedTable, scope.Quote(foreignKey.DBName)))
			values = append(values, scope.Fields()[foreignKey.AssociationDBName].Field.Interface())
		}
		return db.Joins(fmt.Sprintf("INNER JOIN %v ON %v", quotedTable, strings.Join(joinConditions, " AND "))).
			Where(strings.Join(queryConditions, " AND "), values...)
	} else {
		db.Error = errors.New("wrong source type for join table handler")
		return db
	}
}
