package tests

import (
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DummyDialector is a SQL-rendering test double for gorm.Dialector. It registers
// default callbacks and provides quoting and placeholders, but does not open a
// connection or support migrations. Use DryRun for isolated SQL-construction tests;
// use a real dialector for database execution and compatibility tests.
type DummyDialector struct {
	// TranslatedErr is returned by Translate regardless of its input, including nil when no
	// translated error is configured.
	TranslatedErr error
}

// Name returns the fixed test dialect name "dummy".
func (DummyDialector) Name() string {
	return "dummy"
}

// Initialize registers default callbacks with RETURNING enabled for writes and
// reversed last-insert-ID handling. It does not configure a connection pool and
// always returns nil.
func (DummyDialector) Initialize(db *gorm.DB) error {
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{
		CreateClauses:        []string{"INSERT", "VALUES", "ON CONFLICT", "RETURNING"},
		UpdateClauses:        []string{"UPDATE", "SET", "WHERE", "RETURNING"},
		DeleteClauses:        []string{"DELETE", "FROM", "WHERE", "RETURNING"},
		LastInsertIDReversed: true,
	})

	return nil
}

// DefaultValueOf returns the literal DEFAULT expression without inspecting the field.
func (DummyDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

// Migrator returns nil because this test double does not implement schema migration. Do
// not invoke migration operations through it.
func (DummyDialector) Migrator(*gorm.DB) gorm.Migrator {
	return nil
}

// BindVarTo writes a question-mark placeholder. Statement variable collection is the
// caller's responsibility; stmt and v do not alter rendering.
func (DummyDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

// QuoteTo renders backtick-delimited identifiers, handles dotted components and
// existing backticks, and writes to writer. It is a deterministic test renderer, not
// proof of another database's quoting rules.
func (DummyDialector) QuoteTo(writer clause.Writer, str string) {
	var (
		underQuoted, selfQuoted bool
		continuousBacktick      int8
		shiftDelimiter          int8
	)

	for _, v := range []byte(str) {
		switch v {
		case '`':
			continuousBacktick++
			if continuousBacktick == 2 {
				writer.WriteString("``")
				continuousBacktick = 0
			}
		case '.':
			if continuousBacktick > 0 || !selfQuoted {
				shiftDelimiter = 0
				underQuoted = false
				continuousBacktick = 0
				writer.WriteByte('`')
			}
			writer.WriteByte(v)
			continue
		default:
			if shiftDelimiter-continuousBacktick <= 0 && !underQuoted {
				writer.WriteByte('`')
				underQuoted = true
				if selfQuoted = continuousBacktick > 0; selfQuoted {
					continuousBacktick -= 1
				}
			}

			for ; continuousBacktick > 0; continuousBacktick -= 1 {
				writer.WriteString("``")
			}

			writer.WriteByte(v)
		}
		shiftDelimiter++
	}

	if continuousBacktick > 0 && !selfQuoted {
		writer.WriteString("``")
	}
	writer.WriteByte('`')
}

// Explain formats SQL with substituted diagnostic values through logger.ExplainSQL
// using double-quote string delimiters. The result is for assertions or logs, not
// execution.
func (DummyDialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

// DataTypeOf returns an empty type name because this test double has no database type
// mapping.
func (DummyDialector) DataTypeOf(*schema.Field) string {
	return ""
}

// Translate returns TranslatedErr and ignores err. Configure TranslatedErr explicitly
// when testing GORM error translation.
func (d DummyDialector) Translate(err error) error {
	return d.TranslatedErr
}
