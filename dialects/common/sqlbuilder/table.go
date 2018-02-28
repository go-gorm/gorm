package sqlbuilder

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/model"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/inflection"
)

// GetTable get table name for current db operation
func GetTable(tx *gorm.DB) chan string {
	tableChan := make(chan string)

	go func() {
		var tableName string
		if name, ok := tx.Statement.Table.(string); ok {
			tableName = name
		} else {
			for _, v := range []interface{}{tx.Statement.Table, tx.Statement.Dest} {
				if v != nil {
					if t, ok := v.(tabler); ok {
						tableName = t.TableName()
					} else if t, ok := v.(dbTabler); ok {
						tableName = t.TableName(tx)
					} else if s := schema.Parse(v); s != nil {
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
		}

		if tableName != "" {
			if model.DefaultTableNameHandler != nil {
				tableChan <- model.DefaultTableNameHandler(tx, tableName)
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
