package mssql

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/callbacks"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
)

func Create(db *gorm.DB) {
	if db.Statement.Schema != nil && !db.Statement.Unscoped {
		for _, c := range db.Statement.Schema.CreateClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.String() == "" {
		c := db.Statement.Clauses["ON CONFLICT"]
		onConflict, hasConflict := c.Expression.(clause.OnConflict)

		if hasConflict {
			MergeCreate(db, onConflict)
		} else {
			db.Statement.AddClauseIfNotExists(clause.Insert{Table: clause.Table{Name: db.Statement.Table}})
			db.Statement.Build("INSERT")
			db.Statement.WriteByte(' ')

			db.Statement.AddClause(callbacks.ConvertToCreateValues(db.Statement))
			if values, ok := db.Statement.Clauses["VALUES"].Expression.(clause.Values); ok {
				if len(values.Columns) > 0 {
					db.Statement.WriteByte('(')
					for idx, column := range values.Columns {
						if idx > 0 {
							db.Statement.WriteByte(',')
						}
						db.Statement.WriteQuoted(column)
					}
					db.Statement.WriteByte(')')

					outputInserted(db)

					db.Statement.WriteString(" VALUES ")

					for idx, value := range values.Values {
						if idx > 0 {
							db.Statement.WriteByte(',')
						}

						db.Statement.WriteByte('(')
						db.Statement.AddVar(db.Statement, value...)
						db.Statement.WriteByte(')')
					}

					db.Statement.WriteString(";")
				} else {
					db.Statement.WriteString("DEFAULT VALUES")
				}
			}
		}
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
				db.RowsAffected++
				db.AddError(rows.Scan(values))
			}
		}
	} else {
		db.AddError(err)
	}
}

func MergeCreate(db *gorm.DB, onConflict clause.OnConflict) {
	values := callbacks.ConvertToCreateValues(db.Statement)
	setIdentityInsert := false

	if field := db.Statement.Schema.PrioritizedPrimaryField; field != nil {
		if field.DataType == schema.Int || field.DataType == schema.Uint {
			setIdentityInsert = true
			db.Statement.WriteString("SET IDENTITY_INSERT ")
			db.Statement.WriteQuoted(db.Statement.Table)
			db.Statement.WriteString("ON;")
		}
	}

	db.Statement.WriteString("MERGE INTO ")
	db.Statement.WriteQuoted(db.Statement.Table)
	db.Statement.WriteString(" USING (VALUES")
	for idx, value := range values.Values {
		if idx > 0 {
			db.Statement.WriteByte(',')
		}

		db.Statement.WriteByte('(')
		db.Statement.AddVar(db.Statement, value...)
		db.Statement.WriteByte(')')
	}

	db.Statement.WriteString(") AS source (")
	for idx, column := range values.Columns {
		if idx > 0 {
			db.Statement.WriteByte(',')
		}
		db.Statement.WriteQuoted(column.Name)
	}
	db.Statement.WriteString(") ON ")

	var where clause.Where
	for _, field := range db.Statement.Schema.PrimaryFields {
		where.Exprs = append(where.Exprs, clause.Eq{
			Column: clause.Column{Table: db.Statement.Table, Name: field.DBName},
			Value:  clause.Column{Table: "source", Name: field.DBName},
		})
	}
	where.Build(db.Statement)

	if len(onConflict.DoUpdates) > 0 {
		db.Statement.WriteString(" WHEN MATCHED THEN UPDATE SET ")
		onConflict.DoUpdates.Build(db.Statement)
	}

	db.Statement.WriteString(" WHEN NOT MATCHED THEN INSERT (")

	for idx, column := range values.Columns {
		if idx > 0 {
			db.Statement.WriteByte(',')
		}
		db.Statement.WriteQuoted(column.Name)
	}

	db.Statement.WriteString(") VALUES (")

	for idx, column := range values.Columns {
		if idx > 0 {
			db.Statement.WriteByte(',')
		}
		db.Statement.WriteQuoted(clause.Column{
			Table: "source",
			Name:  column.Name,
		})
	}

	db.Statement.WriteString(")")
	outputInserted(db)
	db.Statement.WriteString(";")

	if setIdentityInsert {
		db.Statement.WriteString("SET IDENTITY_INSERT ")
		db.Statement.WriteQuoted(db.Statement.Table)
		db.Statement.WriteString("OFF;")
	}
}

func outputInserted(db *gorm.DB) {
	if db.Statement.Schema.PrioritizedPrimaryField != nil {
		db.Statement.WriteString(" OUTPUT INSERTED.")
		db.Statement.WriteQuoted(db.Statement.Schema.PrioritizedPrimaryField.DBName)
	}
}
