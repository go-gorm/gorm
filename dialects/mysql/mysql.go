package mysql

import (
	"database/sql"
	"fmt"
	"math"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Dialector struct {
	DSN string
}

func Open(dsn string) gorm.Dialector {
	return &Dialector{DSN: dsn}
}

func (dialector Dialector) Name() string {
	return "mysql"
}

func (dialector Dialector) Initialize(db *gorm.DB) (err error) {
	// register callbacks
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	db.ConnPool, err = sql.Open("mysql", dialector.DSN)

	for k, v := range dialector.ClauseBuilders() {
		db.ClauseBuilders[k] = v
	}
	return
}

func (dialector Dialector) ClauseBuilders() map[string]clause.ClauseBuilder {
	return map[string]clause.ClauseBuilder{
		"ON CONFLICT": func(c clause.Clause, builder clause.Builder) {
			if onConflict, ok := c.Expression.(clause.OnConflict); ok {
				builder.WriteString("ON DUPLICATE KEY UPDATE ")
				if len(onConflict.DoUpdates) == 0 {
					if s := builder.(*gorm.Statement).Schema; s != nil {
						var column clause.Column
						onConflict.DoNothing = false

						if s.PrioritizedPrimaryField != nil {
							column = clause.Column{Name: s.PrioritizedPrimaryField.DBName}
						} else {
							for _, field := range s.FieldsByDBName {
								column = clause.Column{Name: field.DBName}
								break
							}
						}
						onConflict.DoUpdates = []clause.Assignment{{Column: column, Value: column}}
					}
				}

				onConflict.DoUpdates.Build(builder)
			} else {
				c.Build(builder)
			}
		},
		"VALUES": func(c clause.Clause, builder clause.Builder) {
			if values, ok := c.Expression.(clause.Values); ok && len(values.Columns) == 0 {
				builder.WriteString("VALUES()")
				return
			}
			c.Build(builder)
		},
	}
}

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{
		DB:        db,
		Dialector: dialector,
	}}}
}

func (dialector Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (dialector Dialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	writer.WriteString(str)
	writer.WriteByte('`')
}

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		sqlType := "int"
		switch {
		case field.Size <= 8:
			sqlType = "tinyint"
		case field.Size <= 16:
			sqlType = "smallint"
		case field.Size <= 32:
			sqlType = "int"
		default:
			sqlType = "bigint"
		}

		if field.DataType == schema.Uint {
			sqlType += " unsigned"
		}

		if field.AutoIncrement || field == field.Schema.PrioritizedPrimaryField {
			sqlType += " AUTO_INCREMENT"
		}
		return sqlType
	case schema.Float:
		if field.Size <= 32 {
			return "float"
		}
		return "double"
	case schema.String:
		size := field.Size
		if size == 0 {
			if field.PrimaryKey || field.HasDefaultValue {
				size = 256
			}
		}

		if size >= 65536 && size <= int(math.Pow(2, 24)) {
			return "mediumtext"
		} else if size > int(math.Pow(2, 24)) || size <= 0 {
			return "longtext"
		}
		return fmt.Sprintf("varchar(%d)", size)
	case schema.Time:
		precision := ""
		if field.Precision == 0 {
			field.Precision = 3
		}

		if field.Precision > 0 {
			precision = fmt.Sprintf("(%d)", field.Precision)
		}

		if field.NotNull || field.PrimaryKey {
			return "datetime" + precision
		}
		return "datetime" + precision + " NULL"
	case schema.Bytes:
		if field.Size > 0 && field.Size < 65536 {
			return fmt.Sprintf("varbinary(%d)", field.Size)
		}

		if field.Size >= 65536 && field.Size <= int(math.Pow(2, 24)) {
			return "mediumblob"
		}

		return "longblob"
	}

	return ""
}
