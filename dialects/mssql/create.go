package mssql

import (
	"reflect"
	"sort"

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
		setIdentityInsert := false
		c := db.Statement.Clauses["ON CONFLICT"]
		onConflict, hasConflict := c.Expression.(clause.OnConflict)

		if field := db.Statement.Schema.PrioritizedPrimaryField; field != nil {
			setIdentityInsert = false
			switch db.Statement.ReflectValue.Kind() {
			case reflect.Struct:
				_, isZero := field.ValueOf(db.Statement.ReflectValue)
				setIdentityInsert = !isZero
			case reflect.Slice:
				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					_, isZero := field.ValueOf(db.Statement.ReflectValue.Index(i))
					setIdentityInsert = !isZero
					break
				}
			}

			if setIdentityInsert && (field.DataType == schema.Int || field.DataType == schema.Uint) {
				setIdentityInsert = true
				db.Statement.WriteString("SET IDENTITY_INSERT ")
				db.Statement.WriteQuoted(db.Statement.Table)
				db.Statement.WriteString(" ON;")
			} else {
				setIdentityInsert = false
			}
		}

		if hasConflict && len(db.Statement.Schema.PrimaryFields) > 0 {
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
					db.Statement.WriteString("DEFAULT VALUES;")
				}
			}
		}

		if setIdentityInsert {
			db.Statement.WriteString("SET IDENTITY_INSERT ")
			db.Statement.WriteQuoted(db.Statement.Table)
			db.Statement.WriteString(" OFF;")
		}
	}

	if !db.DryRun {
		rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

		if err == nil {
			defer rows.Close()

			if len(db.Statement.Schema.FieldsWithDefaultDBValue) > 0 {
				sortedKeys := []string{}
				for _, field := range db.Statement.Schema.FieldsWithDefaultDBValue {
					sortedKeys = append(sortedKeys, field.DBName)
				}
				sort.Strings(sortedKeys)

				returnningFields := make([]*schema.Field, len(sortedKeys))
				for idx, key := range sortedKeys {
					returnningFields[idx] = db.Statement.Schema.LookUpField(key)
				}

				values := make([]interface{}, len(returnningFields))

				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					for rows.Next() {
						for idx, field := range returnningFields {
							values[idx] = field.ReflectValueOf(db.Statement.ReflectValue.Index(int(db.RowsAffected))).Addr().Interface()
						}

						db.RowsAffected++
						db.AddError(rows.Scan(values...))
					}
				case reflect.Struct:
					for idx, field := range returnningFields {
						values[idx] = field.ReflectValueOf(db.Statement.ReflectValue).Addr().Interface()
					}

					if rows.Next() {
						db.RowsAffected++
						db.AddError(rows.Scan(values...))
					}
				}
			}
		} else {
			db.AddError(err)
		}
	}
}

func MergeCreate(db *gorm.DB, onConflict clause.OnConflict) {
	values := callbacks.ConvertToCreateValues(db.Statement)

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
}

func outputInserted(db *gorm.DB) {
	if len(db.Statement.Schema.FieldsWithDefaultDBValue) > 0 {
		sortedKeys := []string{}
		for _, field := range db.Statement.Schema.FieldsWithDefaultDBValue {
			sortedKeys = append(sortedKeys, field.DBName)
		}
		sort.Strings(sortedKeys)

		db.Statement.WriteString(" OUTPUT")
		for idx, key := range sortedKeys {
			if idx > 0 {
				db.Statement.WriteString(",")
			}
			db.Statement.WriteString(" INSERTED.")
			db.Statement.AddVar(db.Statement, clause.Column{Name: key})
		}
	}
}
