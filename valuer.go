package gorm

import (
	"context"
	"reflect"

	"gorm.io/gorm/clause"
)

var dynamicRegisteredGormValuer = make(map[reflect.Type]func(context.Context, *DB, any) clause.Expr)

// RegisterDynamicGormValuerInit shouldn't be called outside the init function
func RegisterDynamicGormValuerInit(valueType reflect.Type, valuer func(context.Context, *DB, any) clause.Expr) {
	dynamicRegisteredGormValuer[valueType] = valuer
}

func GetDynamicGormValuer(valueType reflect.Type) func(context.Context, *DB, any) clause.Expr {
	return dynamicRegisteredGormValuer[valueType]
}
