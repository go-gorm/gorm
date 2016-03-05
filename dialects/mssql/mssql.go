package mssql

import (
	"fmt"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jinzhu/gorm"
)

func setIdentityInsert(scope *gorm.Scope) {
	if scope.Dialect().GetName() == "mssql" {
		scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v ON", scope.TableName()))
	}
}

func init() {
	gorm.DefaultCallback.Create().After("gorm:begin_transaction").Register("mssql:set_identity_insert", setIdentityInsert)
}
