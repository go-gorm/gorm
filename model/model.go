package model

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/builder"
	"github.com/jinzhu/gorm/schema"
)

// DefaultTableNameHandler default table name handler
var DefaultTableNameHandler = func(stmt *builder.Statement, tableName string) string {
	return tableName
}

// GetCreatingAssignments get creating assignments
func GetCreatingAssignments(stmt *builder.Statement, errs *gorm.Errors) chan []schema.Field {
	return nil
}

// GetTable get table name
func GetTable(stmt *builder.Statement, errs *gorm.Errors) chan string {
	tableChan := make(chan string)

	go func() {
		if stmt.Table != nil {
			if table, ok := stmt.Table.(string); ok {
				tableChan <- DefaultTableNameHandler(stmt, table)
			} else if tableSchema := schema.Parse(stmt.Table); tableSchema != nil {
				if tableSchema.TableName != "" {
					tableChan <- DefaultTableNameHandler(stmt, tableSchema.TableName)
				}
				tableSchema.ModelType.Name
			}
		}
	}()

	return tableChan
}

// if scope.Value == nil {
// 	return &modelStruct
// }
// TableName get model's table name
// func (schema *Schema) TableName(stmt *builder.Statement) string {
// 	if s.defaultTableName == "" && db != nil && s.ModelType != nil {
// 		// Set default table name
// 		if tabler, ok := reflect.New(s.ModelType).Interface().(tabler); ok {
// 			s.defaultTableName = tabler.TableName()
// 		} else {
// 			tableName := ToDBName(s.ModelType.Name())
// 			if db == nil || !db.parent.singularTable {
// 				tableName = inflection.Plural(tableName)
// 			}
// 			s.defaultTableName = tableName
// 		}
// 	}
// 	return DefaultTableNameHandler(db, s.defaultTableName)
// }
