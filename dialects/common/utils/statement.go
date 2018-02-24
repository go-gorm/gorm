package utils

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/builder"
	"github.com/jinzhu/gorm/model"
)

// GetCreatingAssignments get creating assignments
func GetCreatingAssignments(stmt *builder.Statement, errs *gorm.Errors) chan []model.Field {
	return nil
}

// GetTable get table name
func GetTable(stmt *builder.Statement, errs *gorm.Errors) chan string {
	return nil
}
