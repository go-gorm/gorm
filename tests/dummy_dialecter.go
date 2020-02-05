package tests

import (
	"github.com/jinzhu/gorm"
)

type DummyDialector struct {
}

func (DummyDialector) Initialize(*gorm.DB) error {
	return nil
}

func (DummyDialector) Migrator() gorm.Migrator {
	return nil
}

func (DummyDialector) BindVar(stmt *gorm.Statement, v interface{}) string {
	return "?"
}

func (DummyDialector) QuoteChars() [2]byte {
	return [2]byte{'`', '`'} // `name`
}
