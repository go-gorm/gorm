package callbacks

import "github.com/jinzhu/gorm"

func Create(scope *gorm.Scope) {
}

func init() {
	gorm.DefaultCallback.Create().Before().Register(Create)
}

func init() {
	DefaultCallback.Create().Before("Delete").After("Lalala").Register("delete", Delete)
	DefaultCallback.Update().Before("Delete").After("Lalala").Remove("replace", Delete)
	DefaultCallback.Delete().Before("Delete").After("Lalala").Replace("replace", Delete)
	DefaultCallback.Query().Before("Delete").After("Lalala").Replace("replace", Delete)
}

// Scope
// HasError(), HasColumn(), CallMethod(), Raw(), Exec()
// TableName(), CombinedQuerySQL()
