package mssql

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
	"github.com/jinzhu/gorm/clause"
)

func Create(db *gorm.DB) {
	if db.Statement.Schema != nil && !db.Statement.Unscoped {
		for _, c := range db.Statement.Schema.CreateClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.String() == "" {
		db.Statement.AddClauseIfNotExists(clause.Insert{
			Table: clause.Table{Name: db.Statement.Table},
		})
		db.Statement.AddClause(callbacks.ConvertToCreateValues(db.Statement))

		db.Statement.Build("INSERT")
		db.Statement.WriteByte(' ')

		c := db.Statement.Clauses["VALUES"]
		if values, ok := c.Expression.(clause.Values); ok {
			if len(values.Columns) > 0 {
				db.Statement.WriteByte('(')
				for idx, column := range values.Columns {
					if idx > 0 {
						db.Statement.WriteByte(',')
					}
					db.Statement.WriteQuoted(column)
				}
				db.Statement.WriteByte(')')

				if db.Statement.Schema.PrioritizedPrimaryField != nil {
					db.Statement.WriteString(" OUTPUT INSERTED.")
					db.Statement.WriteQuoted(db.Statement.Schema.PrioritizedPrimaryField.DBName)
				}

				db.Statement.WriteString(" VALUES ")

				for idx, value := range values.Values {
					if idx > 0 {
						db.Statement.WriteByte(',')
					}

					db.Statement.WriteByte('(')
					db.Statement.AddVar(db.Statement, value...)
					db.Statement.WriteByte(')')
				}
			} else {
				db.Statement.WriteString("DEFAULT VALUES")
			}
		}

		db.Statement.WriteByte(' ')
		db.Statement.Build("ON CONFLICT")
	}

	rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

	if err == nil {
		defer rows.Close()

		switch db.Statement.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			for rows.Next() {
				// for idx, field := range fields {
				// 	values[idx] = field.ReflectValueOf(db.Statement.ReflectValue.Index(int(db.RowsAffected))).Addr().Interface()
				// }

				values := db.Statement.Schema.PrioritizedPrimaryField.ReflectValueOf(db.Statement.ReflectValue.Index(int(db.RowsAffected))).Addr().Interface()
				if err := rows.Scan(values); err != nil {
					db.AddError(err)
				}
				db.RowsAffected++
			}
		case reflect.Struct:
			// for idx, field := range fields {
			// 	values[idx] = field.ReflectValueOf(db.Statement.ReflectValue).Addr().Interface()
			// }
			values := db.Statement.Schema.PrioritizedPrimaryField.ReflectValueOf(db.Statement.ReflectValue).Addr().Interface()

			if rows.Next() {
				err = rows.Scan(values)
			}
		}
	} else {
		db.AddError(err)
	}
}
