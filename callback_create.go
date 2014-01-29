package gorm

import (
	"fmt"
	"strings"
	"time"
)

func BeforeCreate(scope *Scope) {
	scope.CallMethod("BeforeSave")
	scope.CallMethod("BeforeCreate")
}

func UpdateTimeStampWhenCreate(scope *Scope) {
	if !scope.HasError() {
		scope.SetColumn("CreatedAt", time.Now())
		scope.SetColumn("UpdatedAt", time.Now())
	}
}

func Create(scope *Scope) {
	defer scope.Trace(time.Now())

	if !scope.HasError() {
		// set create sql
		var sqls, columns []string

		for _, field := range scope.Fields() {
			if field.DBName != scope.PrimaryKey() && len(field.SqlTag) > 0 && !field.IsIgnored {
				columns = append(columns, scope.Quote(field.DBName))
				sqls = append(sqls, scope.AddToVars(field.Value))
			}
		}

		scope.Raw(fmt.Sprintf(
			"INSERT INTO %v (%v) VALUES (%v) %v",
			scope.TableName(),
			strings.Join(columns, ","),
			strings.Join(sqls, ","),
			scope.Dialect().ReturningStr(scope.PrimaryKey()),
		))

		// execute create sql
		var id interface{}
		if scope.Dialect().SupportLastInsertId() {
			if sql_result, err := scope.DB().Exec(scope.Sql, scope.SqlVars...); scope.Err(err) == nil {
				id, err = sql_result.LastInsertId()
				scope.Err(err)
			}
		} else {
			scope.Err(scope.DB().QueryRow(scope.Sql, scope.SqlVars...).Scan(&id))
		}

		if !scope.HasError() {
			scope.SetColumn(scope.PrimaryKey(), id)
		}
	}
}

func AfterCreate(scope *Scope) {
	scope.CallMethod("AfterCreate")
	scope.CallMethod("AfterSave")
}

func init() {
	DefaultCallback.Create().Register("gorm:begin_transaction", BeginTransaction)
	DefaultCallback.Create().Register("gorm:before_create", BeforeCreate)
	DefaultCallback.Create().Register("gorm:save_before_associations", SaveBeforeAssociations)
	DefaultCallback.Create().Register("gorm:update_time_stamp_when_create", UpdateTimeStampWhenCreate)
	DefaultCallback.Create().Register("gorm:create", Create)
	DefaultCallback.Create().Register("gorm:save_after_associations", SaveAfterAssociations)
	DefaultCallback.Create().Register("gorm:after_create", AfterCreate)
	DefaultCallback.Create().Register("gorm:commit_or_rollback_transaction", CommitOrRollbackTransaction)
}
