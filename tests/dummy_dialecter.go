package tests

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
)

type DummyDialector struct {
}

func (DummyDialector) Initialize(*gorm.DB) error {
	return nil
}

func (DummyDialector) Migrator(*gorm.DB) gorm.Migrator {
	return nil
}

func (DummyDialector) BindVar(stmt *gorm.Statement, v interface{}) string {
	return "?"
}

func (DummyDialector) QuoteChars() [2]byte {
	return [2]byte{'`', '`'} // `name`
}

func (DummyDialector) DataTypeOf(*schema.Field) string {
	return ""
}
