package tests

import (
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/logger"
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

func (DummyDialector) QuoteTo(builder *strings.Builder, str string) {
	builder.WriteByte('`')
	builder.WriteString(str)
	builder.WriteByte('`')
}

func (DummyDialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

func (DummyDialector) DataTypeOf(*schema.Field) string {
	return ""
}
