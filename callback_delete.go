package gorm

import (
	"context"
	"errors"
	"fmt"
)

// Define callbacks for deleting
func init() {
	DefaultCallback.Delete().RegisterContext("gorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.Delete().RegisterContext("gorm:before_delete", beforeDeleteCallback)
	DefaultCallback.Delete().RegisterContext("gorm:delete", deleteCallback)
	DefaultCallback.Delete().RegisterContext("gorm:after_delete", afterDeleteCallback)
	DefaultCallback.Delete().RegisterContext("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// beforeDeleteCallback will invoke `BeforeDelete` method before deleting
func beforeDeleteCallback(_ctx context.Context, scope *Scope) {
	if scope.DB().HasBlockGlobalUpdate() && !scope.hasConditions() {
		scope.Err(errors.New("missing WHERE clause while deleting"))
		return
	}
	if !scope.HasError() {
		scope.CallMethod("BeforeDelete")
	}
}

// deleteCallback used to delete data from database or set deleted_at to current time (when using with soft delete)
func deleteCallback(ctx context.Context, scope *Scope) {
	if !scope.HasError() {
		var extraOption string
		if str, ok := scope.Get("gorm:delete_option"); ok {
			extraOption = fmt.Sprint(str)
		}

		deletedAtField, hasDeletedAtField := scope.FieldByName("DeletedAt")

		if !scope.Search.Unscoped && hasDeletedAtField {
			scope.Raw(fmt.Sprintf(
				"UPDATE %v SET %v=%v%v%v",
				scope.QuotedTableName(),
				scope.Quote(deletedAtField.DBName),
				scope.AddToVars(scope.db.nowFunc()),
				addExtraSpaceIfExist(scope.CombinedConditionSql()),
				addExtraSpaceIfExist(extraOption),
			)).Exec(ctx)
		} else {
			scope.Raw(fmt.Sprintf(
				"DELETE FROM %v%v%v",
				scope.QuotedTableName(),
				addExtraSpaceIfExist(scope.CombinedConditionSql()),
				addExtraSpaceIfExist(extraOption),
			)).Exec(ctx)
		}
	}
}

// afterDeleteCallback will invoke `AfterDelete` method after deleting
func afterDeleteCallback(_ctx context.Context, scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterDelete")
	}
}
