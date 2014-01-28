package gorm

import (
	"fmt"
	"strings"
)

func (scope *Scope) createTable() *Scope {
	var sqls []string
	for _, field := range scope.Fields() {
		if !field.IsIgnored && len(field.SqlTag) > 0 {
			sqls = append(sqls, scope.quote(field.DBName)+" "+field.SqlTag)
		}
	}
	scope.Raw(fmt.Sprintf("CREATE TABLE %v (%v)", scope.TableName(), strings.Join(sqls, ","))).Exec()
	return scope
}

func (scope *Scope) dropTable() *Scope {
	scope.Raw(fmt.Sprintf("DROP TABLE %v", scope.TableName())).Exec()
	return scope
}

func (scope *Scope) modifyColumn(column string, typ string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v MODIFY %v %v", scope.TableName(), scope.quote(column), typ)).Exec()
}

func (scope *Scope) dropColumn(column string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v DROP COLUMN %v", scope.TableName(), scope.quote(column))).Exec()
}

func (scope *Scope) addIndex(column string, names ...string) {
	var indexName string
	if len(names) > 0 {
		indexName = names[0]
	} else {
		indexName = fmt.Sprintf("index_%v_on_%v", scope.TableName(), column)
	}

	scope.Raw(fmt.Sprintf("CREATE INDEX %v ON %v(%v);", indexName, scope.TableName(), scope.quote(column))).Exec()
}

func (scope *Scope) removeIndex(indexName string) {
	scope.Raw(fmt.Sprintf("DROP INDEX %v ON %v", indexName, scope.TableName())).Exec()
}

func (scope *Scope) autoMigrate() *Scope {
	var tableName string
	scope.Raw(fmt.Sprintf("SELECT table_name FROM INFORMATION_SCHEMA.tables where table_name = %v", scope.AddToVars(scope.TableName())))
	scope.DB().QueryRow(scope.Sql, scope.SqlVars...).Scan(&tableName)
	scope.SqlVars = []interface{}{}

	// If table doesn't exist
	if len(tableName) == 0 {
		scope.createTable()
	} else {
		for _, field := range scope.Fields() {
			var column, data string
			scope.Raw(fmt.Sprintf("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = %v and column_name = %v",
				scope.AddToVars(scope.TableName()),
				scope.AddToVars(field.DBName),
			))
			scope.DB().QueryRow(scope.Sql, scope.SqlVars...).Scan(&column, &data)
			scope.SqlVars = []interface{}{}

			// If column doesn't exist
			if len(column) == 0 && len(field.SqlTag) > 0 && !field.IsIgnored {
				scope.Raw(fmt.Sprintf("ALTER TABLE %v ADD %v %v;", scope.TableName(), field.DBName, field.SqlTag)).Exec()
			}
		}
	}
	return scope
}
