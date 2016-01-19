package gorm

import (
	"fmt"
	"strings"
)

// Define callbacks for creating
func init() {
	defaultCallback.Create().Register("gorm:begin_transaction", beginTransactionCallback)
	defaultCallback.Create().Register("gorm:before_create", beforeCreateCallback)
	defaultCallback.Create().Register("gorm:save_before_associations", saveBeforeAssociationsCallback)
	defaultCallback.Create().Register("gorm:update_time_stamp", updateTimeStampForCreateCallback)
	defaultCallback.Create().Register("gorm:create", createCallback)
	defaultCallback.Create().Register("gorm:force_reload_after_create", forceReloadAfterCreateCallback)
	defaultCallback.Create().Register("gorm:save_after_associations", saveAfterAssociationsCallback)
	defaultCallback.Create().Register("gorm:after_create", afterCreateCallback)
	defaultCallback.Create().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// beforeCreateCallback will invoke `BeforeSave`, `BeforeCreate` method before creating
func beforeCreateCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("BeforeSave")
	}
	if !scope.HasError() {
		scope.CallMethod("BeforeCreate")
	}
}

// updateTimeStampForCreateCallback will set `CreatedAt`, `UpdatedAt` when creating
func updateTimeStampForCreateCallback(scope *Scope) {
	if !scope.HasError() {
		now := NowFunc()
		scope.SetColumn("CreatedAt", now)
		scope.SetColumn("UpdatedAt", now)
	}
}

// createCallback the callback used to insert data into database
func createCallback(scope *Scope) {
	if !scope.HasError() {
		defer scope.trace(NowFunc())

		var (
			columns, placeholders        []string
			blankColumnsWithDefaultValue []string
			fields                       = scope.Fields()
		)

		for _, field := range fields {
			if scope.changeableField(field) {
				if field.IsNormal {
					if !field.IsPrimaryKey || !field.IsBlank {
						if field.IsBlank && field.HasDefaultValue {
							blankColumnsWithDefaultValue = append(blankColumnsWithDefaultValue, field.DBName)
							scope.InstanceSet("gorm:blank_columns_with_default_value", blankColumnsWithDefaultValue)
						} else {
							columns = append(columns, scope.Quote(field.DBName))
							placeholders = append(placeholders, scope.AddToVars(field.Field.Interface()))
						}
					}
				} else if field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
					for _, foreignKey := range field.Relationship.ForeignDBNames {
						if foreignField := fields[foreignKey]; !scope.changeableField(foreignField) {
							columns = append(columns, scope.Quote(foreignField.DBName))
							placeholders = append(placeholders, scope.AddToVars(foreignField.Field.Interface()))
						}
					}
				}
			}
		}

		var (
			returningColumn = "*"
			quotedTableName = scope.QuotedTableName()
			primaryField    = scope.PrimaryField()
		)

		if primaryField != nil {
			returningColumn = scope.Quote(primaryField.DBName)
		}

		lastInsertIdReturningSuffix := scope.Dialect().LastInsertIdReturningSuffix(quotedTableName, returningColumn)

		if len(columns) == 0 {
			scope.Raw(fmt.Sprintf("INSERT INTO %v DEFAULT VALUES %v", quotedTableName, lastInsertIdReturningSuffix))
		} else {
			scope.Raw(fmt.Sprintf(
				"INSERT INTO %v (%v) VALUES (%v) %v",
				scope.QuotedTableName(),
				strings.Join(columns, ","),
				strings.Join(placeholders, ","),
				lastInsertIdReturningSuffix,
			))
		}

		// execute create sql
		if lastInsertIdReturningSuffix == "" || primaryField == nil {
			if result, err := scope.SqlDB().Exec(scope.Sql, scope.SqlVars...); scope.Err(err) == nil {
				// set rows affected count
				scope.db.RowsAffected, _ = result.RowsAffected()

				// set primary value to primary field
				if primaryField != nil && primaryField.IsBlank {
					if primaryValue, err := result.LastInsertId(); scope.Err(err) == nil {
						scope.Err(primaryField.Set(primaryValue))
					}
				}
			}
		} else {
			if err := scope.SqlDB().QueryRow(scope.Sql, scope.SqlVars...).Scan(primaryField.Field.Addr().Interface()); scope.Err(err) == nil {
				scope.db.RowsAffected = 1
			}
		}
	}
}

// forceReloadAfterCreateCallback will reload columns that having default value, and set it back to current object
func forceReloadAfterCreateCallback(scope *Scope) {
	if blankColumnsWithDefaultValue, ok := scope.InstanceGet("gorm:blank_columns_with_default_value"); ok {
		scope.DB().New().Select(blankColumnsWithDefaultValue.([]string)).First(scope.Value)
	}
}

// afterCreateCallback will invoke `AfterCreate`, `AfterSave` method after creating
func afterCreateCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterCreate")
	}
	if !scope.HasError() {
		scope.CallMethod("AfterSave")
	}
}
