package gorm

import (
	"context"
	"database/sql"
	"fmt"
)

// Define callbacks for row query
func init() {
	DefaultCallback.RowQuery().RegisterContext("gorm:row_query", rowQueryCallback)
}

type RowQueryResult struct {
	Row *sql.Row
}

type RowsQueryResult struct {
	Rows  *sql.Rows
	Error error
}

// queryCallback used to query data from database
func rowQueryCallback(ctx context.Context, scope *Scope) {
	if result, ok := scope.InstanceGet("row_query_result"); ok {
		scope.prepareQuerySQL()

		if str, ok := scope.Get("gorm:query_hint"); ok {
			scope.SQL = fmt.Sprint(str) + scope.SQL
		}

		if str, ok := scope.Get("gorm:query_option"); ok {
			scope.SQL += addExtraSpaceIfExist(fmt.Sprint(str))
		}

		if rowResult, ok := result.(*RowQueryResult); ok {
			rowResult.Row = scope.SQLDB().QueryRowContext(ctx, scope.SQL, scope.SQLVars...)
		} else if rowsResult, ok := result.(*RowsQueryResult); ok {
			rowsResult.Rows, rowsResult.Error = scope.SQLDB().QueryContext(ctx, scope.SQL, scope.SQLVars...)
		}
	}
}
