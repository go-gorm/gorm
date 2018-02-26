package model

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/inflection"
)

// DefaultTableNameHandler default table name handler
//    DefaultTableNameHandler = func(tx *gorm.DB, tableName string) string {
//    	return tableName
//    }
var DefaultTableNameHandler func(tx *gorm.DB, tableName string) string

// GetCreatingAssignments get creating assignments
func GetCreatingAssignments(tx *gorm.DB) chan []schema.Field {
	return nil
}

// GetTable get table name for current db operation
func GetTable(tx *gorm.DB) chan string {
	tableChan := make(chan string)

	go func() {
		var tableName string
		if name, ok := tx.Statement.Table.(string); ok {
			tableName = name
		} else {
			for _, v := range []interface{}{tx.Statement.Table, tx.Statement.Dest} {
				if t, ok := v.(tabler); ok {
					tableName = t.TableName()
				} else if t, ok := v.(dbTabler); ok {
					tableName = t.TableName(tx)
				} else if s := schema.Parse(tx.Statement.Table); s != nil {
					if s.TableName != "" {
						tableName = s.TableName
					} else {
						tableName = schema.ToDBName(s.ModelType.Name())
						if !tx.Config.SingularTable {
							tableName = inflection.Plural(tableName)
						}
					}
				}

				if tableName != "" {
					break
				}
			}
		}

		if tableName != "" {
			if DefaultTableNameHandler != nil {
				tableChan <- DefaultTableNameHandler(tx, tableName)
			} else {
				tableChan <- tableName
			}
		} else {
			tx.AddError(ErrInvalidTable)
		}
	}()

	return tableChan
}

type tabler interface {
	TableName() string
}

type dbTabler interface {
	TableName(*gorm.DB) string
}
