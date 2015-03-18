package gorm

import (
	"fmt"
	"strings"
)

type JoinTableHandlerInterface interface {
	Table(db *DB) string
	Add(db *DB, source1 interface{}, source2 interface{}) error
	Delete(db *DB, sources ...interface{}) error
	JoinWith(db *DB, source interface{}) *DB
}

type JoinTableSource struct {
	ForeignKey       string
	ForeignKeyPrefix string
	ModelStruct
}

type JoinTableHandler struct {
	TableName string
	Source1   JoinTableSource
	Source2   JoinTableSource
}

func (jt JoinTableHandler) Table(*DB) string {
	return jt.TableName
}

func (jt JoinTableHandler) GetValueMap(db *DB, sources ...interface{}) map[string]interface{} {
	values := map[string]interface{}{}
	for _, source := range sources {
		scope := db.NewScope(source)
		for _, primaryField := range scope.GetModelStruct().PrimaryFields {
			if field, ok := scope.Fields()[primaryField.DBName]; ok {
				values[primaryField.DBName] = field.Field.Interface()
			}
		}
	}
	return values
}

func (jt JoinTableHandler) Add(db *DB, source1 interface{}, source2 interface{}) error {
	scope := db.NewScope("")
	valueMap := jt.GetValueMap(db, source1, source2)

	var setColumns, setBinVars, queryConditions []string
	var values []interface{}
	for key, value := range valueMap {
		setColumns = append(setColumns, key)
		setBinVars = append(setBinVars, `?`)
		queryConditions = append(queryConditions, fmt.Sprintf("%v = ?", scope.Quote(key)))
		values = append(values, value)
	}

	for _, value := range valueMap {
		values = append(values, value)
	}

	quotedTable := jt.Table(db)
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) SELECT %v %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v);",
		quotedTable,
		strings.Join(setColumns, ","),
		strings.Join(setBinVars, ","),
		scope.Dialect().SelectFromDummyTable(),
		quotedTable,
		strings.Join(queryConditions, " AND "),
	)

	return db.Exec(sql, values...).Error
}

func (jt JoinTableHandler) Delete(db *DB, sources ...interface{}) error {
	// return db.Table(jt.Table(db)).Delete("").Error
	return nil
}

func (jt JoinTableHandler) JoinWith(db *DB, sources interface{}) *DB {
	return db
}
