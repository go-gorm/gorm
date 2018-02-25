package utils

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/builder"
	"github.com/jinzhu/gorm/model"
)

// DefaultTableNameHandler default table name handler
var DefaultTableNameHandler = func(stmt *builder.Statement, tableName string) string {
	return tableName
}

// GetCreatingAssignments get creating assignments
func GetCreatingAssignments(stmt *builder.Statement, errs *gorm.Errors) chan []model.Field {
	return nil
}

// GetTable get table name
func GetTable(stmt *builder.Statement, errs *gorm.Errors) chan string {
	return nil
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
