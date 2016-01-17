package gorm

import (
	"fmt"
	"strings"
)

func beforeCreateCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("BeforeSave")
	}
	if !scope.HasError() {
		scope.CallMethod("BeforeCreate")
	}
}

func updateTimeStampForCreateCallback(scope *Scope) {
	if !scope.HasError() {
		now := NowFunc()
		scope.SetColumn("CreatedAt", now)
		scope.SetColumn("UpdatedAt", now)
	}
}

func createCallback(scope *Scope) {
	defer scope.trace(NowFunc())

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
						} else if field.HasDefaultValue {
							var hasDefaultValueColumns []string
							if oldHasDefaultValueColumns, ok := scope.InstanceGet("gorm:force_reload_after_create_attrs"); ok {
								hasDefaultValueColumns = oldHasDefaultValueColumns.([]string)
							}
							hasDefaultValueColumns = append(hasDefaultValueColumns, field.DBName)
							scope.InstanceSet("gorm:force_reload_after_create_attrs", hasDefaultValueColumns)
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
				scope.Dialect().ReturningStr(scope.QuotedTableName(), returningKey),
			))
		} else {
			scope.Raw(fmt.Sprintf(
				"INSERT INTO %v (%v) VALUES (%v) %v",
				scope.QuotedTableName(),
				strings.Join(columns, ","),
				strings.Join(sqls, ","),
				scope.Dialect().ReturningStr(scope.QuotedTableName(), returningKey),
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

func forceReloadAfterCreateCallback(scope *Scope) {
	if columns, ok := scope.InstanceGet("gorm:force_reload_after_create_attrs"); ok {
		scope.DB().New().Select(columns.([]string)).First(scope.Value)
	}
}

func afterCreateCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterCreate")
	}
	if !scope.HasError() {
		scope.CallMethod("AfterSave")
	}
}

func init() {
	defaultCallback.Create().Register("gorm:begin_transaction", beginTransactionCallback)
	defaultCallback.Create().Register("gorm:before_create", beforeCreateCallback)
	defaultCallback.Create().Register("gorm:save_before_associations", saveBeforeAssociationsCallback)
	defaultCallback.Create().Register("gorm:update_time_stamp_when_create", updateTimeStampForCreateCallback)
	defaultCallback.Create().Register("gorm:create", createCallback)
	defaultCallback.Create().Register("gorm:force_reload_after_create", forceReloadAfterCreateCallback)
	defaultCallback.Create().Register("gorm:save_after_associations", saveAfterAssociationsCallback)
	defaultCallback.Create().Register("gorm:after_create", afterCreateCallback)
	defaultCallback.Create().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}
