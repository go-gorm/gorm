package gorm

import (
	"fmt"
	"strings"
)

func BeforeCreate(scope *Scope) {
	scope.CallMethodWithErrorCheck("BeforeSave")
	scope.CallMethodWithErrorCheck("BeforeCreate")
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
		fields := scope.Fields()
		for _, field := range fields {
			if scope.changeableField(field) {
				if field.IsNormal {
					if !field.IsPrimaryKey || (field.IsPrimaryKey && !field.IsBlank) {
						if !field.IsBlank || !field.HasDefaultValue {
							columns = append(columns, scope.Quote(field.DBName))
							sqls = append(sqls, scope.AddToVars(field.Field.Interface()))
						}
					}
				} else if relationship := field.Relationship; relationship != nil && relationship.Kind == "belongs_to" {
					for _, dbName := range relationship.ForeignDBNames {
						if relationField := fields[dbName]; !scope.changeableField(relationField) {
							columns = append(columns, scope.Quote(relationField.DBName))
							sqls = append(sqls, scope.AddToVars(relationField.Field.Interface()))
						}
					}
				}
			}
		}

		returningKey := "*"
		primaryField := scope.PrimaryField()
		if primaryField != nil {
			returningKey = scope.Quote(primaryField.DBName)
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
		if scope.Dialect().SupportLastInsertId() {
			if result, err := scope.SqlDB().Exec(scope.Sql, scope.SqlVars...); scope.Err(err) == nil {
				id, err := result.LastInsertId()
				if scope.Err(err) == nil {
					scope.db.RowsAffected, _ = result.RowsAffected()
					if primaryField != nil && primaryField.IsBlank {
						scope.Err(scope.SetColumn(primaryField, id))
					}
				}
			}
		} else {
			if primaryField == nil {
				if results, err := scope.SqlDB().Exec(scope.Sql, scope.SqlVars...); err == nil {
					scope.db.RowsAffected, _ = results.RowsAffected()
				} else {
					scope.Err(err)
				}
			} else {
				if err := scope.Err(scope.SqlDB().QueryRow(scope.Sql, scope.SqlVars...).Scan(primaryField.Field.Addr().Interface())); err == nil {
					scope.db.RowsAffected = 1
				} else {
					scope.Err(err)
				}
			}
		}
	}
}

func AfterCreate(scope *Scope) {
	scope.CallMethodWithErrorCheck("AfterCreate")
	scope.CallMethodWithErrorCheck("AfterSave")
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
