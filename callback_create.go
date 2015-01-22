package gorm

import (
	"fmt"
	"reflect"
	"strings"
)

func BeforeCreate(scope *Scope) {
	scope.CallMethod("BeforeSave")
	scope.CallMethod("BeforeCreate")
}

func UpdateTimeStampWhenCreate(scope *Scope) {
	if !scope.HasError() {
		now := NowFunc()
		scope.SetColumn("CreatedAt", now)
		scope.SetColumn("UpdatedAt", now)
	}
}

func Create(scope *Scope) {
	defer scope.Trace(NowFunc())

	if !scope.HasError() {
		// set create sql
		var sqls, columns []string
		for _, field := range scope.Fields() {
			if field.IsNormal && (!field.IsPrimaryKey || !scope.PrimaryKeyZero()) {
				if field.DefaultValue != nil && field.IsBlank {
					continue
				}
				columns = append(columns, scope.Quote(field.DBName))
				val := field.Field
				if val.Kind() == reflect.Struct {
					val = val.Addr()
				}
				sqls = append(sqls, scope.AddToVars(val.Interface()))
			}
		}

		returningKey := "*"
		if scope.PrimaryKey() != "" {
			returningKey = scope.Quote(scope.PrimaryKey())
		}

		if len(columns) == 0 {
			scope.Raw(fmt.Sprintf("INSERT INTO %v DEFAULT VALUES %v",
				scope.QuotedTableName(),
				scope.Dialect().ReturningStr(scope.TableName(), returningKey),
			))
		} else {
			scope.Raw(fmt.Sprintf(
				"INSERT INTO %v (%v) VALUES (%v) %v",
				scope.QuotedTableName(),
				strings.Join(columns, ","),
				strings.Join(sqls, ","),
				scope.Dialect().ReturningStr(scope.TableName(), returningKey),
			))
		}

		// execute create sql
		var id interface{}
		if scope.Dialect().SupportLastInsertId() {
			if result, err := scope.DB().Exec(scope.Sql, scope.SqlVars...); scope.Err(err) == nil {
				id, err = result.LastInsertId()
				if scope.Err(err) == nil {
					scope.db.RowsAffected, _ = result.RowsAffected()
				}
			}
		} else {
			if scope.PrimaryKey() == "" {
				if results, err := scope.DB().Exec(scope.Sql, scope.SqlVars...); err != nil {
					scope.db.RowsAffected, _ = results.RowsAffected()
				}
			} else {
				if scope.Err(scope.DB().QueryRow(scope.Sql, scope.SqlVars...).Scan(&id)) == nil {
					scope.db.RowsAffected = 1
				}
			}
		}

		if scope.PrimaryKey() != "" && !scope.HasError() && scope.PrimaryKeyZero() {
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
